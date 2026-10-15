package main

import (
	"testing"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
)

func TestApprovalAfterFallbackNamesTheSecondaryModel(t *testing.T) {
	run := &domain.Run{Nodes: []domain.NodeRun{{
		Node:           domain.Node{ID: "poster", Kind: "image", ModelRef: "flux-krea", FallbackModelRef: "flux-dev"},
		Model:          &domain.Model{Alias: "flux-krea", Provider: "fal", ModelID: "fal-ai/flux/krea"},
		FallbackModel:  &domain.Model{Alias: "flux-dev", Provider: "fal", ModelID: "fal-ai/flux/dev"},
		ActiveModelRef: "flux-dev", Status: "retry_wait",
		Requests: []domain.ProviderRequest{{Phase: "generation_failed"}},
	}}}
	op := nextOperation(run, config.Config{FalURL: "http://localhost:9999"})
	if op.Model != "fal-ai/flux/dev" || op.Action != "submit" || op.URL != "http://localhost:9999/fal-ai/flux/dev" {
		t.Fatal("approval must name exactly the next selected model", op)
	}
}
