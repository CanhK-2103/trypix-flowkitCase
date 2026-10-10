package main

import (
	"encoding/json"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"testing"
)

func TestDirectBudgetIncludesDirectRequestsAndRequiresEvidence(t *testing.T) {
	record := domain.Object{"phase": "complete", "fal_estimated_cost_usd": "0.003", "models": []domain.Model{{Provider: "openrouter"}}, "completion": providers.Completion{Usage: domain.Object{"cost": json.Number("0.000012345678")}}}
	fal, or, e := directBudget(domain.JSON(record))
	if e != nil || fal.String() != "0.003" || or.String() != "0.000012345678" {
		t.Fatal(fal, or, e)
	}
	record["phase"] = "fal_waiting"
	if _, _, e := directBudget(domain.JSON(record)); e == nil {
		t.Fatal("incomplete evidence allowed")
	}
	record["phase"] = "complete"
	record["completion"] = providers.Completion{}
	if _, _, e := directBudget(domain.JSON(record)); e == nil {
		t.Fatal("unknown cost became zero")
	}
}
