package workflow

import (
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"sync"
)

var graphSchema = sync.OnceValue(func() *jsonschema.Schema {
	raw, err := schemas.Files.ReadFile("workflow.schema.json")
	if err != nil {
		panic(err)
	}
	var value map[string]any
	if err := domain.Decode(raw, &value); err != nil {
		panic(err)
	}
	s, err := compile(value)
	if err != nil {
		panic(err)
	}
	return s
})

func Structure(d domain.Definition) []domain.Finding {
	var value any
	_ = domain.Decode(domain.JSON(d), &value)
	if err := graphSchema().Validate(value); err != nil {
		return []domain.Finding{{Code: "INVALID_STRUCTURE", Message: err.Error(), Path: "/definition"}}
	}
	return nil
}
