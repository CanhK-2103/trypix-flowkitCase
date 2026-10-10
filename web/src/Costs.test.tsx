// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, expect, it, vi } from "vitest";
import { ActualMoney, QuotePanel, RunCosts } from "./Costs";
import type { Cost, NodeRun } from "./types";
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(() => {
  for (const root of roots) act(() => root.unmount());
  roots.length = 0;
  document.body.innerHTML = "";
  vi.unstubAllGlobals();
});
function mount() {
  const el = document.createElement("div");
  document.body.append(el);
  const root = createRoot(el);
  roots.push(root);
  return { el, root };
}
it("shows an actual total only when every charge is verified", async () => {
  const { el, root } = mount();
  const cost: Cost = {
    estimated_cost_usd: "0.03",
    reported_cost_usd: "0.001",
    actual_cost_usd: null,
    actual_cost_status: "pending",
    cost_status: "complete",
  };
  await act(async () => root.render(<ActualMoney cost={cost} />));
  expect(el.textContent).toBe("Bekleniyor");
  await act(async () =>
    root.render(
      <ActualMoney cost={{ ...cost, actual_cost_status: "unavailable" }} />,
    ),
  );
  expect(el.textContent).toBe("Doğrulanamadı");
  await act(async () =>
    root.render(
      <ActualMoney
        cost={{
          ...cost,
          actual_cost_usd: "0.0027",
          actual_cost_status: "complete",
        }}
      />,
    ),
  );
  expect(el.textContent).toBe("$0.002700");
});
it("shows a before-run quote through the API without creating a run", async () => {
  const { el, root } = mount();
  const fetch = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({
      cost: { estimated_cost_usd: "0.00315", cost_status: "complete" },
      nodes: [
        {
          node_id: "render",
          model_id: "fal-ai/flux/schnell",
          cost: { estimated_cost_usd: "0.003" },
          assumptions: ["Tek deneme."],
        },
      ],
    }),
  });
  vi.stubGlobal("fetch", fetch);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  await act(async () =>
    root.render(
      <QueryClientProvider client={client}>
        <QuotePanel workflowID="flow" input={{ description: "cup" }} />
      </QueryClientProvider>,
    ),
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
  expect(el.textContent).toContain("Başlangıç tahmini toplam");
  expect(el.textContent).toContain("$0.003150");
  expect(el.textContent).toContain("fal-ai/flux/schnell");
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(fetch.mock.calls[0][0]).toBe("/workflows/flow/estimate");
});
it("shows separate per-node estimate and actual charge with the producing model", async () => {
  const { el, root } = mount();
  const cost: Cost = {
    estimated_cost_usd: "0.003",
    reported_cost_usd: "0.0027",
    actual_cost_usd: "0.0027",
    actual_cost_status: "complete",
    cost_status: "complete",
  };
  const node: NodeRun = {
    node: { id: "render", kind: "image", modelRef: "flux-schnell" },
    status: "succeeded",
    duration_ms: 100,
    attempts: [],
    provider_requests: [
      {
        provider: "fal",
        requested_model_id: "fal-ai/flux/schnell",
        actual_model_id: "fal-ai/flux/schnell",
        phase: "succeeded",
        cost,
      },
    ],
    cost,
  };
  await act(async () => root.render(<RunCosts run={{ nodes: [node] }} />));
  expect(el.textContent).toContain("Çalıştırma tahmini");
  expect(el.textContent).toContain("Gerçek maliyet");
  expect(el.textContent).toContain("$0.003000");
  expect(el.textContent).toContain("$0.002700");
  expect(el.textContent).toContain("fal-ai/flux/schnell");
});
