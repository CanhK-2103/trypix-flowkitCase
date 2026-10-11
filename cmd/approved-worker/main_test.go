package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/riverqueue/river"
)

type memoryStore struct{ run *domain.Run }

func (s *memoryStore) LoadRun(context.Context, string) (*domain.Run, error) {
	var r domain.Run
	_ = domain.Decode(domain.JSON(s.run), &r)
	return &r, nil
}
func (s *memoryStore) SaveRun(_ context.Context, r *domain.Run) error {
	r.Version++
	var copy domain.Run
	_ = domain.Decode(domain.JSON(r), &copy)
	s.run = &copy
	return nil
}
func (*memoryStore) SaveArtifact(context.Context, domain.Artifact) error { return nil }

type mediaFixture struct{}

func (mediaFixture) Download(_ context.Context, _ string, a domain.Artifact) (domain.Artifact, error) {
	return a, nil
}

func TestApprovalPipelineWithLocalHTTPOnly(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "retry requires fresh approval"}[retry], func(t *testing.T) {
			var calls atomic.Int32
			var completions int
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				switch req.URL.Path {
				case "/chat/completions":
					completions++
					if retry && completions == 1 {
						w.WriteHeader(503)
						_, _ = w.Write([]byte(`{"error":{"message":"busy"}}`))
						return
					}
					_, _ = w.Write([]byte(`{"id":"mock-generation","model":"openai/gpt-6-luna","choices":[{"finish_reason":"stop","message":{"content":"A product photo"}}],"usage":{"cost":0.00002}}`))
				case "/pricing/estimate":
					_, _ = w.Write([]byte(`{"currency":"USD","total_cost":0.003}`))
				case "/status":
					_, _ = w.Write([]byte(`{"status":"COMPLETED"}`))
				case "/result":
					_ = json.NewEncoder(w).Encode(domain.Object{"images": []domain.Object{{"url": server.URL + "/image", "width": 512, "height": 512}}})
				case "/fal-ai/flux/schnell":
					_ = json.NewEncoder(w).Encode(domain.Object{"request_id": "mock-fal", "status_url": server.URL + "/status", "response_url": server.URL + "/result"})
				default:
					t.Errorf("Unexpected request %s", req.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			cfg := config.Load()
			cfg.OpenRouterKey, cfg.FalKey = "fixture-key", "fixture-key"
			cfg.OpenRouterURL, cfg.FalURL, cfg.FalPricingURL = server.URL, server.URL, server.URL+"/pricing"
			cfg.RetryBase = time.Millisecond
			now := time.Now().UTC()
			var workflow domain.Workflow
			b, _ := examples.Files.ReadFile("workflows/product.json")
			_ = domain.Decode(b, &workflow)
			r := &domain.Run{ID: "fixture-run", Workflow: workflow, Input: domain.Object{"description": "cup"}, CreatedAt: now, Status: "queued"}
			for _, n := range workflow.Definition.Nodes {
				nr := domain.NodeRun{Node: n, Status: "pending", Cost: engine.ZeroCost()}
				if n.ModelRef != "" {
					b, _ := examples.Files.ReadFile("models/" + n.ModelRef + ".json")
					_ = domain.Decode(b, &nr.Model)
				}
				r.Nodes = append(r.Nodes, nr)
			}
			store := &memoryStore{r}
			log := slog.New(slog.NewJSONHandler(io.Discard, nil))
			e := &engine.Engine{Store: store, Config: cfg, Media: mediaFixture{}, Log: log, Now: func() time.Time { return now }, Jitter: func() time.Duration { return 0 }}
			worker := &controlledWorker{engine: e, tracked: map[string]bool{}, grants: map[string]approval{}, usedIDs: map[string]bool{}}
			job := &river.Job[jobs.RunArgs]{Args: jobs.RunArgs{RunID: r.ID}}
			_ = worker.Work(context.Background(), job)
			if calls.Load() != 0 || store.run.StartedAt != nil {
				t.Fatal("untracked run executed")
			}
			control := worker.handler()
			invoke := func(path string, body any) int {
				res := httptest.NewRecorder()
				control.ServeHTTP(res, httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(domain.JSON(body)))))
				return res.Code
			}
			if invoke("/track", domain.Object{"run_id": r.ID}) != 202 {
				t.Fatal("track")
			}
			_ = worker.Work(context.Background(), job) // input only
			_ = worker.Work(context.Background(), job) // completion blocked
			if calls.Load() != 0 {
				t.Fatal("call without approval")
			}
			approvals := 0
			for i := 0; i < 35 && !store.run.Terminal(); i++ {
				op := nextOperation(store.run, cfg)
				before := calls.Load()
				if op.Action != "" {
					_ = worker.Work(context.Background(), job)
					if calls.Load() != before {
						t.Fatal("automatic extra call")
					}
					op = nextOperation(store.run, cfg) // A new wait is durably recorded.
					a := approval{Operation: op, ID: time.Now().String()}
					stale := a
					stale.Operation.Version--
					if invoke("/approve", stale) != 409 {
						t.Fatal("stale grant accepted")
					}
					if invoke("/approve", a) != 202 {
						t.Fatal("grant")
					}
					if invoke("/approve", a) != 409 {
						t.Fatal("duplicate grant accepted")
					}
					approvals++
				}
				_ = worker.Work(context.Background(), job)
				now = now.Add(time.Second)
				if calls.Load()-before > 1 {
					t.Fatal("multiple calls per approval")
				}
			}
			want := int32(5)
			if retry {
				want++
			}
			if store.run.Status != "succeeded" || calls.Load() != want || int32(approvals) != want {
				t.Fatalf("status=%s calls=%d approvals=%d", store.run.Status, calls.Load(), approvals)
			}
			if store.run.Nodes[2].Requests[0].ProviderRequestID != "mock-fal" {
				t.Fatal("missing ticket checkpoint")
			}
		})
	}
}

func TestOneCallBlocksDifferentAndRepeatedRequests(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	guard := &oneCall{approval: approval{Operation: operation{Method: "GET", URL: server.URL + "/approved"}}, base: http.DefaultTransport, log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	client := &http.Client{Transport: guard}
	if _, err := client.Get(server.URL + "/other"); err == nil {
		t.Fatal("wrong URL allowed")
	}
	res, err := client.Get(server.URL + "/approved")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if _, err := client.Get(server.URL + "/approved"); err == nil {
		t.Fatal("replayed request allowed")
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
}
