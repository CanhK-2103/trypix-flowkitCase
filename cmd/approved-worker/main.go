// approved-worker is a local acceptance tool, not a production API.
// It runs the normal River/Engine pipeline but permits one explicitly approved
// credentialed HTTP operation at a time. It never runs billing reconciliation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/artifacts"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type operation struct {
	RunID    string `json:"run_id"`
	Version  int64  `json:"version"`
	NodeID   string `json:"node_id"`
	Attempt  int    `json:"attempt"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Action   string `json:"action"`
	Method   string `json:"method"`
	URL      string `json:"url"`
}

type approval struct {
	Operation operation `json:"operation"`
	ID        string    `json:"approval_id"`
}

// nextOperation describes the exact next credentialed call. Empty Action means
// that Engine.Step only commits local state or saves the already produced media.
func nextOperation(r *domain.Run, cfg config.Config) operation {
	o := operation{RunID: r.ID, Version: r.Version}
	if r.Terminal() {
		return o
	}
	for _, n := range r.Nodes {
		if n.Status == "succeeded" {
			continue
		}
		model := engine.ActiveModel(&n)
		if model == nil {
			return o
		}
		o.NodeID, o.Provider, o.Model = n.Node.ID, model.Provider, model.ModelID
		o.Attempt = len(n.Attempts)
		var q *domain.ProviderRequest
		if len(n.Requests) > 0 {
			q = &n.Requests[len(n.Requests)-1]
		}
		if q == nil || q.Phase == "generation_failed" {
			o.Attempt++
			o.Method = http.MethodPost
			if n.Node.Kind == "llm" {
				o.Action, o.URL = "completion", cfg.OpenRouterURL+"/chat/completions"
			} else {
				o.Action, o.URL = "submit", strings.TrimRight(cfg.FalURL, "/")+"/"+model.ModelID
			}
			return o
		}
		if n.Status == "retry_wait" {
			o.Attempt++
		}
		switch q.Phase {
		case "waiting_provider":
			if q.Cost.Source == "" {
				o.Action, o.Method, o.URL = "pricing", http.MethodPost, cfg.FalPricingURL+"/estimate"
			} else {
				o.Action, o.Method, o.URL = "status", http.MethodGet, q.StatusURL
			}
		case "fetch_result":
			o.Action, o.Method, o.URL = "result", http.MethodGet, q.ResponseURL
		}
		return o
	}
	return o
}

type oneCall struct {
	mu       sync.Mutex
	used     bool
	approval approval
	base     http.RoundTripper
	log      *slog.Logger
}

func (g *oneCall) RoundTrip(req *http.Request) (*http.Response, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	o := g.approval.Operation
	if g.used || req.Method != o.Method || req.URL.String() != o.URL {
		return nil, errors.New("unapproved HTTP operation blocked locally")
	}
	g.used = true // Consume before dispatch, including response loss.
	g.log.Info("Approved provider request", "event", "demo.approved_request", "run_id", o.RunID, "node_id", o.NodeID, "attempt", o.Attempt, "approval_id", g.approval.ID, "operation", o.Action, "model_id", o.Model)
	res, err := g.base.RoundTrip(req)
	status := 0
	if res != nil {
		status = res.StatusCode
	}
	g.log.Info("Approved provider response", "event", "demo.approved_response", "run_id", o.RunID, "node_id", o.NodeID, "attempt", o.Attempt, "approval_id", g.approval.ID, "provider_http_status", status)
	return res, err
}

type controlledWorker struct {
	river.WorkerDefaults[jobs.RunArgs]
	mu      sync.Mutex
	engine  *engine.Engine
	tracked map[string]bool
	grants  map[string]approval
	usedIDs map[string]bool
}

func (w *controlledWorker) now() time.Time {
	if w.engine.Now != nil {
		return w.engine.Now().UTC()
	}
	return time.Now().UTC()
}

func activeNode(r *domain.Run) *domain.NodeRun {
	for i := range r.Nodes {
		if r.Nodes[i].Status != "succeeded" {
			return &r.Nodes[i]
		}
	}
	return nil
}

func (w *controlledWorker) recordWait(ctx context.Context, r *domain.Run, at time.Time, source string) error {
	n := activeNode(r)
	if n == nil || n.DeadlineAt == nil || r.Terminal() {
		return nil
	}
	if n.ApprovalPause == nil {
		n.ApprovalPause = &domain.ApprovalPause{OriginalDeadline: *n.DeadlineAt, Source: source}
	}
	if n.ApprovalPause.StartedAt != nil {
		return nil
	}
	n.ApprovalPause.StartedAt = &at
	if err := w.engine.Store.SaveRun(ctx, r); err != nil {
		return err
	}
	w.engine.Log.Info("Human approval wait recorded", "event", "demo.approval_wait_started", "run_id", r.ID, "node_id", n.Node.ID, "attempt", len(n.Attempts), "waiting_since", at, "original_deadline_at", n.ApprovalPause.OriginalDeadline, "source", source)
	return nil
}

func (w *controlledWorker) resumeWait(ctx context.Context, r *domain.Run) error {
	n := activeNode(r)
	if n == nil || n.ApprovalPause == nil || n.ApprovalPause.StartedAt == nil || n.DeadlineAt == nil {
		return nil
	}
	p := n.ApprovalPause
	duration := w.now().Sub(*p.StartedAt)
	if duration < 0 {
		return errors.New("approval wait starts in the future")
	}
	deadline := n.DeadlineAt.Add(duration)
	n.DeadlineAt = &deadline
	p.TotalMS += duration.Milliseconds()
	p.StartedAt = nil
	if err := w.engine.Store.SaveRun(ctx, r); err != nil {
		return err
	}
	w.engine.Log.Info("Human wait excluded from provider budget", "event", "demo.approval_wait_finished", "run_id", r.ID, "node_id", n.Node.ID, "attempt", len(n.Attempts), "approval_wait_ms", p.TotalMS, "original_deadline_at", p.OriginalDeadline, "deadline_at", n.DeadlineAt)
	return nil
}

func (w *controlledWorker) Work(ctx context.Context, job *river.Job[jobs.RunArgs]) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	id := job.Args.RunID
	if !w.tracked[id] {
		return river.JobSnooze(2 * time.Second)
	}
	r, err := w.engine.Store.LoadRun(ctx, id)
	if err != nil {
		return err
	}
	op := nextOperation(r, w.engine.Config)
	if op.Action != "" {
		grant, ok := w.grants[id]
		if !ok || grant.Operation != op {
			delete(w.grants, id)
			if err := w.recordWait(ctx, r, w.now(), "approval_gate"); err != nil {
				return err
			}
			return river.JobSnooze(2 * time.Second)
		}
		delete(w.grants, id)
		if err := w.resumeWait(ctx, r); err != nil {
			return err
		}
		client := providers.New(w.engine.Config)
		// Fresh HTTP/1 connections avoid transport-level replays on reused
		// connections. Redirects are also disabled by providers.New.
		transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DisableKeepAlives: true, TLSHandshakeTimeout: 10 * time.Second}
		client.HTTP.Transport = &oneCall{approval: grant, base: transport, log: w.engine.Log}
		w.engine.Provider = client
	}
	delay, err := w.engine.Step(ctx, id)
	if err != nil {
		return err
	}
	if delay > 0 {
		return river.JobSnooze(delay)
	}
	return nil
}

func (w *controlledWorker) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(res http.ResponseWriter, _ *http.Request) { res.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /next", func(res http.ResponseWriter, req *http.Request) {
		w.mu.Lock()
		defer w.mu.Unlock()
		r, err := w.engine.Store.LoadRun(req.Context(), req.URL.Query().Get("run_id"))
		if err != nil {
			http.Error(res, "Run not found", http.StatusNotFound)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(res).Encode(nextOperation(r, w.engine.Config))
	})
	mux.HandleFunc("POST /track", func(res http.ResponseWriter, req *http.Request) {
		var body struct {
			RunID        string     `json:"run_id"`
			WaitingSince *time.Time `json:"waiting_since,omitempty"`
		}
		if json.NewDecoder(http.MaxBytesReader(res, req.Body, 4096)).Decode(&body) != nil {
			http.Error(res, "Invalid request", http.StatusBadRequest)
			return
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		r, err := w.engine.Store.LoadRun(req.Context(), body.RunID)
		if err != nil {
			http.Error(res, "Run not found", http.StatusNotFound)
			return
		}
		// Explicit migration of a known request from the older approval tool.
		// The operator supplies the recorded request-ID commit log timestamp;
		// no model, request, result, cost or original deadline is rewritten.
		if body.WaitingSince != nil {
			n := activeNode(r)
			if n == nil || n.Status != "waiting_provider" || len(n.Requests) == 0 || r.Terminal() || n.ApprovalPause != nil {
				http.Error(res, "Cannot adopt approval wait", http.StatusConflict)
				return
			}
			q := n.Requests[len(n.Requests)-1]
			if q.ProviderRequestID == "" || q.Phase != "waiting_provider" || body.WaitingSince.Before(q.CreatedAt) || body.WaitingSince.After(w.now()) {
				http.Error(res, "Invalid recorded waiting timestamp", http.StatusBadRequest)
				return
			}
			if err := w.recordWait(req.Context(), r, body.WaitingSince.UTC(), "legacy_request_id_commit_log"); err != nil {
				http.Error(res, "Could not persist approval wait", http.StatusInternalServerError)
				return
			}
		}
		w.tracked[body.RunID] = true
		res.WriteHeader(http.StatusAccepted)
	})
	mux.HandleFunc("POST /approve", func(res http.ResponseWriter, req *http.Request) {
		var a approval
		if json.NewDecoder(http.MaxBytesReader(res, req.Body, 8192)).Decode(&a) != nil || a.ID == "" {
			http.Error(res, "Invalid approval", http.StatusBadRequest)
			return
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		r, err := w.engine.Store.LoadRun(req.Context(), a.Operation.RunID)
		if err != nil || !w.tracked[a.Operation.RunID] || w.usedIDs[a.ID] {
			http.Error(res, "Untracked run or consumed approval", http.StatusConflict)
			return
		}
		if expected := nextOperation(r, w.engine.Config); expected.Action == "" || expected != a.Operation {
			http.Error(res, "Approval does not match current operation", http.StatusConflict)
			return
		}
		if _, exists := w.grants[a.Operation.RunID]; exists {
			http.Error(res, "Approval already pending", http.StatusConflict)
			return
		}
		w.usedIDs[a.ID] = true
		w.grants[a.Operation.RunID] = a
		res.WriteHeader(http.StatusAccepted)
	})
	return mux
}

func main() {
	if err := run(); err != nil {
		telemetry.Logger("approved-worker", "cli").Error("Approval worker failed", "event", "demo.failed", "error", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load() // Keys supplied only by the private container environment.
	log := telemetry.Logger("approved-worker", "local")
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	e := &engine.Engine{Store: storage.New(pool), Provider: providers.New(cfg), Media: artifacts.New(cfg.ArtifactDir), Config: cfg, Log: log}
	w := &controlledWorker{engine: e, tracked: map[string]bool{}, grants: map[string]approval{}, usedIDs: map[string]bool{}}
	workers := river.NewWorkers()
	river.AddWorker(workers, w)
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Workers: workers, Logger: log, Queues: map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}}, MaxAttempts: 5, JobTimeout: 45 * time.Second, RescueStuckJobsAfter: 90 * time.Second, CompletedJobRetentionPeriod: -1})
	if err != nil {
		return err
	}
	if err := client.Start(ctx); err != nil {
		return err
	}
	server := &http.Server{Addr: ":8082", Handler: w.handler(), ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()
	log.Info("Approval worker ready; provider calls blocked until explicitly approved", "event", "demo.ready")
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("control server: %w", err)
		}
	}
	shutdown, stop := context.WithTimeout(context.Background(), 40*time.Second)
	defer stop()
	if err := server.Shutdown(shutdown); err != nil {
		return err
	}
	return client.Stop(shutdown)
}
