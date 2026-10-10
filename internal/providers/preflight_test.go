package providers

import (
	"context"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicModelPriceAndMissingModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"vendor/model","pricing":{"prompt":"0.0000001","completion":"0.0000005"}}]}`))
	}))
	defer server.Close()
	cfg := config.Load()
	cfg.OpenRouterURL = server.URL
	c := New(cfg)
	m, e := c.PublicModel(context.Background(), domain.Model{ModelID: "vendor/model"})
	if e != nil || m.Pricing.InputPerMillion != "0.1" || m.Pricing.OutputPerMillion != "0.5" {
		t.Fatal(m, e)
	}
	if _, e := c.PublicModel(context.Background(), domain.Model{ModelID: "missing"}); e == nil {
		t.Fatal("missing model guessed")
	}
}
