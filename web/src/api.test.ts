import { describe, it, expect, vi, afterEach } from "vitest";
import { api, ApiError, pollInterval, retryGet } from "./api";
afterEach(() => vi.unstubAllGlobals());
describe("dashboard requests", () => {
  it("polls active runs and then pending billing until accounting finishes", () => {
    expect(pollInterval("queued")).toBe(2000);
    expect(pollInterval("running")).toBe(2000);
    expect(pollInterval("failed")).toBe(false);
    expect(pollInterval("succeeded")).toBe(false);
    expect(pollInterval("succeeded", "pending")).toBe(15000);
    expect(pollInterval("failed", "pending")).toBe(15000);
    expect(pollInterval("succeeded", "complete")).toBe(false);
    expect(pollInterval("succeeded", "unavailable")).toBe(false);
  });
  it("retries a GET once for transient errors only", () => {
    expect(retryGet(0, new ApiError(503, "DB", "unavailable"))).toBe(true);
    expect(retryGet(1, new ApiError(503, "DB", "unavailable"))).toBe(false);
    expect(retryGet(0, new ApiError(422, "INVALID", "invalid"))).toBe(false);
  });
  it("preserves backend findings and makes a mutation only once", async () => {
    const fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 422,
      json: async () => ({
        error: {
          code: "VALIDATION_FAILED",
          message: "bad graph",
          findings: [
            {
              code: "CYCLE_DETECTED",
              message: "cycle",
              path: "/definition/edges",
            },
          ],
        },
      }),
    });
    vi.stubGlobal("fetch", fetch);
    try {
      await api("/workflows", { method: "POST", body: "{}" });
    } catch (e) {
      expect((e as ApiError).findings[0].code).toBe("CYCLE_DETECTED");
    }
    expect(fetch).toHaveBeenCalledTimes(1);
  });
  it("preserves idempotency header on a manual replay", async () => {
    const fetch = vi
      .fn()
      .mockRejectedValueOnce(new Error("network"))
      .mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: async () => ({ run_id: "run-1" }),
      });
    vi.stubGlobal("fetch", fetch);
    const request = {
      method: "POST",
      body: "{}",
      headers: { "Idempotency-Key": "stable-key" },
    };
    await expect(api("/workflows/id/runs", request)).rejects.toBeInstanceOf(
      ApiError,
    );
    await api("/workflows/id/runs", request);
    expect(fetch.mock.calls[0][1].headers["Idempotency-Key"]).toBe(
      "stable-key",
    );
    expect(fetch.mock.calls[1][1].headers["Idempotency-Key"]).toBe(
      "stable-key",
    );
  });
});
