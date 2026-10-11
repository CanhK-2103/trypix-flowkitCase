import React, { lazy, Suspense, useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  active,
  allPages,
  api,
  ApiError,
  pollInterval,
  retryGet,
  type Finding,
  type Page,
} from "./api";
import type { Run, Workflow } from "./types";
import { Money, ActualMoney, QuotePanel, RunCosts } from "./Costs";
import "./style.css";
const JsonEditor = lazy(() =>
  import("./JsonEditor").then((module) => ({ default: module.JsonEditor })),
);
const client = new QueryClient({
  defaultOptions: {
    queries: { retry: retryGet, retryDelay: 1000, refetchOnWindowFocus: false },
    mutations: { retry: false },
  },
});
const empty: Workflow = {
  name: "Yeni workflow",
  description: "",
  definition: {
    schemaVersion: 1,
    nodes: [
      {
        id: "input",
        kind: "input",
        params: {
          inputSchema: {
            type: "object",
            properties: {
              description: { type: "string", minLength: 1, maxLength: 2000 },
            },
            required: ["description"],
            additionalProperties: false,
          },
          textField: "description",
        },
      },
      { id: "output", kind: "output", params: {} },
    ],
    edges: [{ from: "input", fromPort: "text", to: "output", toPort: "text" }],
  },
};
const format = (v: unknown) => JSON.stringify(v, null, 2);
const short = (id: string) => id.slice(0, 8);
function Badge({ status }: { status: string }) {
  return (
    <span className={`badge ${status}`}>
      {(
        {
          queued: "Kuyrukta",
          running: "Çalışıyor",
          succeeded: "Başarılı",
          failed: "Başarısız",
          pending: "Bekliyor",
          waiting_provider: "Sağlayıcı bekleniyor",
          retry_wait: "Tekrar bekleniyor",
          skipped: "Atlandı",
        } as Record<string, string>
      )[status] ?? status}
    </span>
  );
}
function ErrorBox({ error }: { error: Error | null | undefined }) {
  return error ? (
    <div className="notice error" role="alert">
      <strong>{error instanceof ApiError ? error.code : "Hata"}</strong> ·{" "}
      {error.message}
    </div>
  ) : null;
}
function App() {
  const qc = useQueryClient();
  const session = useQuery({
    queryKey: ["session"],
    queryFn: async () => {
      try {
        const response = await fetch("/auth/session");
        const value = await response.json() as { authenticated?: boolean };
        return response.ok && value.authenticated === true;
      } catch { return false; } // Local Compose has no production gate.
    },
    retry: false,
    staleTime: Infinity,
  });
  const workflows = useQuery({
    queryKey: ["workflows"],
    queryFn: () => allPages<Workflow>("/workflows"),
  });
  const [selected, setSelected] = useState("");
  const [tab, setTab] = useState<"editor" | "run">("run");
  const [draft, setDraft] = useState(format(empty));
  const [dirty, setDirty] = useState(false);
  const loaded = useRef("");
  const [findings, setFindings] = useState<Finding[] | null>(null);
  const [validated, setValidated] = useState(false);
  const [values, setValues] = useState<Record<string, unknown>>({
    description: "",
  });
  const [runID, setRunID] = useState("");
  const replay = useRef<{ body: string; key: string } | null>(null);
  const selectedWorkflow = workflows.data?.find((w) => w.id === selected);
  useEffect(() => {
    if (!selected && workflows.data?.length && loaded.current !== "new") {
      setSelected(workflows.data[0].id!);
    }
  }, [workflows.data, selected]);
  useEffect(() => {
    if (selectedWorkflow && loaded.current !== selected) {
      loaded.current = selected;
      setDraft(format(selectedWorkflow));
      setDirty(false);
      setFindings(null);
      setValidated(false);
      setValues({ description: "" });
    }
  }, [selected, selectedWorkflow]);
  let parsed: Workflow | undefined;
  let syntaxError = "";
  try {
    parsed = JSON.parse(draft) as Workflow;
  } catch (e) {
    syntaxError = (e as Error).message;
  }
  const save = useMutation({
    mutationFn: () =>
      api<Workflow>(selected ? `/workflows/${selected}` : "/workflows", {
        method: selected ? "PUT" : "POST",
        body: draft,
      }),
    onSuccess: (w) => {
      setSelected(w.id!);
      loaded.current = w.id!;
      setDraft(format(w));
      setDirty(false);
      setValidated(true);
      setFindings([]);
      void qc.invalidateQueries({ queryKey: ["workflows"] });
    },
    onError: (e) => {
      if (e instanceof ApiError) setFindings(e.findings);
    },
  });
  const validate = useMutation({
    mutationFn: () =>
      api<{ valid: boolean; findings: Finding[] }>("/workflows/validate", {
        method: "POST",
        body: draft,
      }),
    onSuccess: (result) => {
      setFindings(result.findings);
      setValidated(result.valid);
    },
  });
  const remove = useMutation({
    mutationFn: () => api(`/workflows/${selected}`, { method: "DELETE" }),
    onSuccess: () => {
      setSelected("");
      loaded.current = "";
      void qc.invalidateQueries({ queryKey: ["workflows"] });
    },
  });
  const run = useMutation({
    mutationFn: () => {
      const body = JSON.stringify({ input: values });
      const fingerprint = selected + "|" + body;
      if (!replay.current || replay.current.body !== fingerprint)
        replay.current = { body: fingerprint, key: crypto.randomUUID() };
      return api<{ run_id: string }>(`/workflows/${selected}/runs`, {
        method: "POST",
        body,
        headers: { "Idempotency-Key": replay.current.key },
      });
    },
    onSuccess: (result) => {
      setRunID(result.run_id);
      replay.current = null;
      void qc.invalidateQueries({ queryKey: ["runs"] });
    },
  });
  const detail = useQuery({
    queryKey: ["run", runID],
    queryFn: () => api<Run>(`/runs/${runID}`),
    enabled: !!runID,
    refetchInterval: (query) =>
      pollInterval(
        query.state.data?.status,
        query.state.data?.cost.actual_cost_status,
      ),
  });
  const history = useQuery({
    queryKey: ["runs", selected],
    queryFn: () =>
      api<Page<Run>>(
        `/runs?limit=25${selected ? `&workflow_id=${selected}` : ""}`,
      ),
  });
  const terminal = useRef("");
  useEffect(() => {
    if (
      detail.data &&
      !active(detail.data.status) &&
      terminal.current !== detail.data.run_id
    ) {
      terminal.current = detail.data.run_id;
      void qc.invalidateQueries({ queryKey: ["runs"] });
    }
  }, [detail.data, qc]);
  const inputNode = selectedWorkflow?.definition.nodes.find(
    (n) => n.kind === "input",
  );
  const schema = inputNode?.params?.inputSchema as
    | {
        properties?: Record<
          string,
          {
            type: string;
            minLength?: number;
            maxLength?: number;
            minimum?: number;
            maximum?: number;
          }
        >;
        required?: string[];
      }
    | undefined;
  function selectWorkflow(id: string) {
    setSelected(id);
    setRunID("");
    setTab("run");
    loaded.current = "";
    replay.current = null;
  }
  function newWorkflow() {
    setSelected("");
    loaded.current = "new";
    setDraft(format(empty));
    setDirty(true);
    setFindings(null);
    setValidated(false);
    setTab("editor");
  }
  function edit(value: string) {
    setDraft(value);
    setDirty(true);
    setValidated(false);
    setFindings(null);
  }
  return (
    <div className="shell">
      <aside>
        <a className="brand" href="/">
          <span className="brand-mark">ƒ</span> FlowKit
          <span className="brand-tag">STUDIO</span>
        </a>
        <div className="sidebar-label">ÇALIŞMA ALANI</div>
        <button className="nav-item selected" onClick={() => setTab("run")}>
          <span>◇</span> Workflow’lar{" "}
          <span className="count">{workflows.data?.length ?? 0}</span>
        </button>
        <button className="new-flow" onClick={newWorkflow}>
          ＋ Yeni workflow
        </button>
        <div className="workflow-nav">
          {workflows.data?.map((w) => (
            <button
              key={w.id}
              className={selected === w.id ? "current" : ""}
              onClick={() => selectWorkflow(w.id!)}
            >
              <span className="small-dot" />
              {w.name}
              <span>›</span>
            </button>
          ))}
        </div>
        <div className="sidebar-footer">
          <span className="status-dot" /> API ile yönetilen akışlar
          <small>Go · PostgreSQL · River</small>
        </div>
      </aside>
      <main>
        <header>
          <div className="breadcrumb">
            Çalışma alanı <span>/</span> Workflow’lar
          </div>
          {session.data ? (
            <form method="post" action="/auth/logout">
              <button className="button ghost" type="submit">Çıkış</button>
            </form>
          ) : <span className="local-label">LOCAL WORKSPACE</span>}
        </header>
        <section className="content">
          <div className="page-heading">
            <div>
              <div className="eyebrow">VERİDEN ÜRETİME</div>
              <h1>{selectedWorkflow?.name ?? "Workflow oluştur"}</h1>
              <p>
                {selectedWorkflow?.description ??
                  "Akışını JSON ile tanımla, doğrula ve çalıştır."}
              </p>
            </div>
            <div className="heading-meta">
              {selectedWorkflow && (
                <>
                  <span className="revision">
                    REV {selectedWorkflow.revision}
                  </span>
                  <button
                    className="button ghost danger"
                    disabled={remove.isPending}
                    onClick={() => remove.mutate()}
                  >
                    Sil
                  </button>
                </>
              )}
            </div>
          </div>
          <ErrorBox error={workflows.error} />
          <ErrorBox error={remove.error} />
          <div className="tabs">
            <button
              className={tab === "run" ? "active" : ""}
              onClick={() => setTab("run")}
            >
              Çalıştır & izle
            </button>
            <button
              className={tab === "editor" ? "active" : ""}
              onClick={() => setTab("editor")}
            >
              JSON tanımı {dirty && <span className="unsaved-dot" />}
            </button>
            <span>{selected ? short(selected) : "Yeni tanım"}</span>
          </div>
          {tab === "editor" ? (
            <div className="panel editor-panel">
              <div className="panel-heading">
                <div>
                  <h2>Workflow tanımı</h2>
                  <p>
                    Node’lar, model alias’ları ve bağlantılar tek bir JSON
                    grafında.
                  </p>
                </div>
                <span className="subtle">SCHEMA v1</span>
              </div>
              <Suspense
                fallback={<div className="empty">Editör yükleniyor…</div>}
              >
                <JsonEditor value={draft} onChange={edit} />
              </Suspense>
              {syntaxError && (
                <div className="notice error">
                  JSON sözdizimi: {syntaxError}
                </div>
              )}
              <ErrorBox error={save.error} />
              <ErrorBox error={validate.error} />
              {findings && (
                <div className={`notice ${validated ? "success" : "error"}`}>
                  <strong>
                    {validated
                      ? "Tanım geçerli"
                      : `${findings.length} doğrulama bulgusu`}
                  </strong>
                  {findings.map((f, i) => (
                    <div className="finding" key={i}>
                      <code>{f.code}</code>
                      <span>{f.message}</span>
                      <small>{f.path}</small>
                    </div>
                  ))}
                </div>
              )}
              <div className="panel-actions">
                <span>
                  {dirty ? "Kaydedilmemiş değişiklikler" : "Kaydedilmiş tanım"}
                  {parsed?.definition?.nodes &&
                    ` · ${parsed.definition.nodes.length} node`}
                </span>
                <button
                  className="button ghost"
                  disabled={!!syntaxError || validate.isPending}
                  onClick={() => validate.mutate()}
                >
                  {validate.isPending ? "Doğrulanıyor…" : "Doğrula"}
                </button>
                <button
                  className="button primary"
                  disabled={!!syntaxError || save.isPending}
                  onClick={() => save.mutate()}
                >
                  {save.isPending ? "Kaydediliyor…" : "Tanımı kaydet"}
                </button>
              </div>
            </div>
          ) : (
            <>
              <div className="run-layout">
                <div className="panel input-panel">
                  <div className="panel-heading">
                    <div>
                      <h2>Yeni üretim</h2>
                      <p>Girdiyi ekle. Adımlar arka planda sırayla çalışır.</p>
                    </div>
                    <span className="step-icon">↗</span>
                  </div>
                  {selectedWorkflow ? (
                    <>
                      <div className="chain">
                        {selectedWorkflow.definition.nodes.map((n, i) => (
                          <React.Fragment key={n.id}>
                            {i > 0 && <span>→</span>}
                            <div title={n.modelRef}>
                              {n.kind}
                              <small>{n.modelRef ?? n.id}</small>
                            </div>
                          </React.Fragment>
                        ))}
                      </div>
                      <form
                        onSubmit={(e) => {
                          e.preventDefault();
                          run.mutate();
                        }}
                      >
                        {Object.entries(schema?.properties ?? {}).map(
                          ([name, field]) => (
                            <label className="field" key={name}>
                              <span>
                                {name}
                                {schema?.required?.includes(name) && <b> *</b>}
                              </span>
                              {field.type === "boolean" ? (
                                <input
                                  type="checkbox"
                                  checked={!!values[name]}
                                  onChange={(e) =>
                                    setValues({
                                      ...values,
                                      [name]: e.target.checked,
                                    })
                                  }
                                />
                              ) : field.type === "string" ? (
                                <textarea
                                  placeholder="Örn. Ahşap saplı, mat mavi seramik kahve kupası"
                                  value={String(values[name] ?? "")}
                                  required={schema?.required?.includes(name)}
                                  minLength={field.minLength}
                                  maxLength={field.maxLength}
                                  onChange={(e) =>
                                    setValues({
                                      ...values,
                                      [name]: e.target.value,
                                    })
                                  }
                                />
                              ) : (
                                <input
                                  type="number"
                                  step={field.type === "integer" ? 1 : "any"}
                                  min={field.minimum}
                                  max={field.maximum}
                                  required={schema?.required?.includes(name)}
                                  value={String(values[name] ?? "")}
                                  onChange={(e) =>
                                    setValues({
                                      ...values,
                                      [name]:
                                        e.target.value === ""
                                          ? undefined
                                          : Number(e.target.value),
                                    })
                                  }
                                />
                              )}
                            </label>
                          ),
                        )}
                        <ErrorBox error={run.error} />
                        <QuotePanel
                          workflowID={selected}
                          revision={selectedWorkflow.revision}
                          input={values}
                        />
                        <button
                          className="button primary full"
                          disabled={run.isPending}
                        >
                          {run.isPending ? "Başlatılıyor…" : "Üretimi başlat"}{" "}
                          <span>→</span>
                        </button>
                        <small className="hint">
                          Model ve workflow snapshot’ı her run için saklanır.
                        </small>
                      </form>
                    </>
                  ) : (
                    <div className="empty">
                      Çalıştırmak için önce bir workflow kaydet.
                    </div>
                  )}
                </div>
                <div className="panel recent-panel">
                  <div className="panel-heading">
                    <div>
                      <h2>Son çalıştırmalar</h2>
                      <p>Durum, süre ve üretim maliyeti.</p>
                    </div>
                    <button
                      className="icon-button"
                      title="Geçmişi yenile"
                      onClick={() => void history.refetch()}
                    >
                      ↻
                    </button>
                  </div>
                  <ErrorBox error={history.error} />
                  {history.isLoading ? (
                    <div className="empty">Yükleniyor…</div>
                  ) : history.data?.items.length ? (
                    <div className="history">
                      {history.data.items.map((r) => (
                        <button
                          key={r.run_id}
                          className={r.run_id === runID ? "active" : ""}
                          onClick={() => setRunID(r.run_id)}
                        >
                          <div>
                            <code>{short(r.run_id)}</code>
                            <small>
                              {new Date(r.created_at).toLocaleString("tr-TR")}
                            </small>
                          </div>
                          <Badge status={r.status} />
                          <span className="money">
                            <Money value={r.cost.estimated_cost_usd} />
                          </span>
                        </button>
                      ))}
                      {history.data.next_cursor && (
                        <button
                          onClick={async () => {
                            const page = await api<Page<Run>>(
                              `/runs?limit=25&cursor=${encodeURIComponent(history.data!.next_cursor)}${selected ? `&workflow_id=${selected}` : ""}`,
                            );
                            qc.setQueryData(["runs", selected], {
                              items: [...history.data!.items, ...page.items],
                              next_cursor: page.next_cursor,
                            });
                          }}
                        >
                          Daha fazla göster
                        </button>
                      )}
                    </div>
                  ) : (
                    <div className="empty">
                      <span>◷</span>
                      <strong>Henüz bir run yok</strong>
                      <p>İlk üretiminizi başlattığınızda burada görünecek.</p>
                    </div>
                  )}
                </div>
              </div>
              <ErrorBox error={detail.error} />
              {detail.isLoading && runID && (
                <div className="panel empty">Run detayı yükleniyor…</div>
              )}
              {detail.data && <RunDetail run={detail.data} />}
            </>
          )}
        </section>
        <footer>
          FlowKit <span>Tanımla. Çalıştır. İzle.</span>
          <span>Asenkron workflow platformu</span>
        </footer>
      </main>
    </div>
  );
}
function RunDetail({ run }: { run: Run }) {
  const qc = useQueryClient();
  const refresh = useMutation({
    mutationFn: () =>
      api<Run>(`/runs/${run.run_id}/costs/refresh`, { method: "POST" }),
    onSuccess: (updated) => {
      qc.setQueryData(["run", run.run_id], updated);
    },
  });
  const image = run.output?.image;
  return (
    <div className="panel detail-panel">
      <div className="panel-heading">
        <div>
          <h2>
            Çalıştırma izi <code>{short(run.run_id)}</code>
          </h2>
          <p>
            {run.workflow_snapshot.name} · Revision{" "}
            {run.workflow_snapshot.revision} · {run.run_id}
          </p>
        </div>
        <Badge status={run.status} />
      </div>
      <div className="stats">
        <div>
          <span>TOPLAM SÜRE</span>
          <strong>{(run.duration_ms / 1000).toFixed(2)} sn</strong>
        </div>
        <div>
          <span>KUYRUK BEKLEMESİ</span>
          <strong>{(run.queue_wait_ms / 1000).toFixed(2)} sn</strong>
        </div>
        <div>
          <span>BAŞLANGIÇ TAHMİNİ TOPLAM</span>
          <strong>
            <Money value={run.initial_estimate?.cost.estimated_cost_usd} />
          </strong>
          <small>
            {run.initial_estimate?.cost.cost_status === "partial"
              ? "Kısmi · bilinmeyen maliyet var"
              : "USD · üretim öncesi tahmin"}
          </small>
        </div>
        <div>
          <span>GERÇEK TOPLAM MALİYET</span>
          <strong>
            <ActualMoney cost={run.cost} />
          </strong>
          <small>
            {run.cost.actual_cost_status === "complete"
              ? "USD · tüm sağlayıcı ücretleri doğrulandı"
              : "Sağlayıcıların ücret kayıtları tamamlanınca hesaplanır"}
          </small>
        </div>
      </div>
      <div className="billing-summary">
        <span>
          Çalıştırma tahmini: <Money value={run.cost.estimated_cost_usd} /> ·
          Mevcut ücret kayıtları: <Money value={run.cost.reported_cost_usd} />{" "}
          (eksik olabilir)
        </span>
        {!active(run.status) && run.cost.actual_cost_status !== "complete" && (
          <button
            className="button ghost"
            disabled={refresh.isPending}
            onClick={() => refresh.mutate()}
          >
            {refresh.isPending ? "İsteniyor…" : "Maliyet kaydını yenile"}
          </button>
        )}
      </div>
      <ErrorBox error={refresh.error} />
      {run.nodes.some((n) => n.node.modelRef) && <RunCosts run={run} />}
      {run.error && (
        <div className="notice error">
          <strong>
            {run.failed_node_id} · {run.error.code}
          </strong>
          <p>{run.error.message}</p>
          <small>
            {run.error.provider_error_code} {run.error.provider_error_message}
          </small>
        </div>
      )}
      <div className="trace">
        {run.nodes.map((n, i) => (
          <details key={n.node.id} open={n.status === "failed"}>
            <summary>
              <span className="node-index">
                {String(i + 1).padStart(2, "0")}
              </span>
              <div className="node-title">
                <strong>{n.node.id}</strong>
                <small>
                  {n.provider_requests.at(-1)?.actual_model_id ??
                    n.node.modelRef ??
                    n.node.kind}
                </small>
              </div>
              <span className="attempt-count">{n.attempts.length} deneme</span>
              <span>{(n.duration_ms / 1000).toFixed(2)} sn</span>
              <Badge status={n.status} />
              <span>⌄</span>
            </summary>
            <div className="trace-body">
              <div className="trace-meta">
                <span>
                  Tahmini: <Money value={n.cost.estimated_cost_usd} />
                </span>
                <span>
                  Gerçek: <ActualMoney cost={n.cost} />
                </span>
              </div>
              {n.rendered_prompt && (
                <>
                  <h3>Render edilmiş prompt</h3>
                  <pre>{n.rendered_prompt}</pre>
                </>
              )}
              {n.attempts.map((a) => (
                <div key={a.attempt} className="attempt-row">
                  <strong>Deneme {a.attempt}</strong> · {a.phase} ·{" "}
                  {a.duration_ms} ms{" "}
                  {a.error && (
                    <span className="danger">
                      {a.error.code}: {a.error.message}
                    </span>
                  )}
                </div>
              ))}
              {n.provider_requests.map((p) => (
                <div
                  key={p.provider_request_id ?? p.requested_model_id}
                  className="request-row"
                >
                  <code>{p.provider_request_id ?? "Request ID yok"}</code>
                  <span>
                    {p.actual_model_id ?? p.requested_model_id} · {p.phase}
                  </span>
                  <span>
                    Tahmini: <Money value={p.cost.estimated_cost_usd} /> ·
                    Gerçek: <ActualMoney cost={p.cost} />
                  </span>
                  {p.billing?.error && (
                    <span className="danger">
                      Maliyet: {p.billing.error.code} ·{" "}
                      {p.billing.error.message}
                    </span>
                  )}
                </div>
              ))}
              <h3>Girdi / çıktı</h3>
              <pre>{format({ input: n.input, output: n.output })}</pre>
              {n.error && <pre className="danger">{format(n.error)}</pre>}
            </div>
          </details>
        ))}
      </div>
      {image && (
        <div className="result">
          <img
            src={image.url}
            alt={`${run.workflow_snapshot.name} üretim çıktısı`}
          />
          <div>
            <div className="eyebrow">ÜRETİM TAMAMLANDI</div>
            <h2>Görsel çıktısı</h2>
            <p>Üreten model</p>
            <code>{image.actual_model_id}</code>
            <small>
              {image.width} × {image.height} · kimlik kaynağı:{" "}
              {image.model_identity_source}
            </small>
            <a
              className="button ghost"
              href={image.url}
              target="_blank"
              rel="noreferrer"
            >
              Görseli aç ↗
            </a>
          </div>
        </div>
      )}
      {run.status === "succeeded" && !image && <pre>{format(run.output)}</pre>}
    </div>
  );
}
createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
);
