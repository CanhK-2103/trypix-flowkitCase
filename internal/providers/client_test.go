package providers

import (
	"context"
	"encoding/json"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mustRead(t *testing.T, r *http.Request) []byte {
	t.Helper()
	b, e := io.ReadAll(r.Body)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestCompletionContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("wrong route/auth")
		}
		var body domain.Object
		if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
			t.Error(e)
		}
		if body["model"] != "provider/new-model" || body["stream"] != false || body["max_completion_tokens"] != float64(300) {
			t.Error(body)
		}
		if _, ok := body["max_tokens"]; ok {
			t.Error("deprecated parameter")
		}
		if _, ok := body["usage"]; ok {
			t.Error("deprecated usage toggle")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"gen-1","model":"provider/actual-model","choices":[{"finish_reason":"stop","message":{"content":" studio photo "}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"cost":0.000000123456}}`))
	}))
	defer server.Close()
	cfg := config.Load()
	cfg.OpenRouterURL = server.URL
	cfg.OpenRouterKey = "test-key"
	result, f := New(cfg).Complete(context.Background(), domain.Model{ModelID: "provider/new-model"}, "cup", domain.Object{"max_completion_tokens": 300})
	if f != nil || result.Prompt != "studio photo" || result.Model != "provider/actual-model" {
		t.Fatal(result, f)
	}
	if result.Usage["cost"].(json.Number).String() != "0.000000123456" {
		t.Fatal("lost decimal precision")
	}
}
func TestRefusalEmptyAndTruncatedCompletion(t *testing.T) {
	cases := []struct{ name, body, code string }{{"refusal", `{"refusal":"Cannot comply","content":null}`, "PROVIDER_CONTENT_REJECTED"}, {"empty", `{"content":" "}`, "INVALID_PROVIDER_RESPONSE"}, {"truncated", `{"content":"partial"}`, "INVALID_PROVIDER_RESPONSE"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				finish := "stop"
				if tc.name == "truncated" {
					finish = "length"
				}
				_, _ = w.Write([]byte(`{"id":"gen","model":"model","choices":[{"finish_reason":"` + finish + `","message":` + tc.body + `}],"usage":{"cost":0.01}}`))
			}))
			defer server.Close()
			cfg := config.Load()
			cfg.OpenRouterURL = server.URL
			result, f := New(cfg).Complete(context.Background(), domain.Model{ModelID: "model"}, "input", nil)
			if f == nil || f.Code != tc.code || result.ID != "gen" {
				t.Fatal(result, f)
			}
		})
	}
}
func TestQueueContractAndErrorCompletion(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Key test-key" || r.Header.Get("X-Fal-No-Retry") != "1" || r.Header.Get("x-app-fal-disable-fallback") != "true" {
			t.Error("fal headers")
		}
		switch r.URL.Path {
		case "/vendor/new-model":
			var body domain.Object
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["prompt"] != "photo" || body["sync_mode"] != false {
				t.Error(body)
			}
			_ = json.NewEncoder(w).Encode(domain.Object{"request_id": "ticket", "status_url": server.URL + "/status-from-provider", "response_url": server.URL + "/result-from-provider"})
		case "/status-from-provider":
			_, _ = w.Write([]byte(`{"status":"COMPLETED","error":"runner failed","error_type":"runner_server_error"}`))
		case "/result-from-provider":
			_, _ = w.Write([]byte(`{"images":[{"url":"https://cdn.test/image.png","width":512,"height":512}]}`))
		default:
			t.Error("guessed queue path", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	cfg := config.Load()
	cfg.FalURL = server.URL
	cfg.FalKey = "test-key"
	client := New(cfg)
	ticket, f := client.Submit(context.Background(), domain.Model{ModelID: "vendor/new-model"}, "photo", domain.Object{"num_images": 1})
	if f != nil || ticket.ID != "ticket" {
		t.Fatal(ticket, f)
	}
	state, f := client.Status(context.Background(), ticket)
	if f != nil || state.Error == nil || state.Error.Code != "PROVIDER_5XX" || !state.Error.Retryable {
		t.Fatal(state, f)
	}
	result, f := client.Result(context.Background(), ticket)
	if f != nil || result.Width != 512 {
		t.Fatal(result, f)
	}
}
func TestTypedProviderErrors(t *testing.T) {
	for _, tc := range []struct {
		status     int
		body, code string
		retry      bool
	}{{401, `{"error":{"message":"bad key"}}`, "PROVIDER_AUTH_ERROR", false}, {403, `{"error":{"metadata":{"error_type":"refusal"},"message":"policy"}}`, "PROVIDER_CONTENT_REJECTED", false}, {422, `{"detail":[{"type":"validation","msg":"bad prompt"}]}`, "INVALID_INPUT", false}, {429, `{}`, "PROVIDER_RATE_LIMIT", true}, {503, `{"error_type":"runner_disconnected"}`, "PROVIDER_5XX", true}, {524, `{}`, "PROVIDER_TIMEOUT", true}, {529, `{}`, "PROVIDER_5XX", true}} {
		headers := http.Header{"Retry-After": []string{"4"}}
		failure := classify(tc.status, []byte(tc.body), headers)
		if failure.Code != tc.code || failure.Retryable != tc.retry || failure.RetryAfterMS != 4000 {
			t.Fatal(failure)
		}
	}
}
func TestEchoedKeyIsRedactedFromErrorLogFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"message":"fixture-secret-key is unauthorized"}}`))
	}))
	defer server.Close()
	cfg := config.Load()
	cfg.OpenRouterURL = server.URL
	cfg.OpenRouterKey = "fixture-secret-key"
	_, f := New(cfg).Complete(context.Background(), domain.Model{ModelID: "model"}, "prompt", nil)
	if f == nil || strings.Contains(f.ProviderMessage, cfg.OpenRouterKey) {
		t.Fatal("credential leaked")
	}
}
func TestResponseLossNotResubmitted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker := w.(http.Hijacker)
		conn, _, _ := hijacker.Hijack()
		conn.Close()
	}))
	defer server.Close()
	cfg := config.Load()
	cfg.OpenRouterURL = server.URL
	_, f := New(cfg).Complete(context.Background(), domain.Model{ModelID: "model"}, "input", nil)
	if f == nil || f.Code != "PROVIDER_SUBMISSION_UNKNOWN" || f.Retryable {
		t.Fatal(f)
	}
}
func TestHistoricalPriceSingleCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body domain.Object
		_ = domain.Decode(mustRead(t, r), &body)
		if body["estimate_type"] != "historical_api_price" {
			t.Error(body)
		}
		endpoints := body["endpoints"].(map[string]any)
		if len(endpoints) != 1 || endpoints["vendor/new-model"].(map[string]any)["call_quantity"] != json.Number("1") {
			t.Error(body)
		}
		_, _ = w.Write([]byte(`{"currency":"USD","total_cost":0.003}`))
	}))
	defer server.Close()
	cfg := config.Load()
	cfg.FalPricingURL = server.URL
	price, f := New(cfg).Estimate(context.Background(), domain.Model{ModelID: "vendor/new-model"})
	if f != nil || price != "0.003" {
		t.Fatal(price, f)
	}
}
