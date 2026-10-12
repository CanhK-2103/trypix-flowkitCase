package integration

import (
	"context"
	"testing"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
)

func TestWorkflowMetadataFindingsAgreeAcrossValidationAndWrites(t *testing.T) {
	f := setupAPI(t)
	load(t, f)
	wf := example(t, "workflows/product.json")
	wf["id"] = "invalid-uuid"
	nodes := wf["definition"].(map[string]any)["nodes"].([]any)
	nodes[1].(map[string]any)["modelRef"] = "unregistered"
	nodes[1].(map[string]any)["promptTemplate"] = "{{input.missing}}"
	wf["definition"].(map[string]any)["edges"] = []any{}
	for _, operation := range []struct {
		method, path string
		status       int
	}{
		{"POST", "/workflows/validate", 200}, {"POST", "/workflows", 422}, {"PUT", "/workflows/" + productID, 422},
	} {
		status, body := request(t, f, operation.method, operation.path, wf, "")
		if status != operation.status {
			t.Fatal(status, string(body))
		}
		var result struct {
			Valid    bool
			Findings []domain.Finding
			Error    struct{ Findings []domain.Finding }
		}
		if err := domain.Decode(body, &result); err != nil {
			t.Fatal(err)
		}
		findings := result.Findings
		if operation.status == 422 {
			findings = result.Error.Findings
		} else if result.Valid {
			t.Fatal("Invalid UUID validated as valid")
		}
		found := map[string]bool{}
		for _, finding := range findings {
			found[finding.Code] = true
		}
		for _, code := range []string{"INVALID_WORKFLOW_ID", "UNKNOWN_MODEL_ALIAS", "UNRESOLVED_TEMPLATE_VARIABLE", "REQUIRED_CONNECTION_MISSING"} {
			if !found[code] {
				t.Fatalf("%s %s masked %s: %s", operation.method, operation.path, code, body)
			}
		}
		if operation.method == "PUT" && !found["WORKFLOW_ID_MISMATCH"] {
			t.Fatal("Body/path ID finding missing")
		}
	}
	row, err := f.store.Q.GetWorkflow(context.Background(), storage.UUID(productID))
	if err != nil || row.Revision != 1 {
		t.Fatal("Invalid update changed existing workflow", err)
	}
	var count int
	if err := f.store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM workflows").Scan(&count); err != nil || count != 1 {
		t.Fatal("Invalid create was persisted", count, err)
	}
	validGraph := example(t, "workflows/product.json")
	validGraph["id"] = "invalid-uuid"
	status, body := request(t, f, "POST", "/workflows/validate", validGraph, "")
	var validation struct {
		Valid    bool
		Findings []domain.Finding
	}
	_ = domain.Decode(body, &validation)
	if status != 200 || validation.Valid || len(validation.Findings) != 1 || validation.Findings[0].Code != "INVALID_WORKFLOW_ID" {
		t.Fatal("Metadata-only validation is inconsistent", string(body))
	}
}
