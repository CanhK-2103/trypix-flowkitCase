package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/api"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/artifacts"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage/dbgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type fixture struct {
	store  *storage.Store
	server *httptest.Server
	cfg    config.Config
	log    *slog.Logger
}

func database(t *testing.T) *storage.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("Set TEST_DATABASE_URL for local PostgreSQL integration tests")
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatal("Integration tests require a dedicated database ending in _test")
	}
	pool, e := pgxpool.NewWithConfig(context.Background(), cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	if e := storage.Migrate(context.Background(), pool, log); e != nil {
		t.Fatal(e)
	}
	_, e = pool.Exec(context.Background(), "TRUNCATE models,workflows,runs,node_runs,node_attempts,provider_requests,artifacts,river_job CASCADE")
	if e != nil {
		t.Fatal(e)
	}
	return storage.New(pool)
}
func setupAPI(t *testing.T) *fixture {
	t.Helper()
	store := database(t)
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	queue, e := jobs.New(store.Pool, nil, log, 0)
	if e != nil {
		t.Fatal(e)
	}
	cfg := config.Load()
	cfg.OpenRouterKey = "mock-key"
	cfg.FalKey = "mock-key"
	cfg.ArtifactDir = t.TempDir()
	server := httptest.NewServer((&api.Server{Store: store, Queue: queue, Config: cfg, Log: log}).Handler())
	t.Cleanup(server.Close)
	return &fixture{store, server, cfg, log}
}
func request(t *testing.T, f *fixture, method, path string, body any, key string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(domain.JSON(body))
	}
	req, e := http.NewRequest(method, f.server.URL+path, reader)
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return res.StatusCode, data
}
func example(t *testing.T, path string) domain.Object {
	t.Helper()
	data, e := examples.Files.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var value domain.Object
	if e := domain.Decode(data, &value); e != nil {
		t.Fatal(e)
	}
	return value
}
func load(t *testing.T, f *fixture) {
	t.Helper()
	for _, alias := range []string{"prompt-writer", "flux-schnell", "flux-dev"} {
		status, b := request(t, f, "POST", "/models", example(t, "models/"+alias+".json"), "")
		if status != 201 {
			t.Fatal(status, string(b))
		}
	}
	status, b := request(t, f, "POST", "/workflows", example(t, "workflows/product.json"), "")
	if status != 201 {
		t.Fatal(status, string(b))
	}
}

const productID = "10000000-0000-4000-8000-000000000001"

