# FlowKit project instructions

Implement the approved PLAN.md. Explain decisions in Turkish; use English code identifiers.

- Go API, separate River worker, PostgreSQL, sqlc, Goose; React/TypeScript with TanStack Query and CodeMirror.
- Establish the direct OpenRouter → fal queue experiment first. If local keys are unavailable, prepare the executable experiment and continue independent work; never fabricate live evidence.
- Workflows and compatible model identifiers are data. Never infer models or silently substitute them.
- Validate create/update and return all independent findings.
- Commit fal request IDs immediately, before polling, pricing or media downloads. Persist every execution phase. Resume known IDs; uncertain submissions fail without blind resubmission.
- Maximum three domain attempts per node, transient errors only; queue recovery is separate.
- Run creation and River enqueue share one PostgreSQL transaction. Preserve idempotency and historical snapshots.
- Costs use exact decimal arithmetic, unknown costs are null/partial; polling and restart do not create another charge.
- Preserve the before-run estimate snapshot. Actual cost requires provider accounting for the exact request, never catalog/estimated price; actual totals stay null until all charges are known. Billing failures must not fail/resubmit generation.
- JSON logs include run_id, node_id and attempt, including null context. Never log credentials.
- Tests use local mock providers, pricing and media only. Run gofmt, go test ./..., go vet ./..., integration tests, frontend typecheck/build and sqlc regeneration checks.
- Secrets only in ignored .env; never frontend/Git. Real tests are sequential, minimal, with fal $5 / OpenRouter $2 hard limits and planned $0.50 / $0.05 review thresholds.
- Record actual work, errors and corrections in docs/AI_LOG.md. Distinguish implemented, tested and live verified results.
- Do not restore pre-existing deleted files blindly; recreate files for the approved implementation.
