import fs from "node:fs";
const ref = (name) => ({ $ref: `#/components/schemas/${name}` });
const object = (properties, required = [], extra = false) => ({
  type: "object",
  properties,
  required,
  additionalProperties: extra,
});
const str = { type: "string" },
  time = { type: "string", format: "date-time" },
  id = { type: "string", format: "uuid" },
  money = { type: ["string", "null"], pattern: "^[0-9]+\\.[0-9]{12}$" },
  anyObject = { type: "object", additionalProperties: true };
const graph = JSON.parse(
  fs.readFileSync("schemas/workflow.schema.json", "utf8"),
);
delete graph.$id;
delete graph.$schema;
const schemas = {
  Definition: graph,
  Pricing: object(
    {
      currency: { const: "USD" },
      unit: { enum: ["tokens", "megapixel", "image"] },
      unitPrice: str,
      inputPerMillion: str,
      outputPerMillion: str,
      source: str,
      updatedAt: time,
    },
    ["currency", "unit", "source", "updatedAt"],
  ),
  Model: object(
    {
      alias: str,
      provider: { enum: ["openrouter", "fal"] },
      modelId: str,
      inputProfile: {
        enum: ["openrouter-chat-text-v1", "fal-text-to-image-v1"],
      },
      defaults: anyObject,
      enabled: { type: "boolean" },
      pricing: ref("Pricing"),
    },
    ["alias", "provider", "modelId", "inputProfile", "enabled"],
  ),
  ModelPatch: object({
    enabled: { type: "boolean" },
    defaults: anyObject,
    pricing: ref("Pricing"),
  }),
  Workflow: object(
    {
      id,
      name: str,
      description: str,
      definition: ref("Definition"),
      revision: { type: "integer" },
      created_at: time,
      updated_at: time,
    },
    ["name", "definition"],
  ),
  Finding: object(
    {
      code: str,
      message: str,
      path: str,
      node_id: str,
      edge_index: { type: "integer" },
    },
    ["code", "message", "path"],
  ),
  Validation: object(
    {
      valid: { type: "boolean" },
      findings: { type: "array", items: ref("Finding") },
    },
    ["valid", "findings"],
  ),
  Failure: object(
    {
      code: str,
      message: str,
      provider_error_code: str,
      provider_error_message: str,
      provider_http_status: { type: "integer" },
      retryable: { type: "boolean" },
      retry_after_ms: { type: "integer" },
      submission_uncertain: { type: "boolean" },
    },
    ["code", "message", "retryable"],
  ),
  Error: object(
    {
      error: object(
        {
          code: str,
          message: str,
          request_id: str,
          findings: { type: "array", items: ref("Finding") },
        },
        ["code", "message", "request_id"],
      ),
    },
    ["error"],
  ),
  Cost: object(
    {
      estimated_cost_usd: money,
      reported_cost_usd: money,
      actual_cost_usd: money,
      actual_cost_status: { enum: ["complete", "pending", "unavailable"] },
      actual_cost_source: str,
      cost_status: { enum: ["complete", "partial"] },
      pricing_source: str,
    },
    [
      "estimated_cost_usd",
      "reported_cost_usd",
      "actual_cost_usd",
      "actual_cost_status",
      "cost_status",
    ],
  ),
  Quote: object(
    {
      workflow_revision: { type: "integer" },
      created_at: time,
      cost: ref("Cost"),
      nodes: {
        type: "array",
        items: object(
          {
            node_id: str,
            model_id: str,
            cost: ref("Cost"),
            assumptions: { type: "array", items: str },
          },
          ["node_id", "cost", "assumptions"],
        ),
      },
    },
    ["workflow_revision", "created_at", "cost", "nodes"],
  ),
  StartRun: object({ input: anyObject }, ["input"]),
  AcceptedRun: object(
    {
      run_id: id,
      status: { enum: ["queued", "running", "succeeded", "failed"] },
      status_url: str,
      detail_url: str,
      initial_estimate: { anyOf: [ref("Quote"), { type: "null" }] },
    },
    ["run_id", "status", "status_url", "detail_url"],
  ),
  NodeRun: object(
    {
      node: graph.properties.nodes.items,
      model_snapshot: ref("Model"),
      status: str,
      input: anyObject,
      output: anyObject,
      rendered_prompt: str,
      started_at: time,
      finished_at: time,
      deadline_at: time,
      next_retry_at: time,
      duration_ms: { type: "integer" },
      attempts: { type: "array", items: anyObject },
      provider_requests: { type: "array", items: anyObject },
      error: ref("Failure"),
      cost: ref("Cost"),
    },
    ["node", "status", "duration_ms", "attempts", "provider_requests", "cost"],
  ),
  Run: object(
    {
      run_id: id,
      workflow_snapshot: ref("Workflow"),
      input: anyObject,
      status: { enum: ["queued", "running", "succeeded", "failed"] },
      request_id: str,
      job_id: { type: "integer" },
      version: { type: "integer" },
      created_at: time,
      started_at: time,
      finished_at: time,
      queue_wait_ms: { type: "integer" },
      duration_ms: { type: "integer" },
      nodes: { type: "array", items: ref("NodeRun") },
      failed_node_id: str,
      error: ref("Failure"),
      output: anyObject,
      cost: ref("Cost"),
      initial_estimate: ref("Quote"),
    },
    [
      "run_id",
      "workflow_snapshot",
      "input",
      "status",
      "request_id",
      "job_id",
      "version",
      "created_at",
      "nodes",
      "cost",
    ],
  ),
  RunStatus: object(
    {
      run_id: id,
      status: str,
      failed_node_id: str,
      error: { anyOf: [ref("Failure"), { type: "null" }] },
    },
    ["run_id", "status", "failed_node_id", "error"],
  ),
  Health: object({ status: str }, ["status"]),
  Capabilities: object({ billing_enabled: { type: "boolean" } }, ["billing_enabled"]),
};
for (const [name, item] of [
  ["Models", "Model"],
  ["Workflows", "Workflow"],
  ["Runs", "Run"],
])
  schemas[name] = object(
    { items: { type: "array", items: ref(item) }, next_cursor: str },
    ["items", "next_cursor"],
  );
