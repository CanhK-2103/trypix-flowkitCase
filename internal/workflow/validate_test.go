package workflow

import (
	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"testing"
)

func fixture(t *testing.T) (domain.Workflow, map[string]domain.Model) {
	t.Helper()
	models := map[string]domain.Model{}
	for _, alias := range []string{"prompt-writer", "flux-schnell", "flux-dev"} {
		b, _ := examples.Files.ReadFile("models/" + alias + ".json")
		var m domain.Model
		if e := domain.Decode(b, &m); e != nil {
			t.Fatal(e)
		}
		models[alias] = m
	}
	b, _ := examples.Files.ReadFile("workflows/product.json")
	var w domain.Workflow
	if e := domain.Decode(b, &w); e != nil {
		t.Fatal(e)
	}
	return w, models
}
func TestValidExamples(t *testing.T) {
	_, models := fixture(t)
	for _, name := range []string{"product", "event"} {
		b, _ := examples.Files.ReadFile("workflows/" + name + ".json")
		var w domain.Workflow
		_ = domain.Decode(b, &w)
		if f := Structure(w.Definition); len(f) > 0 {
			t.Fatal(f)
		}
		if f := Validate(w.Definition, models); len(f) > 0 {
			t.Fatal(f)
		}
	}
}
func TestRejectGraphRules(t *testing.T) {
	cases := []struct {
		name, code string
		modify     func(*domain.Definition)
	}{
		{"cycle", "CYCLE_DETECTED", func(d *domain.Definition) {
			d.Edges = append(d.Edges, domain.Edge{From: "render", FromPort: "image", To: "prompt_writer", ToPort: "text"})
		}},
		{"fanout", "MULTIPLE_OUTGOING_EDGES", func(d *domain.Definition) {
			d.Edges = append(d.Edges, domain.Edge{From: "input", FromPort: "text", To: "render", ToPort: "prompt"})
		}},
		{"missing port", "REQUIRED_CONNECTION_MISSING", func(d *domain.Definition) { d.Edges = d.Edges[:2] }},
		{"port mismatch", "PORT_TYPE_MISMATCH", func(d *domain.Definition) { d.Edges[0].ToPort = "prompt" }},
		{"unknown model", "UNKNOWN_MODEL_ALIAS", func(d *domain.Definition) { d.Nodes[2].ModelRef = "not-registered" }},
		{"template future", "UNRESOLVED_TEMPLATE_VARIABLE", func(d *domain.Definition) { d.Nodes[1].PromptTemplate = "{{nodes.render.image}}" }},
		{"template unknown", "UNRESOLVED_TEMPLATE_VARIABLE", func(d *domain.Definition) { d.Nodes[1].PromptTemplate = "{{input.missing}}" }},
		{"fanin", "MULTIPLE_INCOMING_EDGES", func(d *domain.Definition) { d.Edges = append(d.Edges, d.Edges[0]) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, m := fixture(t)
			tc.modify(&w.Definition)
			f := Validate(w.Definition, m)
			for _, v := range f {
				if v.Code == tc.code {
					return
				}
			}
			t.Fatalf("missing %s: %+v", tc.code, f)
		})
	}
}
func TestIndependentFindings(t *testing.T) {
	w, m := fixture(t)
	w.Definition.Nodes[1].ModelRef = "unknown"
	w.Definition.Nodes[1].PromptTemplate = "{{input.missing}}"
	w.Definition.Edges = w.Definition.Edges[:2]
	if len(Validate(w.Definition, m)) < 3 {
		t.Fatal("must report independent findings")
	}
}
func TestInputAndTemplate(t *testing.T) {
	w, _ := fixture(t)
	for _, input := range []domain.Object{{"description": ""}, {"description": 5}, {"description": "ok", "extra": "no"}} {
		if ValidateInput(w.Definition.Nodes[0], input) == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if e := ValidateInput(w.Definition.Nodes[0], domain.Object{"description": "cup"}); e != nil {
		t.Fatal(e)
	}
	s, e := Render("{{input.description}} / {{nodes.writer.prompt}}", domain.Object{"description": "cup"}, map[string]domain.Object{"writer": {"prompt": "photo"}})
	if e != nil || s != "cup / photo" {
		t.Fatal(s, e)
	}
	if _, e := Render("{{input.missing}}", nil, nil); e == nil {
		t.Fatal("missing value accepted")
	}
}
func TestCompatibleCatalogModels(t *testing.T) {
	_, models := fixture(t)
	m := models["flux-schnell"]
	m.Alias = "another-model"
	m.ModelID = "new-provider/model-v2"
	if f := ValidateModel(m); len(f) > 0 {
		t.Fatal(f)
	}
	m = models["prompt-writer"]
	m.ModelID += ":batch"
	if f := ValidateModel(m); len(f) == 0 {
		t.Fatal("batch accepted")
	}
}
