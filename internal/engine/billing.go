package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
)

type billingProvider interface {
	Billed(context.Context, domain.ProviderRequest) (providers.BilledCost, *domain.Failure)
}

// ReconcileCosts runs independently after generation. Billing failures never
// change a successful run into a failed run or resend a generation request.
func (e *Engine) ReconcileCosts(ctx context.Context, id string) error {
	if e.Config.BillingDisabled {
		return nil
	}
	r, err := e.Store.LoadRun(ctx, id)
	if err != nil {
		return err
	}
	if !r.Terminal() {
		return nil
	}
	p, ok := e.Provider.(billingProvider)
	if !ok {
		return nil
	}
	now := e.now()
	changed := false
	for ni := range r.Nodes {
		n := &r.Nodes[ni]
		for qi := range n.Requests {
			q := &n.Requests[qi]
			if q.Cost.Actual != nil && q.Cost.ActualStatus == "complete" {
				continue
			}
			if q.Cost.ActualStatus == "unavailable" {
				continue
			}
			if q.Billing.NextCheckAt != nil && now.Before(*q.Billing.NextCheckAt) {
				continue
			}
			changed = true
			if q.ProviderRequestID == "" {
				q.Cost.ActualStatus = "unavailable"
				q.Billing.Error = &domain.Failure{Code: "BILLING_REQUEST_UNKNOWN", Message: "Request has no durable ID; actual cost cannot be verified"}
				continue
			}
			q.Billing.Attempts++
			q.Billing.CheckedAt = &now
			remote, cancel := context.WithTimeout(ctx, 10*time.Second)
			result, f := p.Billed(remote, *q)
			cancel()
			if f == nil && result.Found && Money(result.Amount) == nil {
				f = &domain.Failure{Code: "INVALID_BILLING_RESPONSE", Message: "Billed amount exceeds the supported USD precision/range"}
			}
			q.Billing.Error = f
			if f == nil && result.Found && Money(result.Amount) != nil {
				q.Cost.Reported = Money(result.Amount)
				q.Cost.Actual = Money(result.Amount)
				q.Cost.ActualStatus = "complete"
				q.Cost.ActualSource = result.Source
				q.Billing.Raw = result.Raw
				q.Billing.NextCheckAt = nil
				e.event(r, n, slog.LevelInfo, "billing.reconciled", "provider_request_id", q.ProviderRequestID, "actual_cost_usd", q.Cost.Actual, "billing_checks", q.Billing.Attempts)
				continue
			}
			if f != nil {
				e.event(r, n, slog.LevelError, "billing.error", "provider_request_id", q.ProviderRequestID, "provider_request_ref", q.ID, "error_code", f.Code, "provider_error_code", f.ProviderCode, "provider_error_message", f.ProviderMessage, "provider_http_status", f.HTTPStatus)
			}
			finished := r.CreatedAt
			if r.FinishedAt != nil {
				finished = *r.FinishedAt
			}
			age := now.Sub(finished)
			if age >= 24*time.Hour || (f != nil && !f.Retryable && f.HTTPStatus != 404) {
				q.Cost.ActualStatus = "unavailable"
				q.Billing.NextCheckAt = nil
				if f == nil {
					q.Billing.Error = &domain.Failure{Code: "BILLING_NOT_AVAILABLE", Message: "Provider has not published billing within 24 hours; refresh accounting to retry"}
				}
			} else {
				delay := 30 * time.Second
				if q.Billing.Attempts > 2 {
					delay = 2 * time.Minute
				}
				if q.Billing.Attempts > 5 {
					delay = 10 * time.Minute
				}
				if q.Billing.Attempts > 10 {
					delay = time.Hour
				}
				if f != nil && time.Duration(f.RetryAfterMS)*time.Millisecond > delay {
					delay = time.Duration(f.RetryAfterMS) * time.Millisecond
				}
				next := now.Add(delay)
				q.Billing.NextCheckAt = &next
				q.Cost.ActualStatus = "pending"
			}
		}
	}
	if !changed {
		return nil
	}
	return e.save(ctx, r)
}

func (e *Engine) RefreshCosts(ctx context.Context, r *domain.Run) error {
	now := e.now()
	for ni := range r.Nodes {
		for qi := range r.Nodes[ni].Requests {
			q := &r.Nodes[ni].Requests[qi]
			if q.Cost.Actual != nil && q.Cost.ActualStatus == "complete" {
				continue
			}
			if q.ProviderRequestID == "" {
				continue
			}
			q.Cost.ActualStatus = "pending"
			q.Billing.NextCheckAt = &now
			q.Billing.Error = nil
			q.Billing.Attempts = 0
		}
	}
	return e.save(ctx, r)
}
