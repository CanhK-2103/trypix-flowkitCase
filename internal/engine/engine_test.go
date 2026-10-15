package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/google/uuid"
	"testing"
	"time"
)

type memory struct {
	run       *domain.Run
	artifacts []domain.Artifact
}

func (m *memory) LoadRun(_ context.Context, _ string) (*domain.Run, error) {
	var copy domain.Run
	_ = domain.Decode(domain.JSON(m.run), &copy)
	return &copy, nil
}
func (m *memory) SaveRun(_ context.Context, r *domain.Run) error {
	var copy domain.Run
	_ = domain.Decode(domain.JSON(r), &copy)
	m.run = &copy
	return nil
}
func (m *memory) SaveArtifact(_ context.Context, a domain.Artifact) error {
	m.artifacts = append(m.artifacts, a)
	return nil
}

type fake struct {
	store                                             *memory
	completeCount, submitCount, pollCount, priceCount int
	completeFailures                                  []*domain.Failure
	pollFailure                                       *domain.Failure
	permanentImage                                    bool
	submitFailures                                    []*domain.Failure
	submittedModels                                   []string
	submittedParams                                   []domain.Object
	estimateFailure                                   *domain.Failure
	terminalFailure                                   *domain.Failure
}

func (f *fake) Complete(_ context.Context, m domain.Model, _ string, _ domain.Object) (providers.Completion, *domain.Failure) {
	f.completeCount++
	if len(f.completeFailures) > 0 {
		e := f.completeFailures[0]
		f.completeFailures = f.completeFailures[1:]
		return providers.Completion{}, e
	}
	return providers.Completion{ID: "completion-1", Model: m.ModelID, Prompt: "A clean product photo", Raw: json.RawMessage(`{"id":"completion-1"}`), Usage: domain.Object{"prompt_tokens": json.Number("100"), "completion_tokens": json.Number("30"), "cost": json.Number("0.000025")}}, nil
}
func (f *fake) Submit(_ context.Context, m domain.Model, _ string, params domain.Object) (providers.Ticket, *domain.Failure) {
	f.submitCount++
	f.submittedModels = append(f.submittedModels, m.ModelID)
	f.submittedParams = append(f.submittedParams, params)
	if len(f.submitFailures) > 0 {
		failure := f.submitFailures[0]
		f.submitFailures = f.submitFailures[1:]
		return providers.Ticket{}, failure
	}
	if f.permanentImage {
		return providers.Ticket{}, &domain.Failure{Code: "INVALID_INPUT", HTTPStatus: 422, ProviderCode: "validation_error", Message: "Invalid image input"}
	}
	return providers.Ticket{ID: fmt.Sprintf("fal-%d", f.submitCount), StatusURL: "http://mock/status", ResponseURL: "http://mock/result"}, nil
}
func (f *fake) Status(_ context.Context, _ providers.Ticket) (providers.QueueState, *domain.Failure) {
	f.pollCount++
	if f.terminalFailure != nil {
		failure := f.terminalFailure
		f.terminalFailure = nil
		return providers.QueueState{Status: "COMPLETED", Error: failure}, nil
	}
	if f.pollFailure != nil {
		e := f.pollFailure
		f.pollFailure = nil
		return providers.QueueState{}, e
	}
	if f.pollCount < 2 {
		return providers.QueueState{Status: "IN_PROGRESS"}, nil
	}
	return providers.QueueState{Status: "COMPLETED"}, nil
}
func (f *fake) Result(_ context.Context, _ providers.Ticket) (providers.ImageResult, *domain.Failure) {
	return providers.ImageResult{URL: "http://mock/image", Width: 512, Height: 512, Raw: json.RawMessage(`{"images":[{"url":"http://mock/image","width":512,"height":512}]}`)}, nil
}
func (f *fake) Estimate(_ context.Context, m domain.Model) (string, *domain.Failure) {
	f.priceCount++
	n := f.store.run.Nodes[2]
	if len(n.Requests) == 0 || n.Requests[len(n.Requests)-1].ProviderRequestID == "" {
		panic("price called before durable ticket")
	}
	if f.estimateFailure != nil {
		return "", f.estimateFailure
	}
	if m.Pricing != nil {
		return m.Pricing.UnitPrice, nil
	}
	return "0.003", nil
}

type fakeMedia struct{}

