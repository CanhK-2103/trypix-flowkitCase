package integration

import (
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"testing"
)

func TestFallbackSnapshotsAndWorkerWithRealPostgres(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	w := example(t, "workflows/product.json")
	image := w["definition"].(map[string]any)["nodes"].([]any)[2].(map[string]any)
	image["fallbackModelRef"] = "unregistered"
	status, _ := request(t, f, "PUT", "/workflows/"+productID, w, "")
	if status != 422 {
		t.Fatal("missing secondary saved", status)
	}
	image["fallbackModelRef"] = "flux-dev"
	status, body := request(t, f, "PUT", "/workflows/"+productID, w, "")
	if status != 200 {
		t.Fatal(status, string(body))
	}
	id := startRun(t, f, "fallback-integration")
	// A catalog edit after acceptance must not replace the fallback snapshot.
	status, body = request(t, f, "PATCH", "/models/flux-dev", domain.Object{"enabled": false}, "")
	if status != 200 {
		t.Fatal(status, string(body))
	}
	m := mock(t)
	m.failModels = map[string]int{"fal-ai/flux/schnell": 1}
	startWorker(t, f, testEngine(f, m))
	run := waitRun(t, f, id)
	n := run.Nodes[2]
	if run.Status != "succeeded" || n.ActiveModelRef != "flux-dev" || n.Fallback == nil || !n.FallbackModel.Enabled || n.Model.ModelID != "fal-ai/flux/schnell" || n.Requests[1].ActualModelID != "fal-ai/flux/dev" {
		t.Fatal("wrong durable fallback", run.Error, n)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.submitCount != 2 || len(m.models) != 2 || m.models[0] != "fal-ai/flux/schnell" || m.models[1] != "fal-ai/flux/dev" || len(n.Attempts) != 2 || len(n.Requests) != 2 {
		t.Fatal("fallback dispatch/ledger", m.models)
	}
	if run.InitialEstimate == nil || n.Fallback.Reason.Code != "PROVIDER_5XX" || run.Cost.Actual != nil {
		t.Fatal("cost/failure history lost")
	}
	status, body = request(t, f, "GET", "/runs/"+id, nil, "")
	if status != 200 {
		t.Fatal(status, string(body))
	}
	var fetched domain.Run
	if domain.Decode(body, &fetched) != nil || fetched.Nodes[2].Fallback == nil || fetched.Nodes[2].ActiveModelRef != "flux-dev" {
		t.Fatal("API lost fallback trace")
	}
}
