-- +goose Up
CREATE TABLE models (
    alias TEXT PRIMARY KEY,
    data JSONB NOT NULL,
    enabled BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflows (
    id UUID PRIMARY KEY,
    data JSONB NOT NULL,
    revision INTEGER NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE runs (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    status TEXT NOT NULL,
    data JSONB NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    idempotency_key TEXT UNIQUE,
    request_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX runs_history ON runs (created_at DESC, id DESC);
CREATE INDEX runs_workflow ON runs (workflow_id,created_at DESC,id DESC);
CREATE TABLE node_runs (
    run_id UUID NOT NULL REFERENCES runs(id),
    node_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    status TEXT NOT NULL,
    data JSONB NOT NULL,
    PRIMARY KEY(run_id,node_id)
);

CREATE TABLE node_attempts (
    run_id UUID NOT NULL,
    node_id TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    data JSONB NOT NULL,
    PRIMARY KEY(run_id,node_id,attempt),
    FOREIGN KEY(run_id,node_id) REFERENCES node_runs(run_id,node_id)
);

CREATE TABLE provider_requests (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL,
    node_id TEXT NOT NULL,
    provider_request_id TEXT,
    phase TEXT NOT NULL,
    estimated_cost_usd NUMERIC(24,12),
    reported_cost_usd NUMERIC(24,12),
    data JSONB NOT NULL,
    FOREIGN KEY(run_id,node_id) REFERENCES node_runs(run_id,node_id)
);

CREATE TABLE artifacts (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES runs(id),
    filename TEXT NOT NULL,
    data JSONB NOT NULL
);
-- +goose Down
DROP TABLE artifacts, provider_requests, node_attempts, node_runs, runs, workflows, models;
