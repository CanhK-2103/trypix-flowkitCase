package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shopspring/decimal"
)

var identifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
var aliasPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var modelPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_./:-]{0,255}$`)
var variable = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

func Merge(defaults, params domain.Object) domain.Object {
	result := domain.Object{}
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range params {
		result[k] = v
	}
	return result
}
func ValidateModel(m domain.Model) []domain.Finding {
	findings := []domain.Finding{}
	add := func(code, msg, path string) {
		findings = append(findings, domain.Finding{Code: code, Message: msg, Path: path})
	}
	if !aliasPattern.MatchString(m.Alias) {
		add("INVALID_MODEL", "Alias must contain 1–64 letters, numbers, underscores or hyphens", "/alias")
	}
	if !modelPattern.MatchString(m.ModelID) || strings.Contains(m.ModelID, "..") || strings.Contains(m.ModelID, "//") {
		add("INVALID_MODEL", "Invalid provider model identifier", "/modelId")
	}
	if m.Provider == "openrouter" && m.InputProfile == "openrouter-chat-text-v1" {
		if strings.HasSuffix(m.ModelID, ":batch") {
			add("UNSUPPORTED_MODEL_VARIANT", "Batch models require a separate execution protocol", "/modelId")
		}
	} else if !(m.Provider == "fal" && m.InputProfile == "fal-text-to-image-v1") {
		add("INVALID_MODEL", "Provider and supported inputProfile must match", "/inputProfile")
	}
	for _, msg := range ValidateParams(m, m.Defaults) {
		add("INVALID_PARAMS", msg, "/defaults")
	}
	if m.Pricing != nil {
		p := m.Pricing
		if p.Currency != "USD" {
			add("INVALID_PRICING", "Currency must be USD", "/pricing/currency")
		}
		if p.Unit != "megapixel" && p.Unit != "image" && p.Unit != "tokens" {
			add("INVALID_PRICING", "Supported units: megapixel, image, tokens", "/pricing/unit")
		}
		for _, v := range []string{p.UnitPrice, p.InputPerMillion, p.OutputPerMillion} {
			if v != "" {
				d, e := decimal.NewFromString(v)
				if e != nil || d.IsNegative() || d.GreaterThan(decimal.RequireFromString("999999999999.999999999999")) || d.Exponent() < -12 {
					add("INVALID_PRICING", "Prices must be nonnegative decimal strings within NUMERIC(24,12)", "/pricing")
				}
			}
		}
	}
	return findings
}
func ValidateParams(m domain.Model, p domain.Object) []string {
	out := []string{}
	allowed := map[string]bool{}
	if m.InputProfile == "openrouter-chat-text-v1" {
		for _, k := range []string{"systemPrompt", "temperature", "reasoning", "max_completion_tokens"} {
			allowed[k] = true
		}
	} else if m.InputProfile == "fal-text-to-image-v1" {
		for _, k := range []string{"image_size", "num_images", "num_inference_steps", "seed"} {
			allowed[k] = true
		}
	}
	keys := []string{}
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := p[k]
		if !allowed[k] {
			out = append(out, "Unsupported parameter: "+k)
			continue
		}
		switch k {
		case "systemPrompt":
			if _, ok := v.(string); !ok {
				out = append(out, "systemPrompt must be a string")
			}
		case "reasoning":
			r, ok := v.(map[string]any)
			if !ok {
				if d, yes := v.(domain.Object); yes {
					r = d
					ok = true
				}
			}
			if !ok || len(r) != 1 {
				out = append(out, "reasoning must contain only effort")
			} else {
				effort, ok := r["effort"].(string)
				if !ok || !contains([]string{"none", "minimal", "low", "medium", "high", "xhigh"}, effort) {
					out = append(out, "Invalid reasoning effort")
				}
			}
		case "temperature":
			if n, ok := number(v); !ok || n < 0 || n > 2 {
				out = append(out, "temperature must be between 0 and 2")
			}
		case "max_completion_tokens":
			if n, ok := integer(v); !ok || n < 1 || n > 4096 {
				out = append(out, "max_completion_tokens must be an integer between 1 and 4096")
			}
		case "num_images":
			if n, ok := integer(v); !ok || n != 1 {
				out = append(out, "Only num_images: 1 is supported")
			}
		case "num_inference_steps":
			if n, ok := integer(v); !ok || n < 1 || n > 100 {
				out = append(out, "num_inference_steps must be an integer between 1 and 100")
			}
		case "seed":
			if _, ok := integer(v); !ok {
				out = append(out, "seed must be an integer")
			}
		case "image_size":
			if s, ok := v.(string); ok {
				if !contains([]string{"square_hd", "square", "portrait_4_3", "portrait_16_9", "landscape_4_3", "landscape_16_9"}, s) {
					out = append(out, "Unknown image_size preset")
				}
			} else {
				var size map[string]any
				switch x := v.(type) {
				case map[string]any:
					size = x
				case domain.Object:
					size = x
				}
				w, okW := integer(size["width"])
				h, okH := integer(size["height"])
				if len(size) != 2 || !okW || !okH || w < 1 || h < 1 || w > 14142 || h > 14142 {
					out = append(out, "image_size must contain positive integer width/height up to 14142")
				}
			}
		}
	}
	return out
}
func contains(list []string, value string) bool {
	for _, x := range list {
		if x == value {
			return true
		}
	}
	return false
}
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, e := n.Float64()
		return f, e == nil
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}
func integer(v any) (int64, bool) {
	switch n := v.(type) {
	case json.Number:
		i, e := n.Int64()
		return i, e == nil
	case int:
		return int64(n), true
	case float64:
		if n == float64(int64(n)) {
			return int64(n), true
		}
	}
	return 0, false
}
func object(v any) map[string]any {
	switch x := v.(type) {
	case map[string]any:
		return x
	case domain.Object:
		return x
	}
	return nil
}

func Validate(d domain.Definition, models map[string]domain.Model) []domain.Finding {
	out := Structure(d)
	add := func(code, msg, path, node string) {
		out = append(out, domain.Finding{Code: code, Message: msg, Path: path, NodeID: node})
	}
	if d.SchemaVersion != 1 {
		add("INVALID_STRUCTURE", "schemaVersion must be 1", "/definition/schemaVersion", "")
	}
	if len(d.Nodes) < 2 || len(d.Nodes) > 50 {
		add("INVALID_STRUCTURE", "Workflow must have between 2 and 50 nodes", "/definition/nodes", "")
	}
	ids := map[string]int{}
	incoming := map[string][]domain.Edge{}
	outgoing := map[string][]domain.Edge{}
	inputs, outputs := 0, 0
	fields := map[string]bool{}
	for i, n := range d.Nodes {
		path := fmt.Sprintf("/definition/nodes/%d", i)
		if !identifier.MatchString(n.ID) {
			add("INVALID_STRUCTURE", "Node ID must be a valid identifier", path+"/id", n.ID)
		}
		if _, ok := ids[n.ID]; ok {
			add("DUPLICATE_NODE_ID", "Duplicate node ID: "+n.ID, path+"/id", n.ID)
		} else {
			ids[n.ID] = i
		}
		switch n.Kind {
		case "input":
			inputs++
			s := object(n.Params["inputSchema"])
			if errs := validateInputSchema(s); len(errs) > 0 {
				for _, msg := range errs {
					add("INVALID_PARAMS", msg, path+"/params/inputSchema", n.ID)
				}
			}
			properties := object(s["properties"])
			for name := range properties {
				fields[name] = true
			}
			tf, ok := n.Params["textField"].(string)
			if !ok || object(properties[tf])["type"] != "string" {
				add("INVALID_PARAMS", "textField must reference a declared string input", path+"/params/textField", n.ID)
			}
			for k := range n.Params {
				if k != "inputSchema" && k != "textField" {
					add("INVALID_PARAMS", "Unsupported input parameter: "+k, path+"/params", n.ID)
				}
			}
		case "output":
			outputs++
			if len(n.Params) > 0 {
				add("INVALID_PARAMS", "Output params must be empty", path+"/params", n.ID)
			}
		case "llm", "image":
			m, ok := models[n.ModelRef]
			if !ok {
				add("UNKNOWN_MODEL_ALIAS", "Model alias is not registered: "+n.ModelRef, path+"/modelRef", n.ID)
			} else {
				if !m.Enabled {
					add("MODEL_DISABLED", "Model is disabled", path+"/modelRef", n.ID)
				}
				if (n.Kind == "llm" && m.InputProfile != "openrouter-chat-text-v1") || (n.Kind == "image" && m.InputProfile != "fal-text-to-image-v1") {
					add("MODEL_KIND_MISMATCH", "Model profile does not support node kind", path+"/modelRef", n.ID)
				}
				for _, msg := range ValidateParams(m, Merge(m.Defaults, n.Params)) {
					add("INVALID_PARAMS", msg, path+"/params", n.ID)
				}
			}
			if n.Kind == "llm" && strings.TrimSpace(n.PromptTemplate) == "" {
				add("INVALID_STRUCTURE", "LLM promptTemplate is required", path+"/promptTemplate", n.ID)
			}
		default:
			add("INVALID_STRUCTURE", "Unknown node kind", path+"/kind", n.ID)
		}
		if (n.Kind == "input" || n.Kind == "output") && (n.ModelRef != "" || n.PromptTemplate != "") {
			add("INVALID_STRUCTURE", "Input/output cannot reference models or prompts", path, n.ID)
		}
	}
	if inputs != 1 {
		add("INVALID_STRUCTURE", "Exactly one input node is required", "/definition/nodes", "")
	}
	if outputs != 1 {
		add("INVALID_STRUCTURE", "Exactly one output node is required", "/definition/nodes", "")
	}
	for i, e := range d.Edges {
		path := fmt.Sprintf("/definition/edges/%d", i)
		from, okF := ids[e.From]
		to, okT := ids[e.To]
		if !okF || !okT {
			add("UNKNOWN_NODE", "Edge references an unknown node", path, "")
			continue
		}
		incoming[e.To] = append(incoming[e.To], e)
		outgoing[e.From] = append(outgoing[e.From], e)
		fp := outputPort(d.Nodes[from].Kind)
		tp := inputPorts(d.Nodes[to].Kind)
		if fp == "" || e.FromPort != fp || !contains(tp, e.ToPort) {
			add("UNKNOWN_PORT", "Edge references an unsupported port", path, e.To)
		} else if e.FromPort != e.ToPort {
			add("PORT_TYPE_MISMATCH", "Port types do not match: "+e.FromPort+" → "+e.ToPort, path, e.To)
		}
	}
	for i, n := range d.Nodes {
		path := fmt.Sprintf("/definition/nodes/%d", i)
		if len(incoming[n.ID]) > 1 {
			add("MULTIPLE_INCOMING_EDGES", "Only one incoming edge is allowed", path, n.ID)
		}
		if len(outgoing[n.ID]) > 1 {
			add("MULTIPLE_OUTGOING_EDGES", "Only one outgoing edge is allowed", path, n.ID)
		}
		if (n.Kind != "input" && len(incoming[n.ID]) == 0) || (n.Kind != "output" && len(outgoing[n.ID]) == 0) {
			add("REQUIRED_CONNECTION_MISSING", "A required connection is missing", path, n.ID)
		}
		if n.Kind == "input" && len(incoming[n.ID]) > 0 || n.Kind == "output" && len(outgoing[n.ID]) > 0 {
			add("INVALID_STRUCTURE", "Input cannot receive edges; output cannot send edges", path, n.ID)
		}
	}
	colors := map[string]int{}
	var dfs func(string)
	cycle := false
	dfs = func(id string) {
		if colors[id] == 1 {
			cycle = true
			return
		}
		if colors[id] == 2 {
			return
		}
		colors[id] = 1
		for _, e := range outgoing[id] {
			dfs(e.To)
		}
		colors[id] = 2
	}
	for id := range ids {
		dfs(id)
	}
	if cycle {
		add("CYCLE_DETECTED", "Graph contains a cycle", "/definition/edges", "")
	}
	reached := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		if reached[id] {
			return
		}
		reached[id] = true
		for _, e := range outgoing[id] {
			visit(e.To)
		}
	}
	for _, n := range d.Nodes {
		if n.Kind == "input" {
			visit(n.ID)
		}
	}
	for i, n := range d.Nodes {
		if !reached[n.ID] {
			add("DISCONNECTED_NODE", "Node is unreachable from input", fmt.Sprintf("/definition/nodes/%d", i), n.ID)
		}
	}
	for i, n := range d.Nodes {
		ancestors := map[string]bool{}
		var previous func(string)
		previous = func(id string) {
			for _, e := range incoming[id] {
				if !ancestors[e.From] {
					ancestors[e.From] = true
					previous(e.From)
				}
			}
		}
		previous(n.ID)
		delete(ancestors, n.ID)
		for _, ref := range References(n.PromptTemplate) {
			parts := strings.Split(ref, ".")
			valid := len(parts) == 2 && parts[0] == "input" && fields[parts[1]]
			if len(parts) == 3 && parts[0] == "nodes" && ancestors[parts[1]] {
				if idx, ok := ids[parts[1]]; ok {
					kind := d.Nodes[idx].Kind
					valid = (kind == "input" && parts[2] == "text") || (kind == "llm" && parts[2] == "prompt")
				}
			}
			if !valid {
				add("UNRESOLVED_TEMPLATE_VARIABLE", "Variable cannot be resolved before this node: "+ref, fmt.Sprintf("/definition/nodes/%d/promptTemplate", i), n.ID)
			}
		}
		remaining := variable.ReplaceAllString(n.PromptTemplate, "")
		if strings.Contains(remaining, "{{") || strings.Contains(remaining, "}}") {
			add("UNRESOLVED_TEMPLATE_VARIABLE", "Malformed template variable", fmt.Sprintf("/definition/nodes/%d/promptTemplate", i), n.ID)
		}
	}
	return out
}
func outputPort(kind string) string {
	switch kind {
	case "input":
		return "text"
	case "llm":
		return "prompt"
	case "image":
		return "image"
	}
	return ""
}
func inputPorts(kind string) []string {
	switch kind {
	case "llm":
		return []string{"text", "prompt"}
	case "image":
		return []string{"prompt"}
	case "output":
		return []string{"text", "prompt", "image"}
	}
	return nil
}
func References(template string) []string {
	out := []string{}
	for _, m := range variable.FindAllStringSubmatch(template, -1) {
		out = append(out, m[1])
	}
	return out
}
func Render(template string, input domain.Object, nodes map[string]domain.Object) (string, error) {
	var failed string
	result := variable.ReplaceAllStringFunc(template, func(token string) string {
		ref := variable.FindStringSubmatch(token)[1]
		parts := strings.Split(ref, ".")
		var v any
		var ok bool
		if len(parts) == 2 && parts[0] == "input" {
			v, ok = input[parts[1]]
		} else if len(parts) == 3 && parts[0] == "nodes" {
			v, ok = nodes[parts[1]][parts[2]]
		}
		if !ok || v == nil {
			failed = ref
			return token
		}
		switch x := v.(type) {
		case string:
			return x
		case json.Number:
			return x.String()
		case bool:
			return fmt.Sprint(x)
		case float64, int:
			return fmt.Sprint(x)
		default:
			failed = ref
			return token
		}
	})
	if failed != "" {
		return "", fmt.Errorf("unresolved scalar variable: %s", failed)
	}
	return result, nil
}
func Order(d domain.Definition) []domain.Node {
	byID := map[string]domain.Node{}
	next := map[string]string{}
	start := ""
	for _, n := range d.Nodes {
		byID[n.ID] = n
		if n.Kind == "input" {
			start = n.ID
		}
	}
	for _, e := range d.Edges {
		next[e.From] = e.To
	}
	out := []domain.Node{}
	seen := map[string]bool{}
	for start != "" && !seen[start] {
		seen[start] = true
		out = append(out, byID[start])
		start = next[start]
	}
	return out
}
func compile(schema map[string]any) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	if e := c.AddResource("input.json", schema); e != nil {
		return nil, e
	}
	return c.Compile("input.json")
}
func validateInputSchema(schema map[string]any) []string {
	out := []string{}
	for key := range schema {
		if !contains([]string{"type", "properties", "required", "additionalProperties"}, key) {
			out = append(out, "Unsupported inputSchema keyword: "+key)
		}
	}
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		out = append(out, "inputSchema must be an object with additionalProperties: false")
	}
	props := object(schema["properties"])
	if required, ok := schema["required"].([]any); ok {
		for _, value := range required {
			key, yes := value.(string)
			if !yes || props[key] == nil {
				out = append(out, "Required input field is not declared: "+fmt.Sprint(value))
			}
		}
	}
	if len(props) == 0 || len(props) > 20 {
		out = append(out, "inputSchema requires 1–20 flat properties")
	}
	for name, v := range props {
		p := object(v)
		for key := range p {
			if !contains([]string{"type", "minLength", "maxLength", "minimum", "maximum", "enum"}, key) {
				out = append(out, "Unsupported scalar schema keyword: "+name+"."+key)
			}
		}
		if !identifier.MatchString(name) || !contains([]string{"string", "number", "integer", "boolean"}, fmt.Sprint(p["type"])) {
			out = append(out, "Input fields must use valid names and scalar types: "+name)
		}
	}
	if len(out) > 0 {
		return out
	}
	if _, err := compile(schema); err != nil {
		out = append(out, "Invalid JSON Schema: "+err.Error())
	}
	return out
}
func ValidateInput(n domain.Node, input domain.Object) error {
	s, err := compile(object(n.Params["inputSchema"]))
	if err != nil {
		return err
	}
	return s.Validate(map[string]any(input))
}
