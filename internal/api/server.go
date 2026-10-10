package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage/dbgen"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
)

type Server struct {
	Store  *storage.Store
	Queue  *river.Client[pgx.Tx]
	Config config.Config
	Log    *slog.Logger
}
type requestKey struct{}

func requestID(r *http.Request) string { v, _ := r.Context().Value(requestKey{}).(string); return v }
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.middleware)
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) { send(w, 200, domain.Object{"status": "ok"}) })
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if e := s.Store.Pool.Ping(ctx); e != nil {
			s.fail(w, r, 503, "DATABASE_UNAVAILABLE", "Database unavailable", nil)
			return
		}
		var ready bool
		if e := s.Store.Pool.QueryRow(ctx, "SELECT to_regclass('runs') IS NOT NULL AND to_regclass('river_job') IS NOT NULL").Scan(&ready); e != nil || !ready {
			s.fail(w, r, 503, "MIGRATION_REQUIRED", "Database migrations are not ready", nil)
			return
		}
		send(w, 200, domain.Object{"status": "ready"})
	})
	r.Post("/models", s.createModel)
	r.Get("/models", s.listModels)
	r.Get("/models/{alias}", s.getModel)
	r.Patch("/models/{alias}", s.patchModel)
	r.Post("/workflows/validate", s.validateWorkflow)
	r.Post("/workflows", s.createWorkflow)
	r.Get("/workflows", s.listWorkflows)
	r.Get("/workflows/{id}", s.getWorkflow)
	r.Put("/workflows/{id}", s.updateWorkflow)
	r.Delete("/workflows/{id}", s.deleteWorkflow)
	r.Post("/workflows/{id}/runs", s.startRun)
	r.Post("/workflows/{id}/estimate", s.estimateRun)
	r.Get("/runs", s.listRuns)
	r.Get("/runs/{id}", s.getRun)
	r.Get("/runs/{id}/status", s.runStatus)
	r.Post("/runs/{id}/costs/refresh", s.refreshCosts)
	r.Get("/artifacts/{id}", s.artifact)
	return r
}
func send(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) fail(w http.ResponseWriter, r *http.Request, status int, code, msg string, findings []domain.Finding) {
	err := domain.Object{"code": code, "message": msg, "request_id": requestID(r)}
	if findings != nil {
		err["findings"] = findings
	}
	s.Log.Warn(msg, "event", "api.rejected", "request_id", requestID(r), "error_code", code, "status", status, "findings", findings)
	send(w, status, domain.Object{"error": err})
}
func (s *Server) dbError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		s.fail(w, r, 404, "NOT_FOUND", "Resource not found", nil)
	case storage.Unique(err):
		s.fail(w, r, 409, "ALREADY_EXISTS", "Resource already exists", nil)
	default:
		s.Log.Error("Database operation failed", "event", "database.error", "request_id", requestID(r), "error", err.Error())
		s.fail(w, r, 503, "DATABASE_UNAVAILABLE", "Database operation failed", nil)
	}
}
func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	d := json.NewDecoder(r.Body)
	d.UseNumber()
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return fmt.Errorf("body must contain one JSON value")
	}
	return nil
}

type recorder struct {
	http.ResponseWriter
	status int
}

