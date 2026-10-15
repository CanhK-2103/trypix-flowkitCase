package workflow

import (
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"strings"
	"testing"
)

func TestFallbackValidation(t *testing.T) {
	w, models := fixture(t)
	w.Definition.Nodes[2].FallbackModelRef = "flux-dev"
	if findings := Validate(w.Definition, models); len(findings) != 0 {
		t.Fatal(findings)
	}
	for _, tc := range []struct {
		name, code string
		edit       func(*domain.Definition, map[string]domain.Model)
	}{
		{"missing", "UNKNOWN_MODEL_ALIAS", func(d *domain.Definition, m map[string]domain.Model) { d.Nodes[2].FallbackModelRef = "absent" }},
		{"same", "INVALID_FALLBACK", func(d *domain.Definition, m map[string]domain.Model) {
			d.Nodes[2].FallbackModelRef = d.Nodes[2].ModelRef
		}},
		{"disabled", "MODEL_DISABLED", func(d *domain.Definition, m map[string]domain.Model) {
			x := m["flux-dev"]
			x.Enabled = false
			m["flux-dev"] = x
		}},
		{"wrong profile", "MODEL_KIND_MISMATCH", func(d *domain.Definition, m map[string]domain.Model) { d.Nodes[2].FallbackModelRef = "prompt-writer" }},
		{"unsupported kind", "INVALID_FALLBACK", func(d *domain.Definition, m map[string]domain.Model) { d.Nodes[1].FallbackModelRef = "flux-dev" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wf, m := fixture(t)
			wf.Definition.Nodes[2].FallbackModelRef = "flux-dev"
			tc.edit(&wf.Definition, m)
			for _, f := range Validate(wf.Definition, m) {
				if f.Code == tc.code && strings.HasSuffix(f.Path, "/fallbackModelRef") {
					return
				}
			}
			t.Fatal("expected fallback finding", tc.code)
		})
	}
}
