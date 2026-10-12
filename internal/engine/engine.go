package engine

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/workflow"
	"github.com/google/uuid"
)

type Store interface {
	LoadRun(context.Context, string) (*domain.Run, error)
	SaveRun(context.Context, *domain.Run) error
	SaveArtifact(context.Context, domain.Artifact) error
}
type Provider interface {
	Complete(context.Context, domain.Model, string, domain.Object) (providers.Completion, *domain.Failure)
	Submit(context.Context, domain.Model, string, domain.Object) (providers.Ticket, *domain.Failure)
	Status(context.Context, providers.Ticket) (providers.QueueState, *domain.Failure)
	Result(context.Context, providers.Ticket) (providers.ImageResult, *domain.Failure)
	Estimate(context.Context, domain.Model) (string, *domain.Failure)
}
type Media interface {
	Download(context.Context, string, domain.Artifact) (domain.Artifact, error)
}
type Engine struct {
	Store    Store
	Provider Provider
	Media    Media
	Config   config.Config
	Log      *slog.Logger
	Now      func() time.Time
	Jitter   func() time.Duration
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}
func (e *Engine) save(ctx context.Context, r *domain.Run) error {
	costs := []domain.Cost{}
	for i := range r.Nodes {
		n := &r.Nodes[i]
		if len(n.Requests) > 0 {
			parts := []domain.Cost{}
			for qi := range n.Requests {
				q := &n.Requests[qi]
				if q.Cost.ActualStatus == "" {
					q.Cost.ActualStatus = "pending"
				}
				parts = append(parts, q.Cost)
			}
			n.Cost = SumCosts(parts)
		} else if r.Terminal() {
			n.Cost = ZeroCost()
		}
		costs = append(costs, n.Cost)
	}
	r.Cost = SumCosts(costs)
	return e.Store.SaveRun(ctx, r)
}
func (e *Engine) event(r *domain.Run, n *domain.NodeRun, level slog.Level, event string, args ...any) {
	if e.Log == nil {
		return
	}
	base := []any{"event", event, "run_id", r.ID, "workflow_id", r.Workflow.ID, "request_id", r.RequestID, "job_id", r.JobID, "node_id", n.Node.ID, "node_kind", n.Node.Kind, "attempt", len(n.Attempts), "status", n.Status}
	if n.Model != nil {
		base = append(base, "provider", n.Model.Provider, "model_alias", n.Model.Alias, "requested_model_id", n.Model.ModelID)
	}
	if n.DeadlineAt != nil {
		base = append(base, "deadline_at", n.DeadlineAt)
	}
	if n.StartedAt != nil {
		base = append(base, "node_elapsed_ms", e.now().Sub(*n.StartedAt).Milliseconds())
	}
	if len(n.Requests) > 0 {
		q := n.Requests[len(n.Requests)-1]
		base = append(base, "phase", q.Phase, "provider_request_id", q.ProviderRequestID, "actual_model_id", q.ActualModelID, "model_identity_source", q.IdentitySource)
	}
	values := map[string]any{}
	for i := 0; i+1 < len(base); i += 2 {
		values[base[i].(string)] = base[i+1]
	}
	for i := 0; i+1 < len(args); i += 2 {
		values[args[i].(string)] = args[i+1]
	}
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	attrs := []any{}
	for _, key := range keys {
		attrs = append(attrs, key, values[key])
	}
	e.Log.Log(context.Background(), level, event, attrs...)
}
func (e *Engine) Step(ctx context.Context, id string) (time.Duration, error) {
	r, err := e.Store.LoadRun(ctx, id)
	if err != nil {
		return 0, err
	}
	if r.Terminal() {
		return 0, nil
	}
	now := e.now()
	if r.StartedAt == nil {
		r.StartedAt = &now
		r.QueueWaitMS = now.Sub(r.CreatedAt).Milliseconds()
		r.Status = "running"
	}
	idx := -1
	for i := range r.Nodes {
		if r.Nodes[i].Status != "succeeded" {
			idx = i
			break
		}
	}
	if idx < 0 {
		r.Status = "succeeded"
		r.FinishedAt = &now
		r.DurationMS = now.Sub(*r.StartedAt).Milliseconds()
		if len(r.Nodes) > 0 {
			r.Output = r.Nodes[len(r.Nodes)-1].Output
		}
		if err := e.save(ctx, r); err != nil {
			return 0, err
		}
		if len(r.Nodes) > 0 {
			e.event(r, &r.Nodes[len(r.Nodes)-1], slog.LevelInfo, "run.succeeded", "duration_ms", r.DurationMS, "estimated_cost_usd", r.Cost.Estimated, "reported_cost_usd", r.Cost.Reported, "cost_status", r.Cost.Status)
		}
		return 0, nil
	}
	n := &r.Nodes[idx]
	e.event(r, n, slog.LevelDebug, "checkpoint.loaded", "checkpoint_version", r.Version)
	if n.NextRetryAt != nil {
		if now.Before(*n.NextRetryAt) {
			return n.NextRetryAt.Sub(now), nil
		}
		n.NextRetryAt = nil
	}
	if n.StartedAt == nil {
		n.StartedAt = &now
		n.Status = "running"
		n.Input = domain.Object{}
		if idx > 0 {
			n.Input = r.Nodes[idx-1].Output
		} else {
			n.Input = r.Input
		}
		if n.Model != nil {
			duration := e.Config.LLMDeadline
			if n.Node.Kind == "image" {
				duration = e.Config.ImageDeadline
			}
			deadline := now.Add(duration)
			n.DeadlineAt = &deadline
		}
		e.event(r, n, slog.LevelInfo, "node.started")
	}
	if n.DeadlineAt != nil && !now.Before(*n.DeadlineAt) {
		return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "PROVIDER_TIMEOUT", Message: "Persisted node deadline exceeded"})
	}
	if n.Node.Kind == "input" {
		textField, _ := n.Node.Params["textField"].(string)
		n.Output = domain.Object{"text": r.Input[textField]}
		return e.finishNode(ctx, r, n)
	}
	if n.Node.Kind == "output" {
		n.Output = n.Input
		return e.finishNode(ctx, r, n)
	}
	if n.Model == nil {
		return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "MODEL_NOT_REGISTERED", Message: "Run snapshot lacks required model"})
	}
	params := workflow.Merge(n.Model.Defaults, n.Node.Params)
	if n.RenderedPrompt == "" {
		if n.Node.PromptTemplate != "" {
			prior := map[string]domain.Object{}
			for i := 0; i < idx; i++ {
				prior[r.Nodes[i].Node.ID] = r.Nodes[i].Output
			}
			n.RenderedPrompt, err = workflow.Render(n.Node.PromptTemplate, r.Input, prior)
			if err != nil {
				return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "TEMPLATE_RESOLUTION_ERROR", Message: err.Error()})
			}
		} else {
			n.RenderedPrompt, _ = n.Input["prompt"].(string)
		}
		if n.RenderedPrompt == "" {
			return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "INVALID_INPUT", Message: "Rendered prompt is empty"})
		}
	}
	var q *domain.ProviderRequest
	if len(n.Requests) > 0 {
		q = &n.Requests[len(n.Requests)-1]
	}
	if q == nil || q.Phase == "generation_failed" {
		if len(n.Attempts) >= 3 {
			return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "RUN_INFRASTRUCTURE_ERROR", Message: "Attempt limit exhausted"})
		}
		request := domain.ProviderRequest{ID: uuid.NewString(), Provider: n.Model.Provider, RequestedModelID: n.Model.ModelID, Phase: "sending", CreatedAt: now, Pricing: n.Model.Pricing, Cost: domain.Cost{Status: "partial"}}
		n.Requests = append(n.Requests, request)
		q = &n.Requests[len(n.Requests)-1]
		n.Attempts = append(n.Attempts, domain.Attempt{Number: len(n.Attempts) + 1, Phase: "sending", StartedAt: now, ProviderRequestRef: q.ID})
		n.Status = "running"
		if err := e.save(ctx, r); err != nil {
			return 0, err
		}
		e.event(r, n, slog.LevelInfo, "provider.dispatch", "provider_request_ref", q.ID, "prompt_length", len(n.RenderedPrompt))
		remoteCtx, cancel := context.WithDeadline(ctx, *n.DeadlineAt)
		defer cancel()
		if n.Node.Kind == "llm" {
			response, f := e.Provider.Complete(remoteCtx, *n.Model, n.RenderedPrompt, params)
			q.Raw = response.Raw
			q.Usage = response.Usage
			q.ProviderRequestID = response.ID
			q.ActualModelID = response.Model
			q.IdentitySource = "provider_response"
			q.Cost = TokenCost(*n.Model, response.Usage)
			if f != nil {
				return e.providerFailure(ctx, r, idx, f, true)
			}
			q.Phase = "result_saved"
			n.Output = domain.Object{"prompt": response.Prompt}
			if err := e.save(ctx, r); err != nil {
				return 0, err
			}
			e.event(r, n, slog.LevelInfo, "provider.result_committed", "provider_request_id", q.ProviderRequestID, "actual_model_id", q.ActualModelID)
			return time.Millisecond, nil
		}
		ticket, f := e.Provider.Submit(remoteCtx, *n.Model, n.RenderedPrompt, params)
		q.ProviderRequestID = ticket.ID
		q.StatusURL = ticket.StatusURL
		q.ResponseURL = ticket.ResponseURL
		q.Raw = ticket.Raw
		q.ActualModelID = n.Model.ModelID
		q.IdentitySource = "endpoint"
		if f != nil {
			return e.providerFailure(ctx, r, idx, f, true)
		}
		q.Phase = "waiting_provider"
		n.Status = "waiting_provider"
		n.Attempts[len(n.Attempts)-1].Phase = q.Phase
		if err := e.save(ctx, r); err != nil {
			return 0, err
		}
		e.event(r, n, slog.LevelInfo, "provider.request_id_committed", "provider_request_id", ticket.ID, "phase", q.Phase)
		return time.Millisecond, nil
	}
	if q.Phase == "sending" {
		return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "PROVIDER_SUBMISSION_UNKNOWN", Message: "Worker resumed a sending phase without a durable provider response; no resubmission", Uncertain: true})
	}
	if len(n.Attempts) > 0 && n.Attempts[len(n.Attempts)-1].FinishedAt != nil && n.Status == "retry_wait" {
		n.Attempts = append(n.Attempts, domain.Attempt{Number: len(n.Attempts) + 1, Phase: q.Phase, StartedAt: now, ProviderRequestRef: q.ID})
		n.Status = "waiting_provider"
		if err := e.save(ctx, r); err != nil {
			return 0, err
		}
	}
	remoteCtx, cancel := context.WithDeadline(ctx, *n.DeadlineAt)
	defer cancel()
	ticket := providers.Ticket{ID: q.ProviderRequestID, StatusURL: q.StatusURL, ResponseURL: q.ResponseURL}
	switch q.Phase {
	case "waiting_provider":
		if q.Cost.Source == "" {
			price, f := e.Provider.Estimate(remoteCtx, *n.Model)
			if f == nil && Money(price) != nil {
				q.Cost = domain.Cost{Estimated: Money(price), Status: "complete", Source: "fal historical_api_price; one call"}
			} else {
				q.Cost = domain.Cost{Status: "partial", Source: "unavailable"}
				if f != nil {
					e.event(r, n, slog.LevelError, "provider.pricing_error", "error_code", f.Code, "provider_error_code", f.ProviderCode, "provider_error_message", f.ProviderMessage, "provider_http_status", f.HTTPStatus)
				}
				e.event(r, n, slog.LevelWarn, "pricing.unavailable")
			}
			if err := e.save(ctx, r); err != nil {
				return 0, err
			}
			// Pricing and queue status are separate durable worker steps.
			return time.Millisecond, nil
		}
		state, f := e.Provider.Status(remoteCtx, ticket)
		if f != nil {
			return e.providerFailure(ctx, r, idx, f, false)
		}
		e.event(r, n, slog.LevelDebug, "provider.status", "provider_request_id", q.ProviderRequestID, "provider_status", state.Status)
		if state.Error != nil {
			return e.providerFailure(ctx, r, idx, state.Error, true)
		}
		if state.Status == "COMPLETED" {
			q.Phase = "fetch_result"
			n.Attempts[len(n.Attempts)-1].Phase = q.Phase
			if err := e.save(ctx, r); err != nil {
				return 0, err
			}
			return time.Millisecond, nil
		}
		return e.Config.PollInterval, nil
	case "fetch_result":
		result, f := e.Provider.Result(remoteCtx, ticket)
		if f != nil {
			return e.providerFailure(ctx, r, idx, f, false)
		}
		q.Raw = result.Raw
		q.Usage = domain.Object{"billable_units": result.BillableUnits}
		q.Phase = "result_saved"
		n.Output = domain.Object{"source_url": result.URL, "width": result.Width, "height": result.Height}
		if q.Cost.Estimated == nil {
			q.Cost = CatalogImageCost(*n.Model, result.Width, result.Height)
		}
		if err := e.save(ctx, r); err != nil {
			return 0, err
		}
		e.event(r, n, slog.LevelInfo, "provider.result_committed", "provider_request_id", q.ProviderRequestID)
		return time.Millisecond, nil
	case "result_saved":
		if n.Node.Kind == "llm" {
			return e.finishNode(ctx, r, n)
		}
		source, _ := n.Output["source_url"].(string)
		width := numericInt(n.Output["width"])
		height := numericInt(n.Output["height"])
		artifact := domain.Artifact{ID: q.ID, RunID: r.ID, NodeID: n.Node.ID, RequestID: q.ID, URL: "/artifacts/" + q.ID, SourceURL: source, Width: width, Height: height, ActualModelID: q.ActualModelID, IdentitySource: q.IdentitySource}
		saved, err := e.Media.Download(remoteCtx, source, artifact)
		if err != nil {
			return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "STORAGE_ERROR", Message: err.Error()})
		}
		if err := e.Store.SaveArtifact(ctx, saved); err != nil {
			return 0, err
		}
		n.Output = domain.Object{"image": saved}
		return e.finishNode(ctx, r, n)
	default:
		return 0, e.fail(ctx, r, idx, &domain.Failure{Code: "RUN_INFRASTRUCTURE_ERROR", Message: "Unsupported persisted provider phase: " + q.Phase})
	}
}
func numericInt(v any) int { var n int; _, _ = fmt.Sscan(fmt.Sprint(v), &n); return n }
func (e *Engine) finishNode(ctx context.Context, r *domain.Run, n *domain.NodeRun) (time.Duration, error) {
	now := e.now()
	n.Status = "succeeded"
	n.FinishedAt = &now
	n.DurationMS = now.Sub(*n.StartedAt).Milliseconds()
	if len(n.Requests) == 0 {
		n.Cost = ZeroCost()
	}
	for i := range n.Requests {
		q := &n.Requests[i]
		if q.FinishedAt == nil {
			q.FinishedAt = &now
		}
		if i == len(n.Requests)-1 {
			q.Phase = "succeeded"
		}
	}
	if len(n.Attempts) > 0 {
		a := &n.Attempts[len(n.Attempts)-1]
		a.Phase = "succeeded"
		a.FinishedAt = &now
		a.DurationMS = now.Sub(a.StartedAt).Milliseconds()
	}
	if err := e.save(ctx, r); err != nil {
		return 0, err
	}
	e.event(r, n, slog.LevelInfo, "node.succeeded", "duration_ms", n.DurationMS, "estimated_cost_usd", n.Cost.Estimated)
	return time.Millisecond, nil
}
func (e *Engine) providerFailure(ctx context.Context, r *domain.Run, idx int, f *domain.Failure, newGeneration bool) (time.Duration, error) {
	n := &r.Nodes[idx]
	now := e.now()
	a := &n.Attempts[len(n.Attempts)-1]
	a.Error = f
	a.FinishedAt = &now
	a.DurationMS = now.Sub(a.StartedAt).Milliseconds()
	a.Phase = "failed"
	q := &n.Requests[len(n.Requests)-1]
	if newGeneration {
		q.Phase = "generation_failed"
		q.FinishedAt = &now
		if f.HTTPStatus >= 400 && f.HTTPStatus < 500 && f.HTTPStatus != 408 && f.HTTPStatus != 429 && q.ProviderRequestID == "" {
			q.Cost = ZeroCost()
		}
	}
	e.event(r, n, slog.LevelError, "provider.error", "error_code", f.Code, "provider_error_code", f.ProviderCode, "provider_error_message", f.ProviderMessage, "provider_http_status", f.HTTPStatus, "retryable", f.Retryable, "provider_request_id", q.ProviderRequestID)
	if !f.Retryable || f.Uncertain || len(n.Attempts) >= 3 {
		return 0, e.fail(ctx, r, idx, f)
	}
	delay := e.Config.RetryBase * time.Duration(1<<(len(n.Attempts)-1))
	if e.Jitter != nil {
		delay += e.Jitter()
	} else {
		delay += time.Duration(rand.IntN(251)) * time.Millisecond
	}
	if time.Duration(f.RetryAfterMS)*time.Millisecond > delay {
		delay = time.Duration(f.RetryAfterMS) * time.Millisecond
	}
	next := now.Add(delay)
	if n.DeadlineAt != nil && !next.Before(*n.DeadlineAt) {
		return 0, e.fail(ctx, r, idx, f)
	}
	n.NextRetryAt = &next
	n.Status = "retry_wait"
	if err := e.save(ctx, r); err != nil {
		return 0, err
	}
	e.event(r, n, slog.LevelInfo, "node.retry_scheduled", "backoff_ms", delay.Milliseconds(), "next_attempt_at", next)
	return delay, nil
}
func (e *Engine) fail(ctx context.Context, r *domain.Run, idx int, f *domain.Failure) error {
	now := e.now()
	n := &r.Nodes[idx]
	n.Status = "failed"
	n.Error = f
	n.FinishedAt = &now
	if n.StartedAt != nil {
		n.DurationMS = now.Sub(*n.StartedAt).Milliseconds()
	}
	if len(n.Attempts) > 0 {
		a := &n.Attempts[len(n.Attempts)-1]
		if a.FinishedAt == nil {
			a.FinishedAt = &now
			a.DurationMS = now.Sub(a.StartedAt).Milliseconds()
			a.Error = f
			a.Phase = "failed"
		}
	}
	for i := idx + 1; i < len(r.Nodes); i++ {
		r.Nodes[i].Status = "skipped"
		r.Nodes[i].Cost = ZeroCost()
	}
	r.Status = "failed"
	r.FailedNodeID = n.Node.ID
	r.Error = f
	r.FinishedAt = &now
	if r.StartedAt != nil {
		r.DurationMS = now.Sub(*r.StartedAt).Milliseconds()
	}
	e.event(r, n, slog.LevelError, "run.failed", "error_code", f.Code, "provider_error_code", f.ProviderCode, "provider_error_message", f.ProviderMessage)
	return e.save(ctx, r)
}
func (e *Engine) FailInfrastructure(ctx context.Context, r *domain.Run) error {
	idx := 0
	for i, n := range r.Nodes {
		if n.Status != "succeeded" {
			idx = i
			break
		}
	}
	return e.fail(ctx, r, idx, &domain.Failure{Code: "RUN_INFRASTRUCTURE_ERROR", Message: "River job reached a terminal infrastructure failure"})
}
