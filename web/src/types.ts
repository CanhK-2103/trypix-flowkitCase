export type Node = {
  id: string;
  kind: string;
  modelRef?: string;
  fallbackModelRef?: string;
  params?: Record<string, unknown>;
  promptTemplate?: string;
};
export type Workflow = {
  id?: string;
  name: string;
  description?: string;
  definition: {
    schemaVersion: number;
    nodes: Node[];
    edges: { from: string; fromPort: string; to: string; toPort: string }[];
  };
  revision?: number;
};
export type Cost = {
  estimated_cost_usd: string | null;
  reported_cost_usd: string | null;
  actual_cost_usd: string | null;
  actual_cost_status: string;
  actual_cost_source?: string;
  cost_status: string;
  pricing_source?: string;
};
export type Quote = {
  workflow_revision: number;
  created_at: string;
  cost: Cost;
  nodes: {
    node_id: string;
    model_id?: string;
    cost: Cost;
    assumptions: string[];
  }[];
};
export type Failure = {
  code: string;
  message: string;
  provider_error_code?: string;
  provider_error_message?: string;
  provider_http_status?: number;
};
export type Artifact = {
  artifact_id: string;
  url: string;
  actual_model_id: string;
  width: number;
  height: number;
  model_identity_source: string;
};
export type Attempt = {
  attempt: number;
  phase: string;
  duration_ms: number;
  error?: Failure;
  provider_request_ref?: string;
};
export type NodeRun = {
  node: Node;
  active_model_ref?: string;
  fallback?: { from: string; to: string; reason: Failure; switched_at: string };
  status: string;
  duration_ms: number;
  input?: unknown;
  output?: unknown;
  rendered_prompt?: string;
  attempts: Attempt[];
  provider_requests: {
    id?: string;
    model_ref?: string;
    provider: string;
    actual_model_id?: string;
    requested_model_id: string;
    provider_request_id?: string;
    cost: Cost;
    phase: string;
    billing?: {
      checks: number;
      error?: Failure;
      checked_at?: string;
      next_check_at?: string;
    };
  }[];
  error?: Failure;
  cost: Cost;
};
export type Run = {
  run_id: string;
  workflow_snapshot: Workflow;
  status: string;
  created_at: string;
  duration_ms: number;
  queue_wait_ms: number;
  nodes: NodeRun[];
  output?: { image?: Artifact };
  error?: Failure;
  failed_node_id?: string;
  cost: Cost;
  input: Record<string, unknown>;
  initial_estimate?: Quote;
};
