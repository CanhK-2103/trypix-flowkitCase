package api

import (
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"testing"
)

func TestInvalidWorkflowIDIsAMetadataFinding(t *testing.T) {
	findings := validateMetadata(domain.Workflow{ID: "invalid-uuid", Name: "Valid name"})
	if len(findings) != 1 || findings[0].Code != "INVALID_WORKFLOW_ID" || findings[0].Path != "/id" {
		t.Fatal(findings)
	}
	findings = validateMetadata(domain.Workflow{ID: "invalid-uuid"})
	if len(findings) != 2 {
		t.Fatal("ID masked independent name finding", findings)
	}
	if findings := validateMetadata(domain.Workflow{Name: "New workflow"}); len(findings) != 0 {
		t.Fatal("Optional create ID rejected", findings)
	}
}
