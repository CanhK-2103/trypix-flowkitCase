package integration

import (
	"context"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestQuoteAndDelayedBillingPersistedWithAPIScheduledRefresh(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	m := mock(t)
	m.delayBilling = true
	ctx := context.Background()
	status, data := request(t, f, "POST", "/workflows/"+productID+"/estimate", domain.Object{"input": domain.Object{"description": "blue cup"}}, "")
	if status != 200 {
		t.Fatal(status, string(data))
	}
	var quote domain.Quote
	if err := domain.Decode(data, &quote); err != nil {
		t.Fatal(err)
	}
	if quote.Cost.Estimated == nil || quote.Cost.Status != "complete" || quote.Nodes[2].Cost.Estimated == nil {
		t.Fatal(quote)
	}
	var count int
	f.store.Pool.QueryRow(ctx, "SELECT count(*) FROM runs").Scan(&count)
	if count != 0 || m.completeCount != 0 || m.submitCount != 0 {
		t.Fatal("quoting created a run or provider call")
	}
	e := testEngine(f, m)
	startWorker(t, f, e)
	id := startRun(t, f, uuid.NewString())
	run := waitRun(t, f, id)
	if run.InitialEstimate == nil || run.Cost.Actual != nil || run.Cost.ActualStatus != "pending" {
		t.Fatal(run.Cost)
	}
	original := string(domain.JSON(run.InitialEstimate))
	if err := jobs.ReconcileBilling(ctx, f.store, e); err != nil {
		t.Fatal(err)
	}
	run, err := f.store.LoadRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "succeeded" || run.Cost.Actual != nil || run.Nodes[2].Requests[0].Billing.Attempts != 1 {
		t.Fatal(run.Cost)
	}
	// Account data arrives after generation. A new engine reads the durable request;
	// the refresh endpoint only schedules a read, never resubmits generation.
	m.mu.Lock()
	m.delayBilling = false
	m.mu.Unlock()
	status, data = request(t, f, "POST", "/runs/"+id+"/costs/refresh", nil, "")
	if status != 202 {
		t.Fatal(status, string(data))
	}
	restarted := testEngine(f, m)
	if err := jobs.ReconcileBilling(ctx, f.store, restarted); err != nil {
		t.Fatal(err)
	}
	run, err = f.store.LoadRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if run.Cost.Actual == nil || *run.Cost.Actual != "0.002720000000" || run.Cost.ActualStatus != "complete" || string(domain.JSON(run.InitialEstimate)) != original {
		t.Fatal(run.Cost)
	}
	var projected string
	if err := f.store.Pool.QueryRow(ctx, "SELECT reported_cost_usd::text FROM provider_requests WHERE run_id=$1 AND provider_request_id='mock-fal-request'", id).Scan(&projected); err != nil || projected != "0.002700000000" {
		t.Fatal(projected, err)
	}
	if err := jobs.ReconcileBilling(ctx, f.store, restarted); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.submitCount != 1 || m.completeCount != 1 || m.billingChecks != 2 {
		t.Fatal("duplicate call or charge", m.submitCount, m.completeCount, m.billingChecks)
	}
	if run.FinishedAt == nil || time.Since(*run.FinishedAt) > time.Minute {
		t.Fatal("generation completion changed")
	}
}
