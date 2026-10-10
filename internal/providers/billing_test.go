package providers

import (
	"context"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFalBilledExactRequestDiscountAndPagination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/billing-events" || r.URL.Query().Get("request_id") != "fal-1" || r.URL.Query().Get("endpoint_id") != "fal-ai/flux/schnell" || r.URL.Query().Get("source") != "billed" || r.Header.Get("Authorization") != "Key accounting-key" {
			t.Errorf("bad billing request %s", r.URL)
		}
		if calls == 1 {
			w.Write([]byte(`{"billing_events":[{"request_id":"other","endpoint_id":"fal-ai/flux/schnell","cost_total":50}],"has_more":true,"next_cursor":"next"}`))
			return
		}
		if r.URL.Query().Get("cursor") != "next" {
			t.Error("cursor not followed")
		}
		w.Write([]byte(`{"billing_events":[{"request_id":"fal-1","endpoint_id":"fal-ai/flux/schnell","cost_total":0.0027,"unit_price":0.003,"cost_discount":0.0003}],"has_more":false,"next_cursor":null}`))
	}))
	defer server.Close()
	c := New(config.Config{FalKey: "generation-key", FalBillingKey: "accounting-key", FalBillingURL: server.URL + "/billing-events"})
	cost, f := c.Billed(context.Background(), domain.ProviderRequest{Provider: "fal", ProviderRequestID: "fal-1", RequestedModelID: "fal-ai/flux/schnell", CreatedAt: time.Now()})
	if f != nil || !cost.Found || cost.Amount != "0.002700000000" || calls != 2 {
		t.Fatal(cost, f, calls)
	}
}

func TestFalBilledMissingInvalidAndUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name, body   string
		status       int
		found        bool
		amount, code string
	}{
		{"delayed", `{"billing_events":[],"has_more":false}`, 200, false, "", ""},
		{"other request", `{"billing_events":[{"request_id":"other","endpoint_id":"model","cost_total":3}],"has_more":false}`, 200, false, "", ""},
		{"nano", `{"billing_events":[{"request_id":"id","endpoint_id":"model","cost_estimate_nano_usd":1234567}],"has_more":false}`, 200, true, "0.001234567000", ""},
		{"zero", `{"billing_events":[{"request_id":"id","endpoint_id":"model","cost_total":0}],"has_more":false}`, 200, true, "0.000000000000", ""},
		{"no bill", `{"billing_events":[{"request_id":"id","endpoint_id":"model","unit_price":0.003,"quantity":1}],"has_more":false}`, 200, false, "", "INVALID_BILLING_RESPONSE"},
		{"bad currency", `{"billing_events":[{"request_id":"id","endpoint_id":"model","cost_total":1,"currency":"EUR"}],"has_more":false}`, 200, false, "", "INVALID_BILLING_RESPONSE"},
		{"permission", `{"error":{"code":"permission","message":"Billing access denied"}}`, 403, false, "", "PROVIDER_AUTH_ERROR"},
		{"5xx", `{"error":{"message":"busy"}}`, 503, false, "", "PROVIDER_5XX"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) }))
			defer server.Close()
			c := New(config.Config{FalKey: "mock", FalBillingURL: server.URL})
			cost, f := c.Billed(context.Background(), domain.ProviderRequest{Provider: "fal", ProviderRequestID: "id", RequestedModelID: "model", CreatedAt: time.Now()})
			if tc.code != "" {
				if f == nil || f.Code != tc.code {
					t.Fatal(cost, f)
				}
				return
			}
			if f != nil || cost.Found != tc.found || (tc.found && cost.Amount != tc.amount) {
				t.Fatal(cost, f)
			}
		})
	}
}
func TestOpenRouterBilledFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generation" || r.URL.Query().Get("id") != "gen-1" {
			t.Error(r.URL)
		}
		w.Write([]byte(`{"data":{"id":"gen-1","total_cost":0.000012345678}}`))
	}))
	defer server.Close()
	c := New(config.Config{OpenRouterURL: server.URL, OpenRouterKey: "mock"})
	cost, f := c.Billed(context.Background(), domain.ProviderRequest{Provider: "openrouter", ProviderRequestID: "gen-1"})
	if f != nil || !cost.Found || cost.Amount != "0.000012345678" {
		t.Fatal(cost, f)
	}
}
