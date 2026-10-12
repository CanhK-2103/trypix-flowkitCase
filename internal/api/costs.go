package api

import (
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

func (s *Server) estimateRun(w http.ResponseWriter, r *http.Request) {
	if !validID(w, r, s) {
		return
	}
	var body struct {
		Input domain.Object `json:"input"`
	}
	if err := readJSON(w, r, &body); err != nil {
		s.fail(w, r, 400, "INVALID_JSON", err.Error(), nil)
		return
	}
	row, err := s.Store.Q.GetWorkflow(r.Context(), storage.UUID(chi.URLParam(r, "id")))
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if row.DeletedAt.Valid {
		s.fail(w, r, 404, "NOT_FOUND", "Workflow deleted", nil)
		return
	}
	var wf domain.Workflow
	if err := domain.Decode(row.Data, &wf); err != nil {
		s.dbError(w, r, err)
		return
	}
	findings, models, err := s.findings(r.Context(), s.Store.Q, wf, false)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if len(findings) > 0 {
		s.fail(w, r, 422, "VALIDATION_FAILED", "Workflow no longer runnable", findings)
		return
	}
	send(w, 200, engine.QuoteWorkflow(wf, models, body.Input, time.Now().UTC()))
}
func (s *Server) refreshCosts(w http.ResponseWriter, r *http.Request) {
	if s.Config.BillingDisabled {
		s.fail(w, r, 409, "BILLING_DISABLED", "Provider accounting queries are disabled in this environment", nil)
		return
	}
	if !validID(w, r, s) {
		return
	}
	run, err := s.Store.LoadRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if !run.Terminal() {
		s.fail(w, r, 409, "RUN_NOT_FINISHED", "Billing refresh is available after execution finishes", nil)
		return
	}
	e := engine.Engine{Store: s.Store}
	if err := e.RefreshCosts(r.Context(), run); err != nil {
		s.dbError(w, r, err)
		return
	}
	s.Log.Info("Billing refresh scheduled", "event", "billing.refresh_requested", "run_id", run.ID, "request_id", requestID(r))
	send(w, 202, run)
}
