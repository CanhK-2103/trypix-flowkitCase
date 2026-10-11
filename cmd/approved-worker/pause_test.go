package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/riverqueue/river"
)

func TestLongApprovalWaitResumesKnownRequestWithoutAnotherSubmission(t *testing.T) {
	calls := 0
	var remote *httptest.Server
	remote = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/status":
			_, _ = w.Write([]byte(`{"status":"COMPLETED"}`))
		case "/result":
			_ = json.NewEncoder(w).Encode(domain.Object{"images": []domain.Object{{"url": remote.URL + "/image", "width": 512, "height": 512}}})
		default:
			t.Errorf("Unexpected request/resubmission: %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer remote.Close()
	now := time.Now().UTC()
	started := now.Add(-20 * time.Minute)
	originalDeadline := started.Add(10 * time.Minute)
	cfg := config.Load()
	cfg.FalURL, cfg.FalKey = remote.URL, "fixture-key"
	newRun := func() *domain.Run {
		return &domain.Run{ID: "pause-fixture", Status: "running", CreatedAt: started, StartedAt: &started, Version: 1,
			InitialEstimate: &domain.Quote{Cost: domain.Cost{Estimated: engine.Money("0.003")}},
			Nodes: []domain.NodeRun{{Node: domain.Node{ID: "image", Kind: "image"}, Model: &domain.Model{Provider: "fal", ModelID: "fixture/model"},
				Status: "waiting_provider", StartedAt: &started, DeadlineAt: &originalDeadline, RenderedPrompt: "photo",
				Attempts: []domain.Attempt{{Number: 1, StartedAt: started, Phase: "waiting_provider"}},
				Requests: []domain.ProviderRequest{{ID: "fixture-ref", ProviderRequestID: "already-submitted", ActualModelID: "fixture/model", Phase: "waiting_provider", CreatedAt: started,
					StatusURL: remote.URL + "/status", ResponseURL: remote.URL + "/result", Cost: domain.Cost{Source: "catalog", Estimated: engine.Money("0.003")}}}}}}
	}
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	// The production engine still fails an expired node without any remote call.
	standardStore := &memoryStore{newRun()}
	standard := &engine.Engine{Store: standardStore, Config: cfg, Log: log, Now: func() time.Time { return now }}
	if _, err := standard.Step(context.Background(), "pause-fixture"); err != nil {
		t.Fatal(err)
	}
	if standardStore.run.Error.Code != "PROVIDER_TIMEOUT" || calls != 0 {
		t.Fatal("production timeout changed")
	}

	store := &memoryStore{newRun()}
	e := &engine.Engine{Store: store, Config: cfg, Log: log, Media: mediaFixture{}, Now: func() time.Time { return now }}
	w := &controlledWorker{engine: e, tracked: map[string]bool{}, grants: map[string]approval{}, usedIDs: map[string]bool{}}
	control := w.handler()
	post := func(path string, body any) int {
		res := httptest.NewRecorder()
		control.ServeHTTP(res, httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(domain.JSON(body)))))
		return res.Code
	}
	waitSince := started.Add(time.Second)
	if post("/track", domain.Object{"run_id": "pause-fixture", "waiting_since": waitSince}) != 202 {
		t.Fatal("legacy wait adoption")
	}
	job := &river.Job[jobs.RunArgs]{Args: jobs.RunArgs{RunID: "pause-fixture"}}
	_ = w.Work(context.Background(), job)
	if calls != 0 || store.run.Nodes[0].ApprovalPause.StartedAt == nil {
		t.Fatal("waiting executed a request")
	}
	for i, action := range []string{"status", "result"} {
		_ = w.Work(context.Background(), job) // Persist next approval wait.
		now = now.Add(40 * time.Minute)       // A human takes much longer than the provider budget.
		op := nextOperation(store.run, cfg)
		if op.Action != action {
			t.Fatal(op)
		}
		if post("/approve", approval{Operation: op, ID: action}) != 202 {
			t.Fatal("approval")
		}
		if err := w.Work(context.Background(), job); err == nil {
			t.Fatal("expected durable snooze")
		}
		if calls != i+1 {
			t.Fatal("extra call")
		}
	}
	for i := 0; i < 3; i++ {
		_ = w.Work(context.Background(), job)
	}
	r := store.run
	n := r.Nodes[0]
	if r.Status != "succeeded" || len(n.Requests) != 1 || len(n.Attempts) != 1 || n.Requests[0].ProviderRequestID != "already-submitted" || calls != 2 {
		t.Fatal("request/attempt duplicated or recovery failed", r.Status, calls)
	}
	if !n.ApprovalPause.OriginalDeadline.Equal(originalDeadline) || n.ApprovalPause.TotalMS < int64((90*time.Minute)/time.Millisecond) || n.ApprovalPause.StartedAt != nil {
		t.Fatal("missing original deadline/wait accounting", n.ApprovalPause)
	}
	if *r.InitialEstimate.Cost.Estimated != "0.003000000000" {
		t.Fatal("estimate changed")
	}
}