func (w *recorder) WriteHeader(status int) { w.status = status; w.ResponseWriter.WriteHeader(status) }
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if _, e := uuid.Parse(id); e != nil {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		r = r.WithContext(context.WithValue(r.Context(), requestKey{}, id))
		start := time.Now()
		rw := &recorder{ResponseWriter: w, status: 200}
		defer func() {
			if p := recover(); p != nil {
				s.Log.Error("API panic", "event", "api.panic", "request_id", id, "panic", fmt.Sprint(p))
				s.fail(rw, r, 500, "INTERNAL_ERROR", "Unexpected application error", nil)
			}
			s.Log.Info("HTTP request", "event", "http.completed", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration_ms", time.Since(start).Milliseconds())
		}()
		next.ServeHTTP(rw, r)
	})
}
func pageLimit(r *http.Request) (int32, error) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return 25, nil
	}
	n, e := strconv.Atoi(v)
	if e != nil || n < 1 || n > 100 {
		return 0, fmt.Errorf("limit must be 1–100")
	}
	return int32(n), nil
}
func (s *Server) createModel(w http.ResponseWriter, r *http.Request) {
	var m domain.Model
	if e := readJSON(w, r, &m); e != nil {
		s.fail(w, r, 400, "INVALID_JSON", e.Error(), nil)
		return
	}
	if f := workflow.ValidateModel(m); len(f) > 0 {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Invalid model", f)
		return
	}
	if m.Defaults == nil {
		m.Defaults = domain.Object{}
	}
	if e := s.Store.Q.CreateModel(r.Context(), dbgen.CreateModelParams{Alias: m.Alias, Data: domain.JSON(m), Enabled: m.Enabled}); e != nil {
		s.dbError(w, r, e)
		return
	}
	s.Log.Info("Model registered", "event", "catalog.created", "model_alias", m.Alias, "requested_model_id", m.ModelID, "request_id", requestID(r))
	send(w, 201, m)
}
func (s *Server) getModel(w http.ResponseWriter, r *http.Request) {
	b, e := s.Store.Q.GetModel(r.Context(), chi.URLParam(r, "alias"))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	send(w, 200, json.RawMessage(b))
}
func (s *Server) patchModel(w http.ResponseWriter, r *http.Request) {
	var patch struct {
		Enabled  *bool           `json:"enabled"`
		Defaults *domain.Object  `json:"defaults"`
		Pricing  *domain.Pricing `json:"pricing"`
	}
	if e := readJSON(w, r, &patch); e != nil {
		s.fail(w, r, 400, "INVALID_JSON", e.Error(), nil)
		return
	}
	tx, e := s.Store.Pool.Begin(r.Context())
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Store.Q.WithTx(tx)
	b, e := q.LockModelUpdate(r.Context(), chi.URLParam(r, "alias"))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	var m domain.Model
	_ = domain.Decode(b, &m)
	if patch.Enabled != nil {
		m.Enabled = *patch.Enabled
	}
	if patch.Defaults != nil {
		m.Defaults = *patch.Defaults
	}
	if patch.Pricing != nil {
		m.Pricing = patch.Pricing
	}
	if f := workflow.ValidateModel(m); len(f) > 0 {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Invalid model update", f)
		return
	}
	if _, e = q.UpdateModel(r.Context(), dbgen.UpdateModelParams{Alias: m.Alias, Data: domain.JSON(m), Enabled: m.Enabled}); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	send(w, 200, m)
}
func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	limit, e := pageLimit(r)
	if e != nil {
		s.fail(w, r, 400, "INVALID_QUERY", e.Error(), nil)
		return
	}
	enabled := pgtype.Bool{}
	if v := r.URL.Query().Get("enabled"); v != "" {
		b, e := strconv.ParseBool(v)
		if e != nil {
			s.fail(w, r, 400, "INVALID_QUERY", "enabled must be boolean", nil)
			return
		}
		enabled = pgtype.Bool{Bool: b, Valid: true}
	}
	rows, e := s.Store.Q.ListModels(r.Context(), dbgen.ListModelsParams{Cursor: r.URL.Query().Get("cursor"), Enabled: enabled, PageLimit: limit + 1})
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	items := []json.RawMessage{}
	next := ""
	for i, row := range rows {
		if i >= int(limit) {
			next = rows[i-1].Alias
			break
		}
		items = append(items, row.Data)
	}
	send(w, 200, domain.Object{"items": items, "next_cursor": next})
}
func workflowBody(w http.ResponseWriter, r *http.Request) (domain.Workflow, []domain.Finding, error) {
	var raw json.RawMessage
	if e := readJSON(w, r, &raw); e != nil {
		return domain.Workflow{}, nil, e
	}
	var wf domain.Workflow
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	d.DisallowUnknownFields()
	if e := d.Decode(&wf); e != nil {
		return wf, []domain.Finding{{Code: "INVALID_STRUCTURE", Message: e.Error(), Path: "/"}}, nil
	}
	return wf, nil, nil
}
func validateMetadata(wf domain.Workflow) []domain.Finding {
	out := []domain.Finding{}
	if strings.TrimSpace(wf.Name) == "" || len(wf.Name) > 200 {
		out = append(out, domain.Finding{Code: "INVALID_STRUCTURE", Message: "Name must contain 1–200 bytes", Path: "/name"})
	}
	if len(wf.Description) > 2000 {
		out = append(out, domain.Finding{Code: "INVALID_STRUCTURE", Message: "Description exceeds 2000 bytes", Path: "/description"})
	}
	return out
}
func (s *Server) findings(ctx context.Context, q *dbgen.Queries, wf domain.Workflow, lock bool) ([]domain.Finding, map[string]domain.Model, error) {
	models, e := s.Store.ModelMap(ctx, q, wf.Definition, lock)
	if e != nil {
		return nil, nil, e
	}
	f := validateMetadata(wf)
	f = append(f, workflow.Validate(wf.Definition, models)...)
	return f, models, nil
}
func (s *Server) validateWorkflow(w http.ResponseWriter, r *http.Request) {
	wf, f, e := workflowBody(w, r)
	if e != nil {
		s.fail(w, r, 400, "INVALID_JSON", e.Error(), nil)
		return
	}
	if len(f) == 0 {
		f, _, e = s.findings(r.Context(), s.Store.Q, wf, false)
		if e != nil {
			s.dbError(w, r, e)
			return
		}
	}
	send(w, 200, domain.Object{"valid": len(f) == 0, "findings": f})
}
func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	wf, f, e := workflowBody(w, r)
	if e != nil {
		s.fail(w, r, 400, "INVALID_JSON", e.Error(), nil)
		return
	}
	if wf.ID == "" {
		wf.ID = uuid.NewString()
	}
	if _, e := uuid.Parse(wf.ID); e != nil {
		s.fail(w, r, 422, "VALIDATION_FAILED", "ID must be UUID", nil)
		return
	}
	if _, e = s.Store.Q.GetWorkflow(r.Context(), storage.UUID(wf.ID)); e == nil {
		s.fail(w, r, 409, "ALREADY_EXISTS", "Workflow ID already exists, including deleted IDs", nil)
		return
	} else if !errors.Is(e, pgx.ErrNoRows) {
		s.dbError(w, r, e)
		return
	}
	tx, e := s.Store.Pool.Begin(r.Context())
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Store.Q.WithTx(tx)
	if len(f) == 0 {
		f, _, e = s.findings(r.Context(), q, wf, true)
	}
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if len(f) > 0 {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Invalid workflow", f)
		return
	}
	now := time.Now().UTC()
	wf.Revision = 1
	wf.CreatedAt = now
	wf.UpdatedAt = now
	e = q.CreateWorkflow(r.Context(), dbgen.CreateWorkflowParams{ID: storage.UUID(wf.ID), Data: domain.JSON(wf), Revision: 1})
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	send(w, 201, wf)
}
func validID(w http.ResponseWriter, r *http.Request, s *Server) bool {
	if _, e := uuid.Parse(chi.URLParam(r, "id")); e != nil {
		s.fail(w, r, 400, "INVALID_ID", "ID must be UUID", nil)
		return false
	}
	return true
}
func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	row, e := s.Store.Q.GetWorkflow(r.Context(), storage.UUID(chi.URLParam(r, "id")))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if row.DeletedAt.Valid {
		s.fail(w, r, 404, "NOT_FOUND", "Workflow deleted", nil)
		return
	}
	send(w, 200, row.Data)
}
func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	wf, f, e := workflowBody(w, r)
	if e != nil {
		s.fail(w, r, 400, "INVALID_JSON", e.Error(), nil)
		return
	}
	id := chi.URLParam(r, "id")
	if wf.ID != "" && wf.ID != id {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Body ID differs from path ID", nil)
		return
	}
	tx, e := s.Store.Pool.Begin(r.Context())
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Store.Q.WithTx(tx)
	row, e := q.LockWorkflowUpdate(r.Context(), storage.UUID(id))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if row.DeletedAt.Valid {
		s.fail(w, r, 404, "NOT_FOUND", "Workflow deleted", nil)
		return
	}
	if len(f) == 0 {
		f, _, e = s.findings(r.Context(), q, wf, true)
	}
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if len(f) > 0 {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Invalid workflow", f)
		return
	}
	var old domain.Workflow
	_ = domain.Decode(row.Data, &old)
	wf.ID = id
	wf.Revision = int(row.Revision) + 1
	wf.CreatedAt = old.CreatedAt
	wf.UpdatedAt = time.Now().UTC()
	count, e := q.UpdateWorkflow(r.Context(), dbgen.UpdateWorkflowParams{ID: storage.UUID(id), Data: domain.JSON(wf), ExpectedRevision: row.Revision})
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if count != 1 {
		s.fail(w, r, 409, "REVISION_CONFLICT", "Workflow changed concurrently", nil)
		return
	}
	if e := tx.Commit(r.Context()); e != nil {
		s.dbError(w, r, e)
		return
	}
	send(w, 200, wf)
}
func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	n, e := s.Store.Q.DeleteWorkflow(r.Context(), storage.UUID(chi.URLParam(r, "id")))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if n == 0 {
		s.fail(w, r, 404, "NOT_FOUND", "Workflow not found", nil)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	limit, e := pageLimit(r)
	if e != nil {
		s.fail(w, r, 400, "INVALID_QUERY", e.Error(), nil)
		return
	}
	cursor := r.URL.Query().Get("cursor")
	if cursor == "" {
		cursor = uuid.Nil.String()
	}
	if _, e := uuid.Parse(cursor); e != nil {
		s.fail(w, r, 400, "INVALID_QUERY", "Invalid cursor UUID", nil)
		return
	}
	rows, e := s.Store.Q.ListWorkflows(r.Context(), dbgen.ListWorkflowsParams{Cursor: storage.UUID(cursor), PageLimit: limit + 1})
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	items := []json.RawMessage{}
	next := ""
	for i, row := range rows {
		if i >= int(limit) {
			next = uuid.UUID(rows[i-1].ID.Bytes).String()
			break
		}
		items = append(items, row.Data)
	}
	send(w, 200, domain.Object{"items": items, "next_cursor": next})
}
func (s *Server) startRun(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	var body struct {
		Input domain.Object `json:"input"`
	}
	if e := readJSON(w, r, &body); e != nil {
		s.fail(w, r, 400, "INVALID_JSON", e.Error(), nil)
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) > 128 || strings.TrimSpace(key) != key || strings.ContainsAny(key, "\r\n") {
		s.fail(w, r, 400, "INVALID_IDEMPOTENCY_KEY", "Key must be at most 128 bytes with no surrounding whitespace", nil)
		return
	}
	id := chi.URLParam(r, "id")
	digest := sha256.Sum256(domain.JSON(domain.Object{"workflow_id": id, "input": body.Input}))
	hash := hex.EncodeToString(digest[:])
	tx, e := s.Store.Pool.Begin(r.Context())
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Store.Q.WithTx(tx)
	if key != "" {
		if e := q.LockIdempotency(r.Context(), key); e != nil {
			s.dbError(w, r, e)
			return
		}
		existing, e := q.FindIdempotency(r.Context(), pgtype.Text{String: key, Valid: true})
		if e == nil {
			if existing.RequestHash != hash {
				s.fail(w, r, 409, "IDEMPOTENCY_CONFLICT", "Key was used for a different workflow/input", nil)
				return
			}
			var run domain.Run
			_ = domain.Decode(existing.Data, &run)
			s.Log.Info("Run replayed", "event", "run.idempotency_replay", "run_id", run.ID, "request_id", requestID(r))
			runResponse(w, 200, run)
			return
		} else if !errors.Is(e, pgx.ErrNoRows) {
			s.dbError(w, r, e)
			return
		}
	}
	row, e := q.LockWorkflow(r.Context(), storage.UUID(id))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if row.DeletedAt.Valid {
		s.fail(w, r, 404, "NOT_FOUND", "Workflow deleted", nil)
		return
	}
	var wf domain.Workflow
	_ = domain.Decode(row.Data, &wf)
	f, models, e := s.findings(r.Context(), q, wf, true)
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	if len(f) > 0 {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Workflow no longer runnable", f)
		return
	}
	ordered := workflow.Order(wf.Definition)
	if e := workflow.ValidateInput(ordered[0], body.Input); e != nil {
		s.fail(w, r, 422, "INVALID_INPUT", e.Error(), nil)
		return
	}
	for _, m := range models {
		if m.Provider == "fal" && s.Config.FalKey == "" || m.Provider == "openrouter" && s.Config.OpenRouterKey == "" {
			s.fail(w, r, 503, "PROVIDER_NOT_CONFIGURED", "Local provider key is missing for "+m.Provider, nil)
			return
		}
	}
	run := domain.Run{ID: uuid.NewString(), Workflow: wf, Input: body.Input, Status: "queued", RequestID: requestID(r), Version: 1, CreatedAt: time.Now().UTC(), Nodes: []domain.NodeRun{}, Cost: domain.Cost{Status: "partial"}}
	quote := engine.QuoteWorkflow(wf, models, body.Input, run.CreatedAt)
	run.InitialEstimate = &quote
	run.Cost = quote.Cost
	for i, node := range ordered {
		nr := domain.NodeRun{Node: node, Status: "pending", Attempts: []domain.Attempt{}, Requests: []domain.ProviderRequest{}, Cost: engine.ZeroCost()}
		if model, ok := models[node.ModelRef]; ok {
			nr.Model = &model
			nr.Cost = quote.Nodes[i].Cost
		}
		run.Nodes = append(run.Nodes, nr)
	}
	job, e := s.Queue.InsertTx(r.Context(), tx, jobs.RunArgs{RunID: run.ID}, nil)
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	run.JobID = job.Job.ID
	e = q.CreateRun(r.Context(), dbgen.CreateRunParams{ID: storage.UUID(run.ID), WorkflowID: storage.UUID(wf.ID), Status: run.Status, Data: domain.JSON(run), IdempotencyKey: pgtype.Text{String: key, Valid: key != ""}, RequestHash: hash, CreatedAt: storage.Timestamp(run.CreatedAt)})
	if e == nil {
		e = s.Store.SyncNodes(r.Context(), q, &run)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	s.Log.Info("Run accepted and enqueued atomically", "event", "run.accepted", "run_id", run.ID, "workflow_id", wf.ID, "job_id", run.JobID, "request_id", run.RequestID)
	runResponse(w, 202, run)
}
func runResponse(w http.ResponseWriter, status int, r domain.Run) {
	w.Header().Set("Location", "/runs/"+r.ID)
	send(w, status, domain.Object{"run_id": r.ID, "status": r.Status, "status_url": "/runs/" + r.ID + "/status", "detail_url": "/runs/" + r.ID, "initial_estimate": r.InitialEstimate})
}
func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	run, e := s.Store.LoadRun(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	send(w, 200, run)
}
func (s *Server) runStatus(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	run, e := s.Store.LoadRun(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	send(w, 200, domain.Object{"run_id": run.ID, "status": run.Status, "failed_node_id": run.FailedNodeID, "error": run.Error})
}
func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	limit, e := pageLimit(r)
	if e != nil {
		s.fail(w, r, 400, "INVALID_QUERY", e.Error(), nil)
		return
	}
	at := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	id := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		decoded, e := base64.RawURLEncoding.DecodeString(cursor)
		parts := strings.Split(string(decoded), "|")
		if e != nil || len(parts) != 2 {
			s.fail(w, r, 400, "INVALID_QUERY", "Invalid cursor", nil)
			return
		}
		at, e = time.Parse(time.RFC3339Nano, parts[0])
		if e != nil || !storage.UUID(parts[1]).Valid {
			s.fail(w, r, 400, "INVALID_QUERY", "Invalid cursor", nil)
			return
		}
		id = parts[1]
	}
	wfID := pgtype.UUID{}
	if value := r.URL.Query().Get("workflow_id"); value != "" {
		wfID = storage.UUID(value)
		if !wfID.Valid {
			s.fail(w, r, 400, "INVALID_QUERY", "workflow_id must be UUID", nil)
			return
		}
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "queued" && status != "running" && status != "succeeded" && status != "failed" {
		s.fail(w, r, 400, "INVALID_QUERY", "Invalid status filter", nil)
		return
	}
	rows, e := s.Store.Q.ListRuns(r.Context(), dbgen.ListRunsParams{WorkflowID: wfID, StatusFilter: status, CursorTime: storage.Timestamp(at), CursorID: storage.UUID(id), PageLimit: limit + 1})
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	items := []domain.Run{}
	next := ""
	for i, row := range rows {
		if i >= int(limit) {
			last := items[len(items)-1]
			next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID))
			break
		}
		var run domain.Run
		if e := domain.Decode(row.Data, &run); e != nil {
			s.dbError(w, r, e)
			return
		}
		items = append(items, run)
	}
	send(w, 200, domain.Object{"items": items, "next_cursor": next})
}
func (s *Server) artifact(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	a, e := s.Store.Artifact(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		s.dbError(w, r, e)
		return
	}
	w.Header().Set("Content-Type", a.MIME)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, filepath.Join(s.Config.ArtifactDir, filepath.Base(a.FileName)))
}
