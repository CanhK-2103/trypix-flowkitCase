-- name: CreateModel :exec
INSERT INTO models(alias,data,enabled) VALUES($1,$2,$3);
-- name: GetModel :one
SELECT data FROM models WHERE alias=$1;
-- name: LockModel :one
SELECT data FROM models WHERE alias=$1 FOR SHARE;
-- name: LockModelUpdate :one
SELECT data FROM models WHERE alias=$1 FOR UPDATE;
-- name: UpdateModel :execrows
UPDATE models SET data=$2,enabled=$3 WHERE alias=$1;
-- name: ListModels :many
SELECT alias,data FROM models WHERE alias>sqlc.arg(cursor)::text AND (sqlc.narg(enabled)::boolean IS NULL OR enabled=sqlc.narg(enabled)) ORDER BY alias LIMIT sqlc.arg(page_limit);
-- name: CreateWorkflow :exec
INSERT INTO workflows(id,data,revision) VALUES($1,$2,$3);
-- name: GetWorkflow :one
SELECT data,revision,deleted_at FROM workflows WHERE id=$1;
-- name: LockWorkflow :one
SELECT data,revision,deleted_at FROM workflows WHERE id=$1 FOR SHARE;
-- name: LockWorkflowUpdate :one
SELECT data,revision,deleted_at FROM workflows WHERE id=$1 FOR UPDATE;
-- name: UpdateWorkflow :execrows
UPDATE workflows SET data=$2,revision=revision+1 WHERE id=$1 AND revision=sqlc.arg(expected_revision) AND deleted_at IS NULL;
-- name: DeleteWorkflow :execrows
UPDATE workflows SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL;
-- name: ListWorkflows :many
SELECT id,data FROM workflows WHERE deleted_at IS NULL AND id>sqlc.arg(cursor)::uuid ORDER BY id LIMIT sqlc.arg(page_limit);
-- name: LockIdempotency :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(lock_key)::text,0));
-- name: FindIdempotency :one
SELECT data,request_hash FROM runs WHERE idempotency_key=$1;
-- name: CreateRun :exec
INSERT INTO runs(id,workflow_id,status,data,idempotency_key,request_hash,created_at) VALUES($1,$2,$3,$4,$5,$6,$7);
-- name: GetRun :one
SELECT data,version FROM runs WHERE id=$1;
-- name: SaveRun :execrows
UPDATE runs SET status=$2,data=$3,version=version+1 WHERE id=$1 AND version=sqlc.arg(expected_version);
-- name: ListRuns :many
SELECT id,data FROM runs WHERE (sqlc.narg(workflow_id)::uuid IS NULL OR workflow_id=sqlc.narg(workflow_id)) AND (sqlc.arg(status_filter)::text='' OR status=sqlc.arg(status_filter)) AND (created_at,id)<(sqlc.arg(cursor_time)::timestamptz,sqlc.arg(cursor_id)::uuid) ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(page_limit);
-- name: UpsertNode :exec
INSERT INTO node_runs(run_id,node_id,position,status,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT(run_id,node_id) DO UPDATE SET status=EXCLUDED.status,data=EXCLUDED.data;
-- name: UpsertAttempt :exec
INSERT INTO node_attempts(run_id,node_id,attempt,data) VALUES($1,$2,$3,$4) ON CONFLICT(run_id,node_id,attempt) DO UPDATE SET data=EXCLUDED.data;
-- name: UpsertProviderRequest :exec
INSERT INTO provider_requests(id,run_id,node_id,provider_request_id,phase,estimated_cost_usd,reported_cost_usd,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET provider_request_id=EXCLUDED.provider_request_id,phase=EXCLUDED.phase,estimated_cost_usd=EXCLUDED.estimated_cost_usd,reported_cost_usd=EXCLUDED.reported_cost_usd,data=EXCLUDED.data;
-- name: SaveArtifact :exec
INSERT INTO artifacts(id,run_id,filename,data) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET filename=EXCLUDED.filename,data=EXCLUDED.data;
-- name: GetArtifact :one
SELECT filename,data FROM artifacts WHERE id=$1;

-- name: ListBillingDue :many
SELECT r.id FROM runs r
WHERE r.status IN ('succeeded','failed') AND EXISTS (
 SELECT 1 FROM provider_requests p WHERE p.run_id=r.id
 AND p.data->'cost'->>'actual_cost_usd' IS NULL
 AND COALESCE(p.data->'cost'->>'actual_cost_status','pending') <> 'unavailable'
 AND (p.data->'billing'->>'next_check_at' IS NULL OR (p.data->'billing'->>'next_check_at')::timestamptz<=now())
)
ORDER BY r.created_at LIMIT $1;
