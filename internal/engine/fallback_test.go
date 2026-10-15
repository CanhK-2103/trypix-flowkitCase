package engine

import (
	"context"
	"encoding/json"
	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"testing"
	"time"
)

func configureFallback(t *testing.T, s *memory) {
	t.Helper()
	data, _ := examples.Files.ReadFile("models/flux-dev.json")
	var model domain.Model
	if err := domain.Decode(data, &model); err != nil {
		t.Fatal(err)
	}
	n := &s.run.Nodes[2]
	n.Node.FallbackModelRef = model.Alias
	n.FallbackModel = &model
	s.run.Workflow.Definition.Nodes[2].FallbackModelRef = model.Alias
	s.run.InitialEstimate = &domain.Quote{Cost: domain.Cost{Estimated: Money("0.003")}}
}
func transient() *domain.Failure {
	return &domain.Failure{Code: "PROVIDER_5XX", HTTPStatus: 503, Retryable: true, Message: "Mock unavailable"}
}

func TestPrimarySuccessDoesNotCallFallback(t *testing.T) {
	e, s, p, now := setup(t)
	configureFallback(t, s)
	drive(t, e, s, now)
	if s.run.Status != "succeeded" || p.submitCount != 1 || s.run.Nodes[2].Fallback != nil || s.artifacts[0].ActualModelID != "fal-ai/flux/schnell" {
		t.Fatal("unexpected fallback")
	}
}
func TestFallbackSelectionSurvivesRestartAndPreservesLedger(t *testing.T) {
	e, s, p, now := setup(t)
	configureFallback(t, s)
	p.submitFailures = []*domain.Failure{transient()}
	for s.run.Nodes[2].Fallback == nil {
		delay, err := e.Step(context.Background(), s.run.ID)
		if err != nil {
			t.Fatal(err)
		}
		*now = now.Add(delay + time.Millisecond)
		if s.run.Terminal() {
			t.Fatal("failed before fallback")
		}
	}
	originalDeadline := *s.run.Nodes[2].DeadlineAt
	if p.submitCount != 1 || s.run.Nodes[2].ActiveModelRef != "flux-dev" {
		t.Fatal("selection was not durable before another submit")
	}
	restarted := &Engine{Store: s, Provider: p, Media: fakeMedia{}, Config: e.Config, Now: e.Now, Jitter: e.Jitter}
	drive(t, restarted, s, now)
	n := s.run.Nodes[2]
	if s.run.Status != "succeeded" || p.submitCount != 2 || len(n.Attempts) != 2 || len(n.Requests) != 2 {
		t.Fatal(s.run.Error, n)
	}
	if p.submittedModels[0] != "fal-ai/flux/schnell" || p.submittedModels[1] != "fal-ai/flux/dev" || n.Model.ModelID != "fal-ai/flux/schnell" || n.FallbackModel.ModelID != "fal-ai/flux/dev" {
		t.Fatal("model snapshots/provenance changed")
	}
	if n.Requests[0].ModelRef != "flux-schnell" || n.Requests[1].ModelRef != "flux-dev" || n.Requests[0].Phase != "generation_failed" || n.Fallback.Reason.Code != "PROVIDER_5XX" {
		t.Fatal("lost request/fallback history")
	}
	if s.artifacts[0].ActualModelID != "fal-ai/flux/dev" || !n.DeadlineAt.Equal(originalDeadline) || *s.run.InitialEstimate.Cost.Estimated != "0.003000000000" {
		t.Fatal("fallback changed deadline, quote or producer")
	}
	if *n.Cost.Estimated != "0.025000000000" || n.Cost.Status != "partial" || n.Cost.Actual != nil {
		t.Fatal("unknown primary charge was lost", n.Cost)
	}
	if p.submittedParams[0]["num_inference_steps"].(json.Number).String() != "4" {
		t.Fatal("primary defaults")
	}
	if p.submittedParams[1]["num_inference_steps"].(json.Number).String() != "28" {
		t.Fatal("secondary defaults")
	}
}
func TestFallbackSharesThreeAttemptLimit(t *testing.T) {
	e, s, p, now := setup(t)
	configureFallback(t, s)
	p.submitFailures = []*domain.Failure{transient(), transient(), transient(), transient()}
	drive(t, e, s, now)
	if s.run.Status != "failed" || p.submitCount != 3 || len(s.run.Nodes[2].Attempts) != 3 || p.submittedModels[0] != "fal-ai/flux/schnell" || p.submittedModels[1] != "fal-ai/flux/dev" || p.submittedModels[2] != "fal-ai/flux/dev" {
		t.Fatal("attempt budget or order", p.submittedModels)
	}
}
func TestFallbackNeverResubmitsUncertainKnownOrPermanentFailures(t *testing.T) {
	for _, test := range []string{"uncertain", "poll", "permanent", "pricing"} {
		t.Run(test, func(t *testing.T) {
			e, s, p, now := setup(t)
			configureFallback(t, s)
			switch test {
			case "uncertain":
				p.submitFailures = []*domain.Failure{{Code: "PROVIDER_SUBMISSION_UNKNOWN", Retryable: true, Uncertain: true}}
			case "poll":
				p.pollFailure = transient()
			case "permanent":
				p.permanentImage = true
			case "pricing":
				p.estimateFailure = transient()
			}
			drive(t, e, s, now)
			if p.submitCount != 1 || s.run.Nodes[2].Fallback != nil || len(s.run.Nodes[2].Requests) != 1 {
				t.Fatal("unsafe fallback", test)
			}
			want := "succeeded"
			if test == "uncertain" || test == "permanent" {
				want = "failed"
			}
			if s.run.Status != want {
				t.Fatal(s.run.Error)
			}
		})
	}
}
func TestConfirmedTerminalProviderErrorCanUseFallback(t *testing.T) {
	e, s, p, now := setup(t)
	configureFallback(t, s)
	p.terminalFailure = transient()
	drive(t, e, s, now)
	if s.run.Status != "succeeded" || p.submitCount != 2 || s.artifacts[0].ActualModelID != "fal-ai/flux/dev" {
		t.Fatal(s.run.Error)
	}
}
