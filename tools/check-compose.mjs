import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import { readFileSync } from "node:fs";

const base = new URL(process.argv[2] ?? "http://localhost:8080");
let runID = null;

async function request(path, method = "GET", body, headers = {}) {
  const response = await fetch(new URL(path, base), {
    method,
    headers: { "Content-Type": "application/json", ...headers },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(5000),
  });
  const text = await response.text();
  const data = response.headers
    .get("content-type")
    ?.includes("application/json")
    ? JSON.parse(text)
    : text;
  return { status: response.status, data };
}

function log(level, event, fields = {}) {
  console.log(
    JSON.stringify({
      time: new Date().toISOString(),
      level,
      service: "compose-check",
      run_id: runID,
      node_id: null,
      attempt: null,
      event,
      ...fields,
    }),
  );
}

async function check() {
  assert(
    ["localhost", "127.0.0.1", "[::1]"].includes(base.hostname),
    "Only a local test stack is allowed",
  );
  assert.equal(base.protocol, "http:");
  assert.equal(base.username, "");
  assert.equal(base.password, "");

  const health = await request("/health/ready");
  assert.equal(health.status, 200);
  assert.equal(health.data.status, "ready");
  const page = await request("/");
  assert.equal(page.status, 200);
  assert.match(page.data, /<title>FlowKit/);
  const asset = page.data.match(/src="(\/assets\/[^\"]+\.js)"/);
  assert(asset, "Dashboard must contain a built JavaScript asset");
  assert.equal((await request(asset[1])).status, 200);

  const catalog = await request("/models");
  assert.equal(catalog.status, 200);
  const models = new Map(catalog.data.items.map((m) => [m.alias, m]));
  for (const alias of ["prompt-writer", "flux-schnell", "flux-dev"]) {
    assert(models.get(alias)?.enabled, `Missing enabled seed model: ${alias}`);
  }
  assert.notEqual(
    models.get("flux-schnell").modelId,
    models.get("flux-dev").modelId,
  );

  const workflows = await request("/workflows");
  assert.equal(workflows.status, 200);
  for (const name of ["product", "event"]) {
    const fixture = JSON.parse(
      readFileSync(
        new URL(`../examples/workflows/${name}.json`, import.meta.url),
        "utf8",
      ),
    );
    assert(
      workflows.data.items.some((w) => w.id === fixture.id),
      `Missing workflow: ${name}`,
    );
    const validation = await request("/workflows/validate", "POST", fixture);
    assert.equal(validation.status, 200);
    assert.equal(
      validation.data.valid,
      true,
      JSON.stringify(validation.data.findings),
    );
  }

  // This workflow contains no provider node, so the check never sends a paid request.
  const fixture = {
    name: "Compose smoke check",
    definition: {
      schemaVersion: 1,
      nodes: [
        {
          id: "input",
          kind: "input",
          params: {
            textField: "message",
            inputSchema: {
              type: "object",
              properties: { message: { type: "string", minLength: 1 } },
              required: ["message"],
              additionalProperties: false,
            },
          },
        },
        { id: "output", kind: "output" },
      ],
      edges: [
        { from: "input", fromPort: "text", to: "output", toPort: "text" },
      ],
    },
  };
  const created = await request("/workflows", "POST", fixture);
  assert.equal(created.status, 201);
  try {
    const path = `/workflows/${created.data.id}/runs`;
    const body = { input: { message: "Compose worker is ready" } };
    const headers = { "Idempotency-Key": `compose-${randomUUID()}` };
    const accepted = await request(path, "POST", body, headers);
    assert.equal(accepted.status, 202);
    runID = accepted.data.run_id;
    assert(runID);
    const replay = await request(path, "POST", body, headers);
    assert.equal(replay.status, 200);
    assert.equal(replay.data.run_id, runID);

    const deadline = Date.now() + 20000;
    let run;
    do {
      const result = await request(`/runs/${runID}`);
      assert.equal(result.status, 200);
      run = result.data;
      if (["succeeded", "failed"].includes(run.status)) break;
      await new Promise((resolve) => setTimeout(resolve, 500));
    } while (Date.now() < deadline);
    assert.equal(run.status, "succeeded", JSON.stringify(run.error));
    assert.equal(run.output.text, body.input.message);
    assert.equal(run.nodes.length, 2);
    assert(
      run.nodes.every(
        (n) => n.status === "succeeded" && n.provider_requests.length === 0,
      ),
    );
    assert.equal(run.cost.estimated_cost_usd, "0.000000000000");
    assert.equal(run.cost.cost_status, "complete");
  } finally {
    const deleted = await request(`/workflows/${created.data.id}`, "DELETE");
    assert.equal(deleted.status, 204);
  }
  log("INFO", "compose.checked", {
    models: models.size,
    workflows: workflows.data.items.length,
  });
}

check().catch((error) => {
  log("ERROR", "compose.check_failed", { error: error.message });
  process.exitCode = 1;
});
