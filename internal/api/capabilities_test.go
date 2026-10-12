package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
)

func TestDisabledBillingIsAdvertisedAndCannotScheduleQueries(t *testing.T) {
	s := &Server{Config: config.Config{BillingDisabled: true}, Log: telemetry.Logger("test", "test")}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/capabilities", 200}, {"POST", "/runs/10000000-0000-4000-8000-000000000001/costs/refresh", 409},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatal(w.Code, w.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if tc.method == "GET" && body["billing_enabled"] != false {
			t.Fatal(body)
		}
		if tc.method == "POST" && body["error"].(map[string]any)["code"] != "BILLING_DISABLED" {
			t.Fatal(body)
		}
	}
}