func TestCatalogWorkflowCRUDAndValidation(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	for _, path := range []string{"/models", "/models/flux-dev", "/workflows", "/workflows/" + productID} {
		status, b := request(t, f, "GET", path, nil, "")
		if status != 200 {
			t.Fatal(status, string(b))
		}
	}
	workflow := example(t, "workflows/product.json")
	workflow["name"] = "Updated name"
	status, b := request(t, f, "PUT", "/workflows/"+productID, workflow, "")
	if status != 200 {
		t.Fatal(status, string(b))
	}
	var updated domain.Workflow
	_ = domain.Decode(b, &updated)
	if updated.Revision != 2 {
		t.Fatal(updated)
	}
	nodes := workflow["definition"].(map[string]any)["nodes"].([]any)
	nodes[1].(map[string]any)["modelRef"] = "unregistered"
	nodes[1].(map[string]any)["promptTemplate"] = "{{input.missing}}"
	status, b = request(t, f, "POST", "/workflows/validate", workflow, "")
	if status != 200 {
		t.Fatal(status, string(b))
	}
	var validation struct {
		Valid    bool
		Findings []domain.Finding
	}
	_ = domain.Decode(b, &validation)
	if validation.Valid || len(validation.Findings) < 2 {
		t.Fatal(string(b))
	}
	status, _ = request(t, f, "PUT", "/workflows/"+productID, workflow, "")
	if status != 422 {
		t.Fatal(status)
	}
	row, e := f.store.Q.GetWorkflow(context.Background(), storage.UUID(productID))
	if e != nil || row.Revision != 2 {
		t.Fatal("invalid update modified old revision", e)
	}
	status, _ = request(t, f, "PATCH", "/models/flux-schnell", domain.Object{"enabled": false}, "")
	if status != 200 {
		t.Fatal(status)
	}
	status, _ = request(t, f, "POST", "/workflows/"+productID+"/runs", domain.Object{"input": domain.Object{"description": "cup"}}, "")
	if status != 422 {
		t.Fatal("disabled model accepted", status)
	}
	status, _ = request(t, f, "DELETE", "/workflows/"+productID, nil, "")
	if status != 204 {
		t.Fatal(status)
	}
	status, _ = request(t, f, "GET", "/workflows/"+productID, nil, "")
	if status != 404 {
		t.Fatal(status)
	}
	status, _ = request(t, f, "POST", "/workflows", example(t, "workflows/product.json"), "")
	if status != 409 {
		t.Fatal("deleted bootstrap ID recreated", status)
	}
}
func TestConcurrentIdempotencyAndSnapshot(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	input := domain.Object{"input": domain.Object{"description": "cup"}}
	var wg sync.WaitGroup
	ids := make(chan string, 10)
	codes := make(chan int, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, b := request(t, f, "POST", "/workflows/"+productID+"/runs", input, "same-key")
			var response struct {
				ID string `json:"run_id"`
			}
			_ = domain.Decode(b, &response)
			ids <- response.ID
			codes <- status
		}()
	}
	wg.Wait()
	close(ids)
	close(codes)
	one := ""
	created := 0
	for id := range ids {
		if one == "" {
			one = id
		}
		if id == "" || id != one {
			t.Fatal("duplicate run", id, one)
		}
	}
	for status := range codes {
		if status == 202 {
			created++
		} else if status != 200 {
			t.Fatal(status)
		}
	}
	if created != 1 {
		t.Fatal("expected one create", created)
	}
	var count int
	_ = f.store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate jobs", count)
	}
	status, _ := request(t, f, "POST", "/workflows/"+productID+"/runs", domain.Object{"input": domain.Object{"description": "other"}}, "same-key")
	if status != 409 {
		t.Fatal(status)
	}
	wf := example(t, "workflows/product.json")
	wf["definition"].(map[string]any)["nodes"].([]any)[2].(map[string]any)["modelRef"] = "flux-dev"
	status, _ = request(t, f, "PUT", "/workflows/"+productID, wf, "")
	if status != 200 {
		t.Fatal(status)
	}
	run, e := f.store.LoadRun(context.Background(), one)
	if e != nil || run.Nodes[2].Model.ModelID != "fal-ai/flux/schnell" {
		t.Fatal("snapshot changed", e)
	}
	request(t, f, "DELETE", "/workflows/"+productID, nil, "")
	status, b := request(t, f, "POST", "/workflows/"+productID+"/runs", input, "same-key")
	if status != 200 || !bytes.Contains(b, []byte(one)) {
		t.Fatal("replay after delete", status, string(b))
	}
}
func TestRunJobTransactionRollbackAndSQLTypes(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	ctx := context.Background()
	tx, e := f.store.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	queue, e := jobs.New(f.store.Pool, nil, f.log, 0)
	if e != nil {
		t.Fatal(e)
	}
	id := uuid.NewString()
	job, e := queue.InsertTx(ctx, tx, jobs.RunArgs{RunID: id}, nil)
	if e != nil {
		t.Fatal(e)
	}
	q := f.store.Q.WithTx(tx)
	if e := q.CreateRun(ctx, dbgen.CreateRunParams{ID: storage.UUID(id), WorkflowID: storage.UUID(productID), Status: "queued", Data: domain.JSON(domain.Object{"job_id": job.Job.ID, "nested": domain.Object{"value": "test"}}), RequestHash: "hash", CreatedAt: storage.Timestamp(time.Now().UTC())}); e != nil {
		t.Fatal(e)
	}
	if e := tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	var count int
	_ = f.store.Pool.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&count)
	if count != 0 {
		t.Fatal("orphan job")
	}
	if _, e := f.store.Q.GetRun(ctx, storage.UUID(id)); e != pgx.ErrNoRows {
		t.Fatal("orphan run", e)
	}
	exact := "0.123456789123"
	numeric, e := storage.Numeric(&exact)
	if e != nil {
		t.Fatal(e)
	}
	var roundtrip string
	if e := f.store.Pool.QueryRow(ctx, "SELECT $1::numeric(24,12)::text", numeric).Scan(&roundtrip); e != nil || roundtrip != exact {
		t.Fatal(roundtrip, e)
	}
	null, _ := storage.Numeric(nil)
	if null.Valid {
		t.Fatal("null became zero")
	}
	at := time.Date(2026, 10, 10, 12, 34, 56, 123456000, time.UTC)
	var result time.Time
	if e := f.store.Pool.QueryRow(ctx, "SELECT $1::timestamptz", storage.Timestamp(at)).Scan(&result); e != nil || !result.Equal(at) {
		t.Fatal(result, e)
	}
}