const response = (name, description = "Success") => ({
  description,
  content: { "application/json": { schema: ref(name) } },
});
const errors = Object.fromEntries(
  [400, 404, 409, 422, 503].map((code) => [
    code,
    response("Error", "Typed error"),
  ]),
);
const body = (name) => ({
  required: true,
  content: { "application/json": { schema: ref(name) } },
});
const pathParam = (name, format) => ({
  in: "path",
  name,
  required: true,
  schema: format === "uuid" ? id : str,
});
const page = [
  {
    in: "query",
    name: "limit",
    schema: { type: "integer", minimum: 1, maximum: 100, default: 25 },
  },
  { in: "query", name: "cursor", schema: str },
];
const operation = (summary, code, result, request, parameters = []) => ({
  summary,
  parameters,
  ...(request ? { requestBody: body(request) } : {}),
  responses: {
    ...errors,
    [code]: result ? response(result) : { description: "Deleted" },
  },
});
const paths = {
  "/capabilities": { get: operation("Environment accounting capability", 200, "Capabilities") },
  "/models": {
    get: operation("List models", 200, "Models", null, [
      ...page,
      { in: "query", name: "enabled", schema: { type: "boolean" } },
    ]),
    post: operation("Register model", 201, "Model", "Model"),
  },
  "/models/{alias}": {
    get: operation("Get model", 200, "Model", null, [pathParam("alias")]),
    patch: operation(
      "Update defaults, pricing or enabled; identity is immutable",
      200,
      "Model",
      "ModelPatch",
      [pathParam("alias")],
    ),
  },
  "/workflows": {
    get: operation("List active workflows", 200, "Workflows", null, page),
    post: operation(
      "Validate and create workflow",
      201,
      "Workflow",
      "Workflow",
    ),
  },
  "/workflows/validate": {
    post: operation(
      "Return independent validation findings without saving",
      200,
      "Validation",
      "Workflow",
    ),
  },
  "/workflows/{id}": {
    get: operation("Get workflow", 200, "Workflow", null, [
      pathParam("id", "uuid"),
    ]),
    put: operation(
      "Validate and replace workflow",
      200,
      "Workflow",
      "Workflow",
      [pathParam("id", "uuid")],
    ),
    delete: operation("Soft delete workflow; preserve runs", 204, null, null, [
      pathParam("id", "uuid"),
    ]),
  },
  "/workflows/{id}/runs": {
    post: operation(
      "Accept an asynchronous run",
      202,
      "AcceptedRun",
      "StartRun",
      [
        pathParam("id", "uuid"),
        {
          in: "header",
          name: "Idempotency-Key",
          schema: { type: "string", maxLength: 128 },
        },
      ],
    ),
  },
  "/workflows/{id}/estimate": {
    post: operation(
      "Estimate per-node and total cost without generation; incomplete inputs/prices return partial",
      200,
      "Quote",
      "StartRun",
      [pathParam("id", "uuid")],
    ),
  },
  "/runs": {
    get: operation("List runs", 200, "Runs", null, [
      ...page,
      { in: "query", name: "workflow_id", schema: id },
      {
        in: "query",
        name: "status",
        schema: { enum: ["queued", "running", "succeeded", "failed"] },
      },
    ]),
  },
  "/runs/{id}": {
    get: operation(
      "Get persisted execution trace, output and costs",
      200,
      "Run",
      null,
      [pathParam("id", "uuid")],
    ),
  },
  "/runs/{id}/status": {
    get: operation("Get run status", 200, "RunStatus", null, [
      pathParam("id", "uuid"),
    ]),
  },
  "/runs/{id}/costs/refresh": {
    post: operation(
      "Schedule accounting reads for a finished run; does not generate again",
      202,
      "Run",
      null,
      [pathParam("id", "uuid")],
    ),
  },
  "/artifacts/{id}": {
    get: {
      summary: "Read durable image",
      parameters: [pathParam("id", "uuid")],
      responses: {
        ...errors,
        200: {
          description: "Image bytes",
          content: Object.fromEntries(
            ["image/png", "image/jpeg", "image/webp"].map((type) => [
              type,
              { schema: { type: "string", format: "binary" } },
            ]),
          ),
        },
      },
    },
  },
  "/health/live": { get: operation("Liveness", 200, "Health") },
  "/health/ready": {
    get: operation("Database/migration readiness", 200, "Health"),
  },
};
paths["/workflows/{id}/runs"].post.responses[200] = response(
  "AcceptedRun",
  "Idempotency replay; no new job",
);
const doc = {
  openapi: "3.1.0",
  info: {
    title: "FlowKit API",
    version: "1.0.0",
    description:
      "Linear workflows as data, asynchronous execution, persisted provenance and estimated costs.",
  },
  servers: [{ url: "http://localhost:8080" }],
  paths,
  components: { schemas },
};
fs.writeFileSync("docs/openapi.json", JSON.stringify(doc, null, 2) + "\n");
