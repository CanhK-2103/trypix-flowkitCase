package engine

import (
	"fmt"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/shopspring/decimal"
)

func Money(s string) *string {
	d, e := decimal.NewFromString(s)
	if e != nil || d.IsNegative() || d.GreaterThan(decimal.RequireFromString("999999999999.999999999999")) {
		return nil
	}
	out := d.StringFixed(12)
	return &out
}
func ZeroCost() domain.Cost {
	return domain.Cost{Estimated: Money("0"), Reported: Money("0"), Actual: Money("0"), ActualStatus: "complete", Status: "complete", Source: "local", ActualSource: "local"}
}
func SumCosts(costs []domain.Cost) domain.Cost {
	estimate, reported, actual := decimal.Zero, decimal.Zero, decimal.Zero
	knownE, knownR := false, false
	completeE := true
	completeA, unavailableA, pendingA := true, false, false
	for _, c := range costs {
		if c.Estimated == nil {
			completeE = false
		} else {
			d, _ := decimal.NewFromString(*c.Estimated)
			estimate = estimate.Add(d)
			knownE = true
		}
		if c.Reported != nil {
			d, _ := decimal.NewFromString(*c.Reported)
			reported = reported.Add(d)
			knownR = true
		}
		if c.Actual == nil || c.ActualStatus != "complete" {
			completeA = false
		} else {
			d, _ := decimal.NewFromString(*c.Actual)
			actual = actual.Add(d)
		}
		if c.ActualStatus == "unavailable" {
			unavailableA = true
		} else if c.Actual == nil || c.ActualStatus != "complete" {
			pendingA = true
		}
		if c.Status == "partial" {
			completeE = false
		}
	}
	out := domain.Cost{Status: "complete", ActualStatus: "pending"}
	if completeA {
		out.Actual = Money(actual.String())
		out.ActualStatus = "complete"
	} else if unavailableA && !pendingA {
		out.ActualStatus = "unavailable"
	}
	if knownE || len(costs) == 0 {
		out.Estimated = Money(estimate.String())
	}
	if knownR || len(costs) == 0 {
		out.Reported = Money(reported.String())
	}
	if !completeE {
		out.Status = "partial"
	}
	return out
}
func TokenCost(m domain.Model, usage domain.Object) domain.Cost {
	out := domain.Cost{Status: "partial"}
	if v, ok := usage["cost"]; ok {
		out.Reported = Money(fmt.Sprint(v))
		out.Actual = out.Reported
		if out.Actual != nil {
			out.ActualStatus = "complete"
			out.ActualSource = "openrouter usage.cost"
		}
	}
	if out.Actual == nil {
		out.ActualStatus = "pending"
	}
	if m.Pricing == nil {
		return out
	}
	p := m.Pricing
	out.Source = p.Source
	in, e1 := decimal.NewFromString(fmt.Sprint(usage["prompt_tokens"]))
	output, e2 := decimal.NewFromString(fmt.Sprint(usage["completion_tokens"]))
	rateIn, e3 := decimal.NewFromString(p.InputPerMillion)
	rateOut, e4 := decimal.NewFromString(p.OutputPerMillion)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return out
	}
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		if cached, err := decimal.NewFromString(fmt.Sprint(details["cached_tokens"])); err == nil && !cached.IsZero() {
			return out
		}
	}
	out.Estimated = Money(in.Mul(rateIn).Add(output.Mul(rateOut)).Div(decimal.NewFromInt(1000000)).String())
	out.Status = "complete"
	return out
}
func CatalogImageCost(m domain.Model, width, height int) domain.Cost {
	out := domain.Cost{Status: "partial"}
	if m.Pricing == nil {
		return out
	}
	p := m.Pricing
	rate, e := decimal.NewFromString(p.UnitPrice)
	if e != nil {
		return out
	}
	units := decimal.NewFromInt(1)
	switch p.Unit {
	case "image":
	case "megapixel":
		if width <= 0 || height <= 0 {
			return out
		}
		units = decimal.NewFromInt(int64(width) * int64(height)).Div(decimal.NewFromInt(1000000)).Ceil()
	default:
		return out
	}
	out.Estimated = Money(rate.Mul(units).String())
	out.Status = "complete"
	out.Source = p.Source
	return out
}
