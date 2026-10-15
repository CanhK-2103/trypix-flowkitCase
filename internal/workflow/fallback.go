package workflow

import (
	"fmt"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
)

func validateFallback(n domain.Node, index int, models map[string]domain.Model) []domain.Finding {
	if n.FallbackModelRef == "" {
		return nil
	}
	path := fmt.Sprintf("/definition/nodes/%d/fallbackModelRef", index)
	out := []domain.Finding{}
	add := func(code, message string) {
		out = append(out, domain.Finding{Code: code, Message: message, Path: path, NodeID: n.ID})
	}
	if n.Kind != "image" {
		add("INVALID_FALLBACK", "Only image nodes support an explicit secondary model")
	}
	if n.FallbackModelRef == n.ModelRef {
		add("INVALID_FALLBACK", "Primary and secondary model aliases must differ")
	}
	secondary, exists := models[n.FallbackModelRef]
	if !exists {
		add("UNKNOWN_MODEL_ALIAS", "Secondary model alias is not registered: "+n.FallbackModelRef)
		return out
	}
	if !secondary.Enabled {
		add("MODEL_DISABLED", "Secondary model is disabled")
	}
	if secondary.InputProfile != "fal-text-to-image-v1" {
		add("MODEL_KIND_MISMATCH", "Secondary model must support the image input profile")
	}
	if primary, ok := models[n.ModelRef]; ok && (secondary.Provider != primary.Provider || secondary.InputProfile != primary.InputProfile) {
		add("INVALID_FALLBACK", "Primary and secondary must use the same provider and input profile")
	}
	for _, message := range ValidateParams(secondary, Merge(secondary.Defaults, n.Params)) {
		add("INVALID_PARAMS", "Secondary model: "+message)
	}
	return out
}
