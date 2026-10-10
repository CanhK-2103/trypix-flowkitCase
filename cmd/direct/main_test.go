package main

import (
	"encoding/json"
	"flag"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectExperimentWithLocalHTTPFixtures(t *testing.T) {
	var server *httptest.Server
	calls := []string{}
	path := filepath.Join(t.TempDir(), "evidence.json")
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"openai/gpt-6-luna","pricing":{"prompt":"0.0000001","completion":"0.0000005"}}]}`))
		case "/pricing/estimate":
			_, _ = w.Write([]byte(`{"currency":"USD","total_cost":0.003}`))
		case "/chat/completions":
			_, _ = w.Write([]byte(`{"id":"test-gen","model":"openai/gpt-6-luna","choices":[{"finish_reason":"stop","message":{"content":"Product photo"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"cost":0.00002}}`))
		case "/fal-ai/flux/schnell":
			_ = json.NewEncoder(w).Encode(domain.Object{"request_id": "test-ticket", "status_url": server.URL + "/status", "response_url": server.URL + "/result"})
		case "/status":
			var checkpoint domain.Object
			data, e := os.ReadFile(path)
			if e != nil {
				t.Error(e)
			}
			_ = domain.Decode(data, &checkpoint)
			if checkpoint["phase"] != "fal_waiting" {
				t.Error("polled before durable ticket")
			}
			_, _ = w.Write([]byte(`{"status":"COMPLETED"}`))
		case "/result":
			_, _ = w.Write([]byte(`{"images":[{"url":"https://mock.test/image.png","width":512,"height":512}]}`))
		default:
			t.Error("unexpected external request", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("OPENROUTER_API_KEY", "fixture-key")
	t.Setenv("FAL_KEY", "fixture-key")
	t.Setenv("OPENROUTER_BASE_URL", server.URL)
	t.Setenv("FAL_QUEUE_BASE_URL", server.URL)
	t.Setenv("FAL_PRICING_BASE_URL", server.URL+"/pricing")
	t.Chdir("../..")
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args = oldArgs; flag.CommandLine = oldFlags })
	os.Args = []string{"direct", "-evidence", path}
	flag.CommandLine = flag.NewFlagSet("direct", flag.ContinueOnError)
	if e := execute(); e != nil {
		t.Fatal(e)
	}
	if len(calls) != 6 {
		t.Fatal(calls)
	}
	before := len(calls)
	flag.CommandLine = flag.NewFlagSet("direct", flag.ContinueOnError)
	if e := execute(); e == nil {
		t.Fatal("duplicate paid experiment allowed")
	}
	if len(calls) != before {
		t.Fatal("rerun sent requests")
	}
}