type mockProvider struct {
	server                     *httptest.Server
	mu                         sync.Mutex
	submitCount, completeCount int
	failLLM                    int
	failImage                  bool
	hold                       bool
	png                        []byte
	models                     []string
}

func mock(t *testing.T) *mockProvider {
	t.Helper()
	m := &mockProvider{}
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.Set(0, 0, color.RGBA{20, 100, 80, 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, im)
	m.png = buf.Bytes()
	m.server = httptest.NewServer(http.HandlerFunc(m.handler))
	t.Cleanup(m.server.Close)
	return m
}
func (m *mockProvider) handler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case r.URL.Path == "/chat/completions":
		m.completeCount++
		if m.failLLM > 0 {
			m.failLLM--
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"error":{"code":503,"message":"mock busy"}}`))
			return
		}
		var body domain.Object
		_ = domain.Decode(mustBody(r), &body)
		_ = json.NewEncoder(w).Encode(domain.Object{"id": "mock-generation", "model": body["model"], "choices": []domain.Object{{"finish_reason": "stop", "message": domain.Object{"content": "A clean product photograph"}}}, "usage": domain.Object{"prompt_tokens": 100, "completion_tokens": 20, "cost": json.Number("0.00002")}})
	case r.URL.Path == "/pricing/estimate":
		_, _ = w.Write([]byte(`{"currency":"USD","total_cost":0.003}`))
	case r.URL.Path == "/status":
		if m.hold {
			_, _ = w.Write([]byte(`{"status":"IN_PROGRESS"}`))
		} else {
			_, _ = w.Write([]byte(`{"status":"COMPLETED"}`))
		}
	case r.URL.Path == "/result":
		_ = json.NewEncoder(w).Encode(domain.Object{"images": []domain.Object{{"url": m.server.URL + "/image", "width": 512, "height": 512}}, "seed": 1, "has_nsfw_concepts": []bool{false}, "timings": domain.Object{"inference": 0.01}, "prompt": "A clean product photograph"})
	case r.URL.Path == "/image":
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(m.png)
	default:
		m.submitCount++
		m.models = append(m.models, strings.TrimPrefix(r.URL.Path, "/"))
		if m.failImage {
			w.WriteHeader(422)
			_, _ = w.Write([]byte(`{"error_type":"bad_request","detail":"mock invalid input"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(domain.Object{"request_id": "mock-fal-request", "status_url": m.server.URL + "/status", "response_url": m.server.URL + "/result"})
	}
}
func mustBody(r *http.Request) []byte { b, _ := io.ReadAll(r.Body); return b }
func testEngine(f *fixture, m *mockProvider) *engine.Engine {
	cfg := f.cfg
	cfg.OpenRouterURL = m.server.URL
	cfg.FalURL = m.server.URL
	cfg.FalPricingURL = m.server.URL + "/pricing"
	cfg.PollInterval = 20 * time.Millisecond
	cfg.RetryBase = 20 * time.Millisecond
	media := artifacts.New(cfg.ArtifactDir)
	media.AllowHTTP = true
	return &engine.Engine{Store: f.store, Provider: providers.New(cfg), Media: media, Config: cfg, Log: f.log, Jitter: func() time.Duration { return 0 }}
}
func startWorker(t *testing.T, f *fixture, e *engine.Engine) *river.Client[pgx.Tx] {
	t.Helper()
	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.Worker{Engine: e})
	client, err := river.NewClient(riverpgxv5.New(f.store.Pool), &river.Config{Workers: workers, Queues: map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}}, Logger: f.log, TestOnly: true, JobTimeout: time.Second, RescueStuckJobsAfter: 2 * time.Second, FetchCooldown: 5 * time.Millisecond, FetchPollInterval: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = client.StopAndCancel(ctx)
	})
	return client
}
func startRun(t *testing.T, f *fixture, key string) string {
	t.Helper()
	start := time.Now()
	status, b := request(t, f, "POST", "/workflows/"+productID+"/runs", domain.Object{"input": domain.Object{"description": "cup"}}, key)
	if status != 202 {
		t.Fatal(status, string(b))
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("run acceptance waited on provider")
	}
	var response struct {
		ID string `json:"run_id"`
	}
	_ = domain.Decode(b, &response)
	return response.ID
}
func waitRun(t *testing.T, f *fixture, id string) *domain.Run {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		r, e := f.store.LoadRun(context.Background(), id)
		if e != nil {
			t.Fatal(e)
		}
		if r.Terminal() {
			return r
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("run timeout", id)
	return nil
}
func TestAsyncWorkerSuccessRetryAndPermanentFailure(t *testing.T) {
	for _, tc := range []struct {
		name             string
		retry, permanent bool
	}{{"success", false, false}, {"transient retry", true, false}, {"permanent failure", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupAPI(t)
			load(t, f)
			m := mock(t)
			if tc.retry {
				m.failLLM = 1
			}
			m.failImage = tc.permanent
			startWorker(t, f, testEngine(f, m))
			id := startRun(t, f, uuid.NewString())
			run := waitRun(t, f, id)
			if tc.permanent {
				if run.Status != "failed" || run.FailedNodeID != "render" || run.Error.Code != "INVALID_INPUT" || run.Error.ProviderCode != "bad_request" {
					t.Fatal(run)
				}
				return
			}
			if run.Status != "succeeded" {
				t.Fatal(run.Error)
			}
			if tc.retry && len(run.Nodes[1].Attempts) != 2 {
				t.Fatal("retry trace")
			}
			if *run.Nodes[2].Cost.Estimated != "0.003000000000" {
				t.Fatal(run.Cost)
			}
			status, b := request(t, f, "GET", "/artifacts/"+run.Nodes[2].Requests[0].ID, nil, "")
			if status != 200 || !bytes.Equal(b, m.png) {
				t.Fatal("artifact not durable", status)
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			if m.submitCount != 1 {
				t.Fatal("duplicate submission")
			}
		})
	}
}

type checkpointStore struct{ *storage.Store }

func (s checkpointStore) SaveRun(ctx context.Context, r *domain.Run) error {
	if e := s.Store.SaveRun(ctx, r); e != nil {
		return e
	}
	if os.Getenv("FLOWKIT_HOLD_CHECKPOINT") == "1" {
		for _, n := range r.Nodes {
			if len(n.Requests) > 0 && n.Node.Kind == "image" && n.Requests[len(n.Requests)-1].Phase == "waiting_provider" {
				select {}
			}
		}
	}
	return nil
}
func TestWorkerProcess(t *testing.T) {
	if os.Getenv("FLOWKIT_HELPER_WORKER") != "1" {
		return
	}
	pool, e := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	cfg := config.Load()
	cfg.OpenRouterURL = os.Getenv("FLOWKIT_MOCK_URL")
	cfg.FalURL = cfg.OpenRouterURL
	cfg.FalPricingURL = cfg.OpenRouterURL + "/pricing"
	cfg.ArtifactDir = os.Getenv("FLOWKIT_ARTIFACT_DIR")
	cfg.OpenRouterKey = "mock"
	cfg.FalKey = "mock"
	cfg.PollInterval = 20 * time.Millisecond
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	store := storage.New(pool)
	media := artifacts.New(cfg.ArtifactDir)
	media.AllowHTTP = true
	eng := &engine.Engine{Store: checkpointStore{store}, Provider: providers.New(cfg), Media: media, Config: cfg, Log: log}
	f := &fixture{store: store, cfg: cfg, log: log}
	startWorker(t, f, eng)
	for {
		time.Sleep(time.Second)
	}
}
func TestKilledWorkerResumesDurableFalTicket(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	m := mock(t)
	m.hold = true
	spawn := func(hold string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestWorkerProcess$", "-test.timeout=120s")
		cmd.Env = append(os.Environ(), "FLOWKIT_HELPER_WORKER=1", "FLOWKIT_MOCK_URL="+m.server.URL, "FLOWKIT_ARTIFACT_DIR="+f.cfg.ArtifactDir, "FLOWKIT_HOLD_CHECKPOINT="+hold)
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
		}
		return cmd
	}
	first := spawn("1")
	t.Cleanup(func() {
		if first.ProcessState == nil {
			_ = first.Process.Kill()
			_ = first.Wait()
		}
	})
	id := startRun(t, f, "restart-demo")
	deadline := time.Now().Add(40 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		run, e := f.store.LoadRun(context.Background(), id)
		if e != nil {
			t.Fatal(e)
		}
		if len(run.Nodes[2].Requests) > 0 && run.Nodes[2].Requests[0].ProviderRequestID == "mock-fal-request" {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		t.Fatal("checkpoint not saved")
	}
	if e := first.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	_ = first.Wait()
	m.mu.Lock()
	m.hold = false
	m.mu.Unlock()
	second := spawn("0")
	defer func() { _ = second.Process.Kill(); _ = second.Wait() }()
	run := waitRun(t, f, id)
	m.mu.Lock()
	defer m.mu.Unlock()
	if run.Status != "succeeded" || m.submitCount != 1 || m.completeCount != 1 || len(run.Nodes[2].Attempts) != 1 || len(run.Nodes[2].Requests) != 1 {
		t.Fatal("restart duplicated work", run.Error, m.submitCount, m.completeCount)
	}
	if *run.Nodes[2].Cost.Estimated != "0.003000000000" {
		t.Fatal("restart duplicated cost")
	}
	t.Log("Killed worker after request-ID commit and before River Work returned; recovered same request with one submit, one domain attempt and one cost record")
}
func TestHotAddModelAndWorkflowWithoutRestart(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	m := mock(t)
	startWorker(t, f, testEngine(f, m))
	newModel := example(t, "models/flux-dev.json")
	newModel["alias"] = "runtime-added"
	newModel["modelId"] = "same-provider/new-compatible-model"
	status, b := request(t, f, "POST", "/models", newModel, "")
	if status != 201 {
		t.Fatal(status, string(b))
	}
	wf := example(t, "workflows/product.json")
	wf["definition"].(map[string]any)["nodes"].([]any)[2].(map[string]any)["modelRef"] = "runtime-added"
	status, b = request(t, f, "PUT", "/workflows/"+productID, wf, "")
	if status != 200 {
		t.Fatal(status, string(b))
	}
	run := waitRun(t, f, startRun(t, f, "hot-model"))
	if run.Status != "succeeded" || run.Nodes[2].Requests[0].ActualModelID != "same-provider/new-compatible-model" {
		t.Fatal(run.Error)
	}
	status, b = request(t, f, "POST", "/workflows", example(t, "workflows/event.json"), "")
	if status != 201 {
		t.Fatal(status, string(b))
	}
	t.Log("Catalog/model reference and second workflow added through HTTP on the same API and worker instances")
}
