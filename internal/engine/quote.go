package engine

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/workflow"
)

// QuoteWorkflow makes no provider calls. Token counts are a character heuristic,
// with the configured maximum output reserved once per node; retries are excluded.
func QuoteWorkflow(w domain.Workflow, models map[string]domain.Model, input domain.Object, now time.Time) domain.Quote {
	q := domain.Quote{WorkflowRevision: w.Revision, CreatedAt: now.UTC(), Nodes: []domain.QuoteNode{}}
	chars := map[string]int{}
	known := map[string]bool{}
	costs := []domain.Cost{}
	for _, n := range workflow.Order(w.Definition) {
		part := domain.QuoteNode{NodeID: n.ID, Cost: ZeroCost(), Assumptions: []string{}}
		if n.Kind == "input" {
			field, _ := n.Params["textField"].(string)
			if v, ok := input[field]; ok && v != nil {
				chars[n.ID] = utf8.RuneCountInString(fmt.Sprint(v))
				known[n.ID] = true
			}
		}
		if n.Kind == "llm" || n.Kind == "image" {
			m, ok := models[n.ModelRef]
			part.Cost = domain.Cost{Status: "partial", ActualStatus: "pending"}
			if ok {
				part.ModelID = m.ModelID
				params := workflow.Merge(m.Defaults, n.Params)
				if n.Kind == "image" {
					width, height := imageDimensions(params["image_size"])
					part.Cost = CatalogImageCost(m, width, height)
					part.Cost.ActualStatus = "pending"
					part.Assumptions = append(part.Assumptions, "Katalog fiyatı × seçilen görsel boyutu; tek üretim, retry hariç.")
				} else {
					length, resolved := templateCharacters(n.PromptTemplate, input, chars, known)
					system, _ := params["systemPrompt"].(string)
					maxOutput := numericInt(params["max_completion_tokens"])
					if resolved && maxOutput > 0 {
						tokens := (length+utf8.RuneCountInString(system)+3)/4 + 16
						part.Cost = TokenCost(m, domain.Object{"prompt_tokens": tokens, "completion_tokens": maxOutput})
					}
					chars[n.ID] = maxOutput * 4
					known[n.ID] = maxOutput > 0
					part.Assumptions = append(part.Assumptions, "Girdi için yaklaşık 4 karakter/token + 16 mesaj tokenı; çıktı için max_completion_tokens. Kesin üst sınır değildir; tek deneme, retry hariç.")
					if !resolved {
						part.Assumptions = append(part.Assumptions, "Girdi tamamlanmadığından bu adımın tahmini henüz bilinmiyor.")
					}
				}
			}
		}
		q.Nodes = append(q.Nodes, part)
		costs = append(costs, part.Cost)
	}
	q.Cost = SumCosts(costs)
	return q
}

var quoteVariable = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

func templateCharacters(template string, input domain.Object, chars map[string]int, known map[string]bool) (int, bool) {
	length := utf8.RuneCountInString(quoteVariable.ReplaceAllString(template, ""))
	resolved := true
	for _, match := range quoteVariable.FindAllStringSubmatch(template, -1) {
		path := strings.Split(match[1], ".")
		if len(path) == 2 && path[0] == "input" {
			if v, ok := input[path[1]]; ok && v != nil {
				length += utf8.RuneCountInString(fmt.Sprint(v))
			} else {
				resolved = false
			}
		} else if len(path) == 3 && path[0] == "nodes" && known[path[1]] {
			length += chars[path[1]]
		} else {
			resolved = false
		}
	}
	return length, resolved
}
func imageDimensions(value any) (int, int) {
	if m, ok := value.(map[string]any); ok {
		return numericInt(m["width"]), numericInt(m["height"])
	}
	if m, ok := value.(domain.Object); ok {
		return numericInt(m["width"]), numericInt(m["height"])
	}
	presets := map[string][2]int{"square_hd": {1024, 1024}, "square": {512, 512}, "portrait_4_3": {768, 1024}, "portrait_16_9": {576, 1024}, "landscape_4_3": {1024, 768}, "landscape_16_9": {1024, 576}}
	if size, ok := presets[fmt.Sprint(value)]; ok {
		return size[0], size[1]
	}
	return 0, 0
}
