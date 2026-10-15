package engine

import "github.com/CanhK-2103/trypix-flowkitCase/internal/domain"

// Selection is durable; both model snapshots remain immutable.
func ActiveModel(n *domain.NodeRun) *domain.Model {
	if n.Model == nil {
		return nil
	}
	if n.ActiveModelRef == "" || n.ActiveModelRef == n.Model.Alias {
		return n.Model
	}
	if n.FallbackModel != nil && n.ActiveModelRef == n.FallbackModel.Alias {
		return n.FallbackModel
	}
	return nil
}

func canFallback(n *domain.NodeRun, f *domain.Failure, newGeneration bool) bool {
	return n.Node.Kind == "image" && newGeneration && f.Retryable && !f.Uncertain &&
		n.Fallback == nil && n.Model != nil && ActiveModel(n) == n.Model &&
		n.FallbackModel != nil && n.Node.FallbackModelRef == n.FallbackModel.Alias &&
		n.Model.Alias != n.FallbackModel.Alias && n.Model.Provider == n.FallbackModel.Provider &&
		n.Model.InputProfile == n.FallbackModel.InputProfile
}
