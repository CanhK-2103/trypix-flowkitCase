package providers

import (
	"context"
	"fmt"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/shopspring/decimal"
	"io"
	"net/http"
	"time"
)

// PublicModel verifies the chosen identifier and current list prices without generating tokens.
func (c *Client) PublicModel(ctx context.Context, m domain.Model) (domain.Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "GET", c.Config.OpenRouterURL+"/models", nil)
	if e != nil {
		return m, e
	}
	res, e := c.HTTP.Do(req)
	if e != nil {
		return m, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return m, fmt.Errorf("public Models API HTTP %d", res.StatusCode)
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 16*1024*1024+1))
	if e != nil || len(data) > 16*1024*1024 {
		return m, fmt.Errorf("model metadata response unreadable or too large")
	}
	var body struct {
		Data []struct {
			ID      string                              `json:"id"`
			Pricing struct{ Prompt, Completion string } `json:"pricing"`
		} `json:"data"`
	}
	if e := domain.Decode(data, &body); e != nil {
		return m, e
	}
	for _, item := range body.Data {
		if item.ID == m.ModelID {
			input, e1 := decimal.NewFromString(item.Pricing.Prompt)
			output, e2 := decimal.NewFromString(item.Pricing.Completion)
			if e1 != nil || e2 != nil || input.IsNegative() || output.IsNegative() {
				return m, fmt.Errorf("missing trustworthy public token price")
			}
			m.Pricing = &domain.Pricing{Currency: "USD", Unit: "tokens", InputPerMillion: input.Mul(decimal.NewFromInt(1000000)).String(), OutputPerMillion: output.Mul(decimal.NewFromInt(1000000)).String(), Source: "OpenRouter public Models API", UpdatedAt: time.Now().UTC()}
			return m, nil
		}
	}
	return m, fmt.Errorf("chosen model %s is absent from public Models API; no substitution", m.ModelID)
}
