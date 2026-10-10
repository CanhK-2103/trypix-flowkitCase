package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type exchange struct {
	At       time.Time       `json:"at"`
	Method   string          `json:"method"`
	Path     string          `json:"path"`
	Body     any             `json:"body,omitempty"`
	Status   int             `json:"http_status"`
	Response json.RawMessage `json:"response"`
}
type proof struct {
	GitSHA     string `json:"git_sha"`
	Containers string `json:"containers"`
}
type manifest struct {
	Session  string          `json:"session_id"`
	URL      string          `json:"api_url"`
	Step     int             `json:"completed_step"`
	Phase    string          `json:"phase"`
	Before   proof           `json:"before"`
	After    *proof          `json:"after,omitempty"`
	Requests []exchange      `json:"requests"`
	Runs     []*domain.Run   `json:"runs"`
	Direct   json.RawMessage `json:"direct_experiment"`
	Error    string          `json:"error,omitempty"`
}
type demo struct {
	url, path, project string
	client             *http.Client
	provider           *providers.Client
	record             manifest
}

func main() {
	if e := run(); e != nil {
		telemetry.Logger("acceptance", "cli").Error("Acceptance demo stopped", "event", "acceptance.failed", "error", e.Error())
		os.Exit(1)
	}
}
func (d *demo) save() error {
	if e := os.MkdirAll(filepath.Dir(d.path), 0700); e != nil {
		return e
	}
	b := domain.JSON(d.record)
	temp := d.path + ".tmp"
	if e := os.WriteFile(temp, b, 0600); e != nil {
		return e
	}
	return os.Rename(temp, d.path)
}
func (d *demo) request(ctx context.Context, method, path string, body any, key string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(domain.JSON(body))
	}
	req, e := http.NewRequestWithContext(ctx, method, d.url+path, reader)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, e := d.client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	data, e := io.ReadAll(io.LimitReader(res.Body, 8*1024*1024))
	if e != nil {
		return nil, e
	}
	if method != "GET" {
		d.record.Requests = append(d.record.Requests, exchange{At: time.Now().UTC(), Method: method, Path: path, Body: body, Status: res.StatusCode, Response: data})
		if e := d.save(); e != nil {
			return nil, e
		}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return data, fmt.Errorf("%s %s HTTP %d: %s", method, path, res.StatusCode, string(data))
	}
	return data, nil
}
func readExample(path string, v any) error {
	b, e := examples.Files.ReadFile(path)
	if e != nil {
		return e
	}
	return domain.Decode(b, v)
}
func (d *demo) model(ctx context.Context, m domain.Model) error {
	data, e := d.request(ctx, "GET", "/models/"+m.Alias, nil, "")
	if e == nil {
		var current domain.Model
		_ = domain.Decode(data, &current)
		if current.ModelID != m.ModelID || current.Provider != m.Provider || !current.Enabled {
			return fmt.Errorf("existing alias %s differs from expected model", m.Alias)
		}
		return nil
	}
	_, e = d.request(ctx, "POST", "/models", m, "")
	return e
}
func (d *demo) workflow(ctx context.Context, w domain.Workflow) error {
	if _, e := d.request(ctx, "GET", "/workflows/"+w.ID, nil, ""); e == nil {
		return nil
	}
	_, e := d.request(ctx, "POST", "/workflows", w, "")
	return e
}
func snapshot(project string) (proof, error) {
	git, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil {
		return proof{}, e
	}
	containers, e := exec.Command("docker", "compose", "-p", project, "ps", "-q", "api", "worker").Output()
	if e != nil {
		return proof{}, e
	}
	ids := strings.Fields(string(containers))
	if len(ids) != 2 {
		return proof{}, fmt.Errorf("expected API and worker containers in project %s", project)
	}
	args := []string{"inspect", "--format", `{{.Id}}|{{.State.StartedAt}}|{{.Image}}`}
	args = append(args, ids...)
	info, e := exec.Command("docker", args...).Output()
	if e != nil {
		return proof{}, e
	}
	return proof{GitSHA: strings.TrimSpace(string(git)), Containers: strings.TrimSpace(string(info))}, nil
}
func directBudget(raw json.RawMessage) (decimal.Decimal, decimal.Decimal, error) {
	var record struct {
		Phase       string               `json:"phase"`
		FalEstimate string               `json:"fal_estimated_cost_usd"`
		Completion  providers.Completion `json:"completion"`
		Models      []domain.Model       `json:"models"`
	}
	if e := domain.Decode(raw, &record); e != nil {
		return decimal.Zero, decimal.Zero, e
	}
	if record.Phase != "complete" {
		return decimal.Zero, decimal.Zero, fmt.Errorf("direct experiment must be complete before FlowKit live demo")
	}
	fal, e := decimal.NewFromString(record.FalEstimate)
	if e != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("direct fal cost unavailable")
	}
	var cost domain.Cost
	for _, m := range record.Models {
		if m.Provider == "openrouter" {
			cost = engine.TokenCost(m, record.Completion.Usage)
		}
	}
	amount := cost.Reported
	if amount == nil {
		amount = cost.Estimated
	}
	if amount == nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("direct LLM cost unavailable")
	}
	or, e := decimal.NewFromString(*amount)
	return fal, or, e
}
func (d *demo) budget(ctx context.Context, image, llm domain.Model) error {
	fal, or, e := directBudget(d.record.Direct)
	if e != nil {
		return e
	}
	for _, run := range d.record.Runs {
		for _, n := range run.Nodes {
			for _, request := range n.Requests {
				amount := request.Cost.Estimated
				if request.Provider == "openrouter" && request.Cost.Reported != nil {
					amount = request.Cost.Reported
				}
				if amount == nil {
					return fmt.Errorf("prior production cost unknown; review before another live run")
				}
				v, e := decimal.NewFromString(*amount)
				if e != nil {
					return e
				}
				if request.Provider == "fal" {
					fal = fal.Add(v)
				} else {
					or = or.Add(v)
				}
			}
		}
	}
	price, f := d.provider.Estimate(ctx, image)
	if f != nil {
		return f
	}
	estimate, e := decimal.NewFromString(price)
	if e != nil || estimate.IsNegative() {
		return fmt.Errorf("invalid fal estimate")
	}
	if fal.Add(estimate.Mul(decimal.NewFromInt(3))).GreaterThan(decimal.RequireFromString("0.50")) {
		return fmt.Errorf("fal planned cost plus three attempts exceeds $0.50 demo threshold")
	}
	input, e1 := decimal.NewFromString(llm.Pricing.InputPerMillion)
	output, e2 := decimal.NewFromString(llm.Pricing.OutputPerMillion)
	if e1 != nil || e2 != nil {
		return fmt.Errorf("LLM pricing unavailable")
	}
	reserve := input.Mul(decimal.NewFromInt(1000)).Add(output.Mul(decimal.NewFromInt(300))).Div(decimal.NewFromInt(1000000)).Mul(decimal.NewFromInt(3))
	if or.Add(reserve).GreaterThan(decimal.RequireFromString("0.05")) {
		return fmt.Errorf("OpenRouter planned cost exceeds $0.05 demo threshold")
	}
	return nil
}
func (d *demo) produce(ctx context.Context, wfID, label, description string, image, llm domain.Model) error {
	if e := d.budget(ctx, image, llm); e != nil {
		return e
	}
	data, e := d.request(ctx, "POST", "/workflows/"+wfID+"/runs", domain.Object{"input": domain.Object{"description": description}}, d.record.Session+"-"+label)
	if e != nil {
		return e
	}
	var accepted struct {
		ID string `json:"run_id"`
	}
	if e := domain.Decode(data, &accepted); e != nil {
		return e
	}
	for {
		data, e = d.request(ctx, "GET", "/runs/"+accepted.ID, nil, "")
		if e != nil {
			return e
		}
		var run domain.Run
		if e := domain.Decode(data, &run); e != nil {
			return e
		}
		if run.Terminal() {
			already := false
			for _, old := range d.record.Runs {
				if old.ID == run.ID {
					already = true
				}
			}
			if !already {
				d.record.Runs = append(d.record.Runs, &run)
			}
			if e := d.save(); e != nil {
				return e
			}
			if run.Status != "succeeded" {
				return fmt.Errorf("run %s failed at %s: %v", run.ID, run.FailedNodeID, run.Error)
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
func run() error {
	url := flag.String("url", "http://localhost:8081", "Empty acceptance API URL")
	project := flag.String("project", "flowkit-acceptance", "Compose project for no-restart proof")
	path := flag.String("evidence", "evidence/local/acceptance.json", "Checkpoint/evidence file")
	direct := flag.String("direct", "evidence/local/direct.json", "Completed direct experiment evidence")
	resume := flag.Bool("resume", false, "Resume existing demo with the same idempotency keys")
	flag.Parse()
	if e := config.LoadDotEnv(".env"); e != nil {
		return e
	}
	cfg := config.Load()
	if cfg.OpenRouterKey == "" || cfg.FalKey == "" {
		return fmt.Errorf("local OPENROUTER_API_KEY and FAL_KEY required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	d := &demo{url: strings.TrimRight(*url, "/"), path: *path, project: *project, client: &http.Client{Timeout: 30 * time.Second}, provider: providers.New(cfg)}
	if b, e := os.ReadFile(*path); e == nil {
		if !*resume {
			return fmt.Errorf("existing evidence requires -resume; no new production started")
		}
		if e := domain.Decode(b, &d.record); e != nil {
			return e
		}
		if d.record.URL != d.url || d.record.Phase == "complete" {
			return fmt.Errorf("demo URL differs or experiment already complete")
		}
	} else {
		directBytes, e := os.ReadFile(*direct)
		if e != nil {
			return e
		}
		if _, _, e := directBudget(directBytes); e != nil {
			return e
		}
		data, e := d.request(ctx, "GET", "/workflows?limit=1", nil, "")
		if e != nil {
			return e
		}
		var page struct {
			Items []any `json:"items"`
		}
		if e := domain.Decode(data, &page); e != nil {
			return e
		}
		if len(page.Items) != 0 {
			return fmt.Errorf("acceptance requires a separate empty database with LOAD_EXAMPLES=false")
		}
		before, e := snapshot(*project)
		if e != nil {
			return e
		}
		d.record = manifest{Session: uuid.NewString(), URL: d.url, Phase: "running", Before: before, Direct: directBytes, Requests: []exchange{}, Runs: []*domain.Run{}}
		if e := d.save(); e != nil {
			return e
		}
	}
	var llm, schnell, dev domain.Model
	var product, event domain.Workflow
	for p, v := range map[string]any{"models/prompt-writer.json": &llm, "models/flux-schnell.json": &schnell, "models/flux-dev.json": &dev, "workflows/product.json": &product, "workflows/event.json": &event} {
		if e := readExample(p, v); e != nil {
			return e
		}
	}
	var e error
	llm, e = d.provider.PublicModel(ctx, llm)
	if e != nil {
		return e
	}
	steps := []func() error{
		func() error {
			for _, m := range []domain.Model{llm, schnell} {
				if e := d.model(ctx, m); e != nil {
					return e
				}
			}
			return d.workflow(ctx, product)
		},
		func() error {
			return d.produce(ctx, product.ID, "schnell", "Matte blue ceramic coffee mug with a wooden handle", schnell, llm)
		},
		func() error {
			if e := d.model(ctx, dev); e != nil {
				return e
			}
			product.Definition.Nodes[2].ModelRef = dev.Alias
			_, e := d.request(ctx, "PUT", "/workflows/"+product.ID, product, "")
			return e
		},
		func() error {
			return d.produce(ctx, product.ID, "switched-dev", "Matte blue ceramic coffee mug with a wooden handle", dev, llm)
		},
		func() error {
			product.Definition.Nodes[2].ModelRef = schnell.Alias
			if _, e := d.request(ctx, "PUT", "/workflows/"+product.ID, product, ""); e != nil {
				return e
			}
			return d.workflow(ctx, event)
		},
		func() error {
			return d.produce(ctx, event.ID, "event-dev", "A nighttime jazz concert in a garden with warm lanterns", dev, llm)
		},
	}
	for i := d.record.Step; i < len(steps); i++ {
		if e := steps[i](); e != nil {
			d.record.Error = e.Error()
			_ = d.save()
			return e
		}
		d.record.Step = i + 1
		d.record.Error = ""
		if e := d.save(); e != nil {
			return e
		}
	}
	after, e := snapshot(*project)
	if e != nil {
		return e
	}
	d.record.After = &after
	if after != d.record.Before {
		d.record.Error = "Git SHA, container identity, image or StartedAt changed during demo"
		_ = d.save()
		return fmt.Errorf("no-restart proof failed")
	}
	d.record.Phase = "complete"
	if e := d.save(); e != nil {
		return e
	}
	telemetry.Logger("acceptance", "cli").Info("Real acceptance completed with unchanged containers/code", "event", "acceptance.completed", "evidence", d.path, "runs", len(d.record.Runs))
	return nil
}
