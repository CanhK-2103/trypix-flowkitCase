package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/shopspring/decimal"
)

type BilledCost struct {
	Amount string
	Source string
	Raw    json.RawMessage
	Found  bool
}

// Billed reads provider accounting for this exact request. Unit price estimates
// and workspace totals are never used as an actual per-request charge.
func (c *Client) Billed(ctx context.Context, q domain.ProviderRequest) (BilledCost, *domain.Failure) {
	if q.ProviderRequestID == "" {
		return BilledCost{}, &domain.Failure{Code: "BILLING_REQUEST_UNKNOWN", Message: "No durable request ID available for billing reconciliation"}
	}
	if q.Provider == "openrouter" {
		data, _, f := c.request(ctx, "GET", c.Config.OpenRouterURL+"/generation?id="+url.QueryEscape(q.ProviderRequestID), c.Config.OpenRouterKey, nil, false, false)
		if f != nil {
			if f.HTTPStatus == 404 {
				return BilledCost{}, nil
			}
			return BilledCost{}, f
		}
		var body struct {
			Data struct {
				ID    string      `json:"id"`
				Total json.Number `json:"total_cost"`
			} `json:"data"`
		}
		if domain.Decode(data, &body) != nil || body.Data.ID != q.ProviderRequestID {
			return BilledCost{}, billingInvalid()
		}
		if body.Data.Total == "" {
			return BilledCost{}, nil
		}
		amount, e := decimal.NewFromString(body.Data.Total.String())
		if e != nil || amount.IsNegative() {
			return BilledCost{}, billingInvalid()
		}
		return BilledCost{Amount: amount.StringFixed(12), Source: "openrouter generation.total_cost", Raw: data, Found: true}, nil
	}
	if q.Provider != "fal" {
		return BilledCost{}, billingInvalid()
	}
	key := c.Config.FalBillingKey
	if key == "" {
		key = c.Config.FalKey
	}
	endpoint := c.Config.FalBillingURL
	if endpoint == "" {
		endpoint = "https://api.fal.ai/v1/models/billing-events"
	}
	params := url.Values{"request_id": {q.ProviderRequestID}, "endpoint_id": {q.RequestedModelID}, "source": {"billed"}, "limit": {"100"}, "start": {q.CreatedAt.Add(-time.Hour).UTC().Format(time.RFC3339)}}
	sum := decimal.Zero
	matched := []domain.Object{}
	seen := map[string]bool{}
	for page := 0; page < 10; page++ {
		data, _, f := c.request(ctx, "GET", endpoint+"?"+params.Encode(), key, nil, false, true)
		if f != nil {
			return BilledCost{}, f
		}
		var body struct {
			Events []domain.Object `json:"billing_events"`
			Next   *string         `json:"next_cursor"`
			More   bool            `json:"has_more"`
		}
		if domain.Decode(data, &body) != nil || body.Events == nil {
			return BilledCost{}, billingInvalid()
		}
		for _, event := range body.Events {
			if event["request_id"] != q.ProviderRequestID || event["endpoint_id"] != q.RequestedModelID {
				continue
			}
			var amount decimal.Decimal
			var err error
			if value, ok := event["cost_total"]; ok && value != nil {
				amount, err = decimal.NewFromString(fmt.Sprint(value))
			} else if value, ok := event["cost_estimate_nano_usd"]; ok && value != nil {
				amount, err = decimal.NewFromString(fmt.Sprint(value))
				amount = amount.Div(decimal.NewFromInt(1000000000))
			} else {
				return BilledCost{}, billingInvalid()
			}
			if err != nil || amount.IsNegative() {
				return BilledCost{}, billingInvalid()
			}
			if currency, ok := event["currency"]; ok && currency != "USD" {
				return BilledCost{}, billingInvalid()
			}
			sum = sum.Add(amount)
			matched = append(matched, event)
		}
		if !body.More {
			return BilledCost{Amount: sum.StringFixed(12), Source: "fal billing-events source=billed cost_total", Raw: domain.JSON(matched), Found: len(matched) > 0}, nil
		}
		if body.Next == nil || *body.Next == "" || seen[*body.Next] {
			return BilledCost{}, billingInvalid()
		}
		seen[*body.Next] = true
		params.Set("cursor", *body.Next)
	}
	return BilledCost{}, &domain.Failure{Code: "BILLING_PAGE_LIMIT", Message: "Billing results exceed the safe pagination limit"}
}
func billingInvalid() *domain.Failure {
	return &domain.Failure{Code: "INVALID_BILLING_RESPONSE", Message: "Provider billing response has no trustworthy matching USD amount"}
}