func (fakeMedia) Download(_ context.Context, _ string, a domain.Artifact) (domain.Artifact, error) {
	a.FileName = a.ID + ".image"
	a.MIME = "image/png"
	a.Size = 10
	return a, nil
}
func setup(t *testing.T) (*Engine, *memory, *fake, *time.Time) {
	t.Helper()
	var w domain.Workflow
	b, _ := examples.Files.ReadFile("workflows/product.json")
	_ = domain.Decode(b, &w)
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	r := &domain.Run{ID: uuid.NewString(), Workflow: w, Input: domain.Object{"description": "blue cup"}, Status: "queued", CreatedAt: now, Nodes: []domain.NodeRun{}}
	for _, n := range w.Definition.Nodes {
		nr := domain.NodeRun{Node: n, Status: "pending", Cost: ZeroCost(), Attempts: []domain.Attempt{}, Requests: []domain.ProviderRequest{}}
		if n.ModelRef != "" {
			b, _ := examples.Files.ReadFile("models/" + n.ModelRef + ".json")
			var m domain.Model
			_ = domain.Decode(b, &m)
			nr.Model = &m
			nr.Cost = domain.Cost{Status: "partial"}
		}
		r.Nodes = append(r.Nodes, nr)
	}
	store := &memory{run: r}
	provider := &fake{store: store}
	cfg := config.Load()
	cfg.PollInterval = time.Second
	cfg.RetryBase = 2 * time.Second
	e := &Engine{Store: store, Provider: provider, Media: fakeMedia{}, Config: cfg, Now: func() time.Time { return now }, Jitter: func() time.Duration { return 0 }}
	return e, store, provider, &now
}
func drive(t *testing.T, e *Engine, s *memory, now *time.Time) {
	t.Helper()
	for i := 0; i < 100; i++ {
		delay, err := e.Step(context.Background(), s.run.ID)
		if err != nil {
			t.Fatal(err)
		}
		*now = now.Add(delay + time.Millisecond)
		if s.run.Terminal() {
			return
		}
	}
	t.Fatal("did not reach terminal state")
}
func TestSuccessfulRun(t *testing.T) {
	e, s, p, now := setup(t)
	drive(t, e, s, now)
	if s.run.Status != "succeeded" || p.completeCount != 1 || p.submitCount != 1 || len(s.artifacts) != 1 {
		t.Fatal(s.run.Status, p)
	}
	if *s.run.Cost.Estimated != "0.003025000000" {
		t.Fatal(s.run.Cost)
	}
	if s.artifacts[0].ActualModelID != "fal-ai/flux/schnell" {
		t.Fatal(s.artifacts)
	}
	if string(domain.JSON(s.run.Nodes[0].Input)) != string(domain.JSON(s.run.Input)) {
		t.Fatal("Input node trace did not preserve the submitted input")
	}
}
func TestTransientRetryThenSuccess(t *testing.T) {
	e, s, p, now := setup(t)
	p.completeFailures = []*domain.Failure{{Code: "PROVIDER_5XX", Message: "busy", Retryable: true, HTTPStatus: 503}}
	drive(t, e, s, now)
	if s.run.Status != "succeeded" || p.completeCount != 2 || len(s.run.Nodes[1].Attempts) != 2 {
		t.Fatal(s.run.Status, p.completeCount, s.run.Nodes[1])
	}
	if s.run.Nodes[1].Attempts[0].Error.Code != "PROVIDER_5XX" {
		t.Fatal("lost error")
	}
	if s.run.Cost.Status != "partial" {
		t.Fatal("unknown first request must remain partial")
	}
}
func TestPermanentFailureCorrectNode(t *testing.T) {
	e, s, p, now := setup(t)
	p.permanentImage = true
	drive(t, e, s, now)
	if s.run.Status != "failed" || s.run.FailedNodeID != "render" || s.run.Error.Code != "INVALID_INPUT" || s.run.Error.ProviderCode != "validation_error" || p.submitCount != 1 || s.run.Nodes[3].Status != "skipped" {
		t.Fatal(fmt.Sprintf("%+v", s.run))
	}
}
func TestRetryCap(t *testing.T) {
	e, s, p, now := setup(t)
	for i := 0; i < 5; i++ {
		p.completeFailures = append(p.completeFailures, &domain.Failure{Code: "PROVIDER_TIMEOUT", Message: "timeout", Retryable: true})
	}
	drive(t, e, s, now)
	if p.completeCount != 3 || len(s.run.Nodes[1].Attempts) != 3 || s.run.Status != "failed" {
		t.Fatal(p.completeCount, s.run.Status)
	}
}
func TestResumeKnownFalRequest(t *testing.T) {
	e, s, p, now := setup(t)
	for len(s.run.Nodes[2].Requests) == 0 {
		delay, err := e.Step(context.Background(), s.run.ID)
		if err != nil {
			t.Fatal(err)
		}
		*now = now.Add(delay + time.Millisecond)
	}
	savedID := s.run.Nodes[2].Requests[0].ProviderRequestID
	e2 := &Engine{Store: s, Provider: p, Media: fakeMedia{}, Config: e.Config, Now: e.Now, Jitter: e.Jitter}
	drive(t, e2, s, now)
	if s.run.Status != "succeeded" || p.submitCount != 1 || len(s.run.Nodes[2].Attempts) != 1 || s.run.Nodes[2].Requests[0].ProviderRequestID != savedID || p.priceCount != 1 {
		t.Fatal("resume duplicated submission/attempt/cost")
	}
}
func TestPollRetryKeepsRequestAndCost(t *testing.T) {
	e, s, p, now := setup(t)
	p.pollFailure = &domain.Failure{Code: "PROVIDER_TIMEOUT", Message: "poll timed out", Retryable: true}
	drive(t, e, s, now)
	n := s.run.Nodes[2]
	if s.run.Status != "succeeded" || p.submitCount != 1 || len(n.Attempts) != 2 || len(n.Requests) != 1 || *n.Cost.Estimated != "0.003000000000" {
		t.Fatal(n)
	}
}
func TestUncertainSubmissionIsNotRetried(t *testing.T) {
	e, s, p, now := setup(t)
	p.completeFailures = []*domain.Failure{{Code: "PROVIDER_SUBMISSION_UNKNOWN", Message: "response lost", Uncertain: true}}
	drive(t, e, s, now)
	if p.completeCount != 1 || s.run.Error.Code != "PROVIDER_SUBMISSION_UNKNOWN" {
		t.Fatal(s.run.Error)
	}
}
func TestDecimalUnknownCosts(t *testing.T) {
	a := domain.Cost{Estimated: Money("0.1"), Reported: Money("0.2"), Status: "complete"}
	b := domain.Cost{Estimated: Money("0.2"), Status: "complete"}
	sum := SumCosts([]domain.Cost{a, b})
	if *sum.Estimated != "0.300000000000" {
		t.Fatal(sum)
	}
	sum = SumCosts([]domain.Cost{a, {Status: "partial"}})
	if sum.Status != "partial" || *sum.Estimated != "0.100000000000" {
		t.Fatal(sum)
	}
	if Money("-1") != nil || Money("bad") != nil {
		t.Fatal("invalid money accepted")
	}
}
