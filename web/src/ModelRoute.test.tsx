// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it } from "vitest";
import { ModelRoute } from "./ModelRoute";
import type { NodeRun } from "./types";
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

it("shows the explicit primary/secondary order and recorded switch reason", async () => {
  const el = document.createElement("div");
  const root = createRoot(el);
  const node: NodeRun = {
    node: {
      id: "poster",
      kind: "image",
      modelRef: "flux-krea",
      fallbackModelRef: "flux-dev",
    },
    status: "succeeded",
    duration_ms: 1,
    attempts: [],
    provider_requests: [],
    cost: {
      estimated_cost_usd: null,
      reported_cost_usd: null,
      actual_cost_usd: null,
      actual_cost_status: "pending",
      cost_status: "partial",
    },
  };
  await act(async () => root.render(<ModelRoute node={node} />));
  expect(el.textContent).toContain("Birincil: flux-krea");
  expect(el.textContent).toContain("Yedek: flux-dev");
  expect(el.textContent).not.toContain("Yedek modele geçildi");
  node.fallback = {
    from: "flux-krea",
    to: "flux-dev",
    reason: { code: "PROVIDER_5XX", message: "Unavailable" },
    switched_at: "2026-10-15T00:00:00Z",
  };
  await act(async () => root.render(<ModelRoute node={node} />));
  expect(el.textContent).toContain("flux-krea → flux-dev");
  expect(el.textContent).toContain("PROVIDER_5XX");
  await act(async () => root.unmount());
});
