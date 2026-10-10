import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "./api";
import type { Cost, Quote, Run } from "./types";

export function Money({ value }: { value: string | null | undefined }) {
  return <>{value == null ? "Bilinmiyor" : `$${Number(value).toFixed(6)}`}</>;
}
export function ActualMoney({ cost }: { cost: Cost }) {
  if (cost.actual_cost_status === "complete" && cost.actual_cost_usd != null)
    return <Money value={cost.actual_cost_usd} />;
  return (
    <>
      {cost.actual_cost_status === "unavailable"
        ? "Doğrulanamadı"
        : "Bekleniyor"}
    </>
  );
}
export function RunCosts({
  run,
}: {
  run: Pick<Run, "nodes" | "initial_estimate">;
}) {
  return (
    <div className="node-costs">
      <table>
        <caption>Adım maliyetleri · USD</caption>
        <thead>
          <tr>
            <th>Adım / model</th>
            <th>Başlangıç tahmini</th>
            <th>Çalıştırma tahmini</th>
            <th>Gerçek maliyet</th>
          </tr>
        </thead>
        <tbody>
          {run.nodes
            .filter((n) => n.node.modelRef)
            .map((n) => (
              <tr key={n.node.id}>
                <td>
                  <strong>{n.node.id}</strong>
                  <small>
                    {n.provider_requests.at(-1)?.actual_model_id ??
                      n.node.modelRef}
                  </small>
                </td>
                <td>
                  <Money
                    value={
                      run.initial_estimate?.nodes.find(
                        (q) => q.node_id === n.node.id,
                      )?.cost.estimated_cost_usd
                    }
                  />
                </td>
                <td>
                  <Money value={n.cost.estimated_cost_usd} />
                </td>
                <td>
                  <ActualMoney cost={n.cost} />
                </td>
              </tr>
            ))}
        </tbody>
      </table>
    </div>
  );
}
export function QuotePanel({
  workflowID,
  revision,
  input,
}: {
  workflowID: string;
  revision?: number;
  input: Record<string, unknown>;
}) {
  const body = JSON.stringify({ input });
  const [debounced, setDebounced] = useState(body);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(body), 300);
    return () => clearTimeout(timer);
  }, [body]);
  const quote = useQuery({
    queryKey: ["estimate", workflowID, revision, debounced],
    queryFn: ({ signal }) =>
      api<Quote>(`/workflows/${workflowID}/estimate`, {
        method: "POST",
        body: debounced,
        signal,
      }),
    retry: false,
  });
  const updating = body !== debounced || quote.isFetching;
  return (
    <div className="cost-preview" aria-live="polite">
      <div className="cost-preview-heading">
        <strong>Başlangıç tahmini toplam</strong>
        <span>
          {updating ? (
            "Hesaplanıyor…"
          ) : (
            <Money value={quote.data?.cost.estimated_cost_usd} />
          )}
        </span>
      </div>
      {quote.error && (
        <small role="alert">Tahmin alınamadı: {quote.error.message}</small>
      )}
      {quote.data && !updating && (
        <>
          {quote.data.nodes
            .filter((n) => n.model_id)
            .map((n) => (
              <div className="cost-line" key={n.node_id}>
                <span>
                  {n.node_id}
                  <small>{n.model_id}</small>
                </span>
                <span>
                  <Money value={n.cost.estimated_cost_usd} />
                </span>
              </div>
            ))}
          <small>
            {quote.data.cost.cost_status === "partial"
              ? "Kısmi tahmin; fiyatı veya girdisi eksik adım var. "
              : ""}
            Katalog fiyatlarıyla tek deneme; LLM çıktı token sınırı kullanılır.
            Retry hariç, kesin ücret değildir.
          </small>
          <details>
            <summary>Hesaplama varsayımları</summary>
            {quote.data.nodes
              .filter((n) => n.model_id)
              .map((n) => (
                <p key={n.node_id}>
                  {n.node_id}: {n.assumptions.join(" ")}
                </p>
              ))}
          </details>
        </>
      )}
    </div>
  );
}
