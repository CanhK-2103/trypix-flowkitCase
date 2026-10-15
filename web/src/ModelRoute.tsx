import type { NodeRun } from "./types";

export function ModelRoute({ node }: { node: NodeRun }) {
  if (!node.node.fallbackModelRef) return null;
  return (
    <div className="notice">
      <strong>
        Birincil: {node.node.modelRef} · Yedek: {node.node.fallbackModelRef}
      </strong>
      {node.fallback ? (
        <p>
          Yedek modele geçildi: {node.fallback.from} → {node.fallback.to}.
          Neden: {node.fallback.reason.code} · {node.fallback.reason.message}
        </p>
      ) : (
        <p>
          Güvenli geçici üretim hatasında yedek model kullanılır. Toplam en
          fazla 3 deneme.
        </p>
      )}
    </div>
  );
}
