# FlowKit project instructions

This repository implements the Trypix FlowKit case. Build in understandable stages with the candidate. Explain substantive design decisions in Turkish and keep code identifiers in English.

## Requirements

- Prefer Go for the backend. The main assessment is backend correctness and flexibility.
- First prove a direct OpenRouter -> fal queue workflow with real providers, then migrate that same model, prompt and parameters into the engine.
- Workflow definitions and model aliases must live in data. Adding a compatible model or a second workflow must not require code changes or a restart.
- Never infer a missing model or silently substitute a different model.
- Validate every workflow creation and update; return a list of findings.
- Execute runs asynchronously. Store the fal request ID immediately after submission.
- Retry only transient provider errors, at most three total attempts per node.
- Record actual model, node inputs/outputs, attempts, timing and typed errors persistently.
- Log JSON with run_id, node_id and attempt; never log API keys.
- Tests must never call real providers. Use local HTTP fixtures or mock adapters.
- Do not claim acceptance scenarios succeeded unless they were actually executed.
- Finish required functionality before optional bonuses.

## Working conventions

- Keep provider secrets in ignored .env files. Never place them in frontend code or Git.
- Keep dependencies and abstractions small; document tradeoffs.
- Run gofmt, go test ./... and go vet ./... for meaningful Go changes.
- Maintain docs/AI_LOG.md with actual assistance, errors and corrections. Do not invent a story about AI mistakes.
- README must distinguish current functionality from planned functionality.
- No external publication or messages are needed to implement and test locally.
