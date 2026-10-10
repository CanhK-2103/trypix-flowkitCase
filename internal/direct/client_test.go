package direct

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDirectProviderSequence(t *testing.T) {
	var server *httptest.Server
	var submitted int
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/chat/completions":
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-openrouter" {
				t.Error("incorrect OpenRouter method/auth")
			}
			var body LLMRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Model != "test-llm" || body.Provider["allow_fallbacks"] || len(body.Messages) != 2 {
				t.Error("incorrect LLM request")
			}
			_, _ = w.Write([]byte(`{"id":"llm-id","model":"test-llm","choices":[{"message":{"role":"assistant","content":"A black mug in a studio"},"finish_reason":"stop"}]}`))
		case "/fal-ai/flux/schnell":
			submitted++
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Key test-fal" {
				t.Error("incorrect fal method/auth")
			}
			var body ImageRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Prompt != "A black mug in a studio" || body.NumImages != 1 || body.ImageSize.Width != 512 {
				t.Error("incorrect image payload")
			}
			// Retrieval URLs intentionally differ from the submitted model path.
			_ = json.NewEncoder(w).Encode(QueueHandle{RequestID: "fal-id", StatusURL: server.URL + "/tracking/status", ResponseURL: server.URL + "/tracking/result"})
		case "/tracking/status":
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Key test-fal" {
				t.Error("incorrect status method/auth")
			}
			_, _ = w.Write([]byte(`{"status":"COMPLETED"}`))
		case "/tracking/result":
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Key test-fal" {
				t.Error("incorrect result method/auth")
			}
			_, _ = w.Write([]byte(`{"images":[{"url":"https://example.invalid/image.jpg","width":512,"height":512,"content_type":"image/jpeg"}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := New("test-openrouter", "test-fal")
	c.OpenRouterBase, c.FalBase = server.URL, server.URL
	ctx := context.Background()
	response, err := c.GeneratePrompt(ctx, PromptRequest("test-llm", "black mug"))
	if err != nil {
		t.Fatal(err)
	}
	handle, err := c.Submit(ctx, "fal-ai/flux/schnell", ImagePayload(response.Choices[0].Message.Content))
	if err != nil {
		t.Fatal(err)
	}
	status, err := c.Status(ctx, handle)
	if err != nil || status.Status != "COMPLETED" {
		t.Fatalf("status: %#v, %v", status, err)
	}
	result, err := c.Result(ctx, handle)
	if err != nil || len(result.Images) != 1 {
		t.Fatalf("result: %#v, %v", result, err)
	}
	if submitted != 1 {
		t.Fatalf("expected one submit, got %d", submitted)
	}
}

func TestQueueStatusRejectsTerminalErrorAndUnknownState(t *testing.T) {
	for _, body := range []string{`{"status":"COMPLETED","error":"invalid input","error_type":"validation_error"}`, `{"status":"UNRECOGNIZED"}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			c := New("", "test-fal")
			c.FalBase = server.URL
			_, err := c.Status(context.Background(), QueueHandle{RequestID: "id", StatusURL: server.URL + "/status", ResponseURL: server.URL + "/result"})
			if err == nil {
				t.Fatal("expected failure")
			}
		})
	}
}

func TestProviderHTTPErrorIsReturnedWithoutResubmitting(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"detail":"busy"}`))
	}))
	defer server.Close()
	c := New("", "test-fal")
	c.FalBase = server.URL
	_, err := c.Submit(context.Background(), "fal-ai/flux/schnell", ImagePayload("mug"))
	var providerError *HTTPError
	if !errors.As(err, &providerError) || providerError.Status != 503 {
		t.Fatalf("expected typed 503, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one call, got %d", calls)
	}
}

func TestQueueURLCannotLeakCredentialsToAnotherOrigin(t *testing.T) {
	c := New("", "secret")
	for _, address := range []string{"https://evil.invalid/requests/1", "http://queue.fal.run/requests/1", "https://queue.fal.run:444/requests/1", "https://user@queue.fal.run/requests/1", "/requests/1"} {
		if err := c.ValidateHandle(QueueHandle{RequestID: "id", StatusURL: address, ResponseURL: "https://queue.fal.run/requests/1"}); err == nil {
			t.Errorf("accepted untrusted URL %s", address)
		}
	}
}

func TestPromptResponseMustHaveModelAndCompleteContent(t *testing.T) {
	for _, body := range []string{`{"choices":[]}`, `{"model":"llm","choices":[{"message":{"content":"partial"},"finish_reason":"length"}]}`, `{"error":{"code":500,"message":"failed"}}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			c := New("test", "")
			c.OpenRouterBase = server.URL
			if _, err := c.GeneratePrompt(context.Background(), PromptRequest("llm", "mug")); err == nil {
				t.Fatal("expected failure")
			}
		})
	}
}
