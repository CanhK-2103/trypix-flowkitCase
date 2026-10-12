package engine

import (
	"context"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"testing"
	"time"
)

type billingFake struct {
	*fake
	checks    int
	failure   *domain.Failure
	available bool
}

func (p *billingFake) Billed(_ context.Context, q domain.ProviderRequest) (providers.BilledCost, *domain.Failure) {
	p.checks++
	if q.ProviderRequestID != "fal-1" {
		panic("wrong request reconciled")
	}
	return providers.BilledCost{Found: p.available, Amount: "0.0027", Source: "mock billed", Raw: domain.JSON(domain.Object{"request_id": q.ProviderRequestID, "cost_total": "0.0027"})}, p.failure
}
func TestDisabledBillingPreservesCompletedGenerationWithoutQueries(t *testing.T) {
	e, s, p, now := setup(t)
	drive(t, e, s, now)
	b := &billingFake{fake: p, available: true}
	e.Provider = b
	e.Config.BillingDisabled = true
	before := string(domain.JSON(s.run))
	if err := e.ReconcileCosts(context.Background(), s.run.ID); err != nil {
		t.Fatal(err)
	}
	if b.checks != 0 || string(domain.JSON(s.run)) != before || s.run.Status != "succeeded" {
		t.Fatal("Disabled accounting changed run or contacted provider")
	}
}
func TestDelayedActualTotalResumeAndNoDoubleCharge(t *testing.T) {
	e, s, p, now := setup(t)
	drive(t, e, s, now)
	b := &billingFake{fake: p}
	e.Provider = b
	if s.run.Cost.Actual != nil {
		t.Fatal("partial report exposed as actual total")
	}
	ctx := context.Background()
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil {
		t.Fatal(err)
	}
	if s.run.Cost.Actual != nil || s.run.Nodes[2].Requests[0].Billing.NextCheckAt == nil {
		t.Fatal(s.run.Cost)
	}
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil || b.checks != 1 {
		t.Fatal("due time ignored", err)
	}
	*now = now.Add(time.Minute)
	b.available = true
	// A restarted engine loads durable accounting state and only reads billing.
	restarted := &Engine{Store: s, Provider: b, Now: e.Now}
	if err := restarted.ReconcileCosts(ctx, s.run.ID); err != nil {
		t.Fatal(err)
	}
	if s.run.Status != "succeeded" || s.run.Cost.ActualStatus != "complete" || s.run.Cost.Actual == nil || *s.run.Cost.Actual != "0.002725000000" {
		t.Fatal(s.run.Cost)
	}
	if err := restarted.ReconcileCosts(ctx, s.run.ID); err != nil || b.checks != 2 || p.submitCount != 1 || p.completeCount != 1 {
		t.Fatal("billing created duplicate charge", err, b.checks)
	}
	if *s.run.Nodes[2].Cost.Estimated != "0.003000000000" || *s.run.Nodes[2].Cost.Actual != "0.002700000000" {
		t.Fatal("estimate overwritten", s.run.Nodes[2].Cost)
	}
}
func TestBillingPermissionFailureDoesNotFailRunAndCanRefresh(t *testing.T) {
	e, s, p, now := setup(t)
	drive(t, e, s, now)
	b := &billingFake{fake: p, failure: &domain.Failure{Code: "PROVIDER_AUTH_ERROR", HTTPStatus: 403, Message: "Billing permission missing"}}
	e.Provider = b
	ctx := context.Background()
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil {
		t.Fatal(err)
	}
	if s.run.Status != "succeeded" || s.run.Cost.ActualStatus != "unavailable" || s.run.Cost.Actual != nil {
		t.Fatal(s.run.Cost)
	}
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil || b.checks != 1 {
		t.Fatal("permanent billing error retried", err)
	}
	if err := e.RefreshCosts(ctx, s.run); err != nil {
		t.Fatal(err)
	}
	b.failure = nil
	b.available = true
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil || s.run.Cost.ActualStatus != "complete" {
		t.Fatal(s.run.Cost, err)
	}
}
func TestActualTotalIncludesAllRequestsAndUnknownIsNotZero(t *testing.T) {
	a := ZeroCost()
	a.Actual = Money("0.1")
	b := ZeroCost()
	b.Actual = Money("0.2")
	sum := SumCosts([]domain.Cost{a, b})
	if sum.Actual == nil || *sum.Actual != "0.300000000000" {
		t.Fatal(sum)
	}
	b.Actual = nil
	b.ActualStatus = "pending"
	sum = SumCosts([]domain.Cost{a, b})
	if sum.Actual != nil || sum.ActualStatus != "pending" {
		t.Fatal(sum)
	}
	blocked := domain.Cost{ActualStatus: "unavailable", Status: "partial"}
	if mixed := SumCosts([]domain.Cost{blocked, b}); mixed.ActualStatus != "pending" {
		t.Fatal("stopped polling another pending charge", mixed)
	}
	if mixed := SumCosts([]domain.Cost{blocked, a}); mixed.ActualStatus != "unavailable" || mixed.Actual != nil {
		t.Fatal(mixed)
	}
}

func TestTransientBillingFailureAndAccountingTimeout(t *testing.T) {
	e, s, p, now := setup(t)
	drive(t, e, s, now)
	b := &billingFake{fake: p, failure: &domain.Failure{Code: "PROVIDER_5XX", Retryable: true, HTTPStatus: 503}}
	e.Provider = b
	ctx := context.Background()
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil {
		t.Fatal(err)
	}
	if s.run.Status != "succeeded" || s.run.Cost.ActualStatus != "pending" || s.run.Nodes[2].Requests[0].Billing.Error.Code != "PROVIDER_5XX" {
		t.Fatal(s.run.Cost)
	}
	*now = now.Add(25 * time.Hour)
	b.failure = nil
	if err := e.ReconcileCosts(ctx, s.run.ID); err != nil {
		t.Fatal(err)
	}
	if s.run.Cost.ActualStatus != "unavailable" || s.run.Cost.Actual != nil || p.submitCount != 1 {
		t.Fatal(s.run.Cost)
	}
}
func TestInitialQuoteExactImageUnknownAndSnapshot(t *testing.T) {
	e, s, p, now := setup(t)
	models := map[string]domain.Model{}
	for _, n := range s.run.Nodes {
		if n.Model != nil {
			models[n.Model.Alias] = *n.Model
		}
	}
	quote := QuoteWorkflow(s.run.Workflow, models, s.run.Input, *now)
	s.run.InitialEstimate = &quote
	if quote.Cost.Estimated == nil || quote.Cost.Status != "complete" || *quote.Nodes[2].Cost.Estimated != "0.003000000000" || quote.Cost.Actual != nil {
		t.Fatal(quote)
	}
	before := string(domain.JSON(quote))
	drive(t, e, s, now)
	if string(domain.JSON(s.run.InitialEstimate)) != before || p.completeCount != 1 || p.submitCount != 1 {
		t.Fatal("quote changed or called providers")
	}
	m := models["flux-schnell"]
	m.Pricing = nil
	models[m.Alias] = m
	unknown := QuoteWorkflow(s.run.Workflow, models, s.run.Input, *now)
	if unknown.Cost.Status != "partial" || unknown.Nodes[2].Cost.Estimated != nil {
		t.Fatal("missing price became zero", unknown)
	}
}
