package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/shopspring/decimal"
	"os"
	"path/filepath"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/workflow"
)

func read(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return domain.Decode(b, v)
}
func write(path string, v any) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	b := domain.JSON(v)
	temp := path + ".tmp"
	if e := os.WriteFile(temp, b, 0600); e != nil {
		return e
	}
	return os.Rename(temp, path)
}
func main() {
	if err := execute(); err != nil {
		telemetry.Logger("direct", "cli").Error("Direct experiment failed", "event", "direct.failed", "error", err.Error())
		os.Exit(1)
	}
}
func execute() error {
	input := flag.String("input", "Matte blue ceramic coffee mug with a wooden handle", "Product description")
	evidence := flag.String("evidence", "evidence/local/direct.json", "Evidence/checkpoint file")
	resume := flag.Bool("resume", false, "Resume a previously saved fal ticket without resubmission")
	flag.Parse()
	if e := config.LoadDotEnv(".env"); e != nil {
		return e
	}
	cfg := config.Load()
	if cfg.OpenRouterKey == "" || cfg.FalKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY and FAL_KEY must be set locally in .env")
	}
	var llm, image domain.Model
	var wf domain.Workflow
	for p, v := range map[string]any{"examples/models/prompt-writer.json": &llm, "examples/models/flux-schnell.json": &image, "examples/workflows/product.json": &wf} {
		if e := read(p, v); e != nil {
			return e
		}
	}
	client := providers.New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	record := domain.Object{}
	ticket := providers.Ticket{}
	if _, err := os.Stat(*evidence); err == nil {
		if !*resume {
			return fmt.Errorf("evidence file already exists; use -resume to avoid duplicate paid generation")
		}
		if err := read(*evidence, &record); err != nil {
			return err
		}
		if record["phase"] == "complete" {
			return fmt.Errorf("experiment already complete; no request sent")
		}
		ticketMap, ok := record["ticket"].(map[string]any)
		if !ok {
			return fmt.Errorf("no durable fal ticket; submission outcome may be uncertain, do not retry blindly")
		}
		if e := domain.Decode(domain.JSON(ticketMap), &ticket); e != nil {
			return e
		}
		var models []domain.Model
		if e := domain.Decode(domain.JSON(record["models"]), &models); e != nil {
			return e
		}
		for _, snapshot := range models {
			if snapshot.Provider == "fal" {
				image = snapshot
			}
			if snapshot.Provider == "openrouter" {
				llm = snapshot
			}
		}
	} else {
		var err error
		llm, err = client.PublicModel(ctx, llm)
		if err != nil {
			return err
		}
		prompt, err := workflow.Render(wf.Definition.Nodes[1].PromptTemplate, domain.Object{"description": *input}, nil)
		if err != nil {
			return err
		}
		price, f := client.Estimate(ctx, image)
		if f != nil {
			return f
		}
		priceValue, e := decimal.NewFromString(price)
		if e != nil || priceValue.IsNegative() || priceValue.GreaterThan(decimal.RequireFromString("0.50")) {
			return fmt.Errorf("fal estimate exceeds demo review threshold or is invalid: %s", price)
		}
		inputRate, e1 := decimal.NewFromString(llm.Pricing.InputPerMillion)
		outputRate, e2 := decimal.NewFromString(llm.Pricing.OutputPerMillion)
		if e1 != nil || e2 != nil || inputRate.Mul(decimal.NewFromInt(1000)).Add(outputRate.Mul(decimal.NewFromInt(300))).Div(decimal.NewFromInt(1000000)).GreaterThan(decimal.RequireFromString("0.05")) {
			return fmt.Errorf("OpenRouter estimated request exceeds demo review threshold")
		}
		record["fal_estimated_cost_usd"] = price
		record["started_at"] = time.Now().UTC()
		record["input"] = domain.Object{"description": *input}
		record["workflow"] = wf
		record["models"] = []domain.Model{llm, image}
		record["rendered_prompt"] = prompt
		record["llm_params"] = workflow.Merge(llm.Defaults, wf.Definition.Nodes[1].Params)
		record["phase"] = "llm_sending"
		if e := write(*evidence, record); e != nil {
			return e
		}
		response, f := client.Complete(ctx, llm, prompt, workflow.Merge(llm.Defaults, wf.Definition.Nodes[1].Params))
		record["completion"] = response
		if f != nil {
			record["error"] = f
			_ = write(*evidence, record)
			return f
		}
		record["phase"] = "fal_sending"
		record["image_params"] = workflow.Merge(image.Defaults, wf.Definition.Nodes[2].Params)
		if e := write(*evidence, record); e != nil {
			return e
		}
		ticket, f = client.Submit(ctx, image, response.Prompt, workflow.Merge(image.Defaults, wf.Definition.Nodes[2].Params))
		if ticket.ID != "" {
			record["ticket"] = ticket
		}
		if f != nil {
			record["error"] = f
			_ = write(*evidence, record)
			return f
		}
		record["ticket"] = ticket
		record["phase"] = "fal_waiting"
		if e := write(*evidence, record); e != nil {
			return e
		}
	}
	log := telemetry.Logger("direct", "cli")
	log.Info("Fal ticket saved before polling", "event", "direct.provider_checkpoint", "provider_request_id", ticket.ID, "actual_model_id", image.ModelID)
	for {
		state, f := client.Status(ctx, ticket)
		if f != nil {
			return f
		}
		if state.Error != nil {
			return state.Error
		}
		if state.Status == "COMPLETED" {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	result, f := client.Result(ctx, ticket)
	if f != nil {
		return f
	}
	record["result"] = result
	record["phase"] = "complete"
	record["finished_at"] = time.Now().UTC()
	if e := write(*evidence, record); e != nil {
		return e
	}
	log.Info("Direct experiment completed", "event", "direct.completed", "provider_request_id", ticket.ID, "actual_model_id", image.ModelID, "evidence", *evidence)
	return nil
}
