package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flowkit/internal/direct"
)

type evidence struct {
	ID           string               `json:"id"`
	StartedAt    time.Time            `json:"started_at"`
	FinishedAt   *time.Time           `json:"finished_at,omitempty"`
	Input        string               `json:"input"`
	LLMRequest   direct.LLMRequest    `json:"llm_request"`
	LLMResponse  *direct.LLMResponse  `json:"llm_response,omitempty"`
	ImageModel   string               `json:"image_model"`
	ImageRequest *direct.ImageRequest `json:"image_request,omitempty"`
	Queue        *direct.QueueHandle  `json:"queue,omitempty"`
	QueueStatus  *direct.QueueStatus  `json:"queue_status,omitempty"`
	ImageResult  *direct.ImageResult  `json:"image_result,omitempty"`
	Error        string               `json:"error,omitempty"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		message := redact(err.Error())
		slog.Error(message, "run_id", "direct", "node_id", "setup_or_provider", "attempt", 1)
		os.Exit(1)
	}
}

func run() error {
	input := flag.String("input", "Matte black ceramic coffee mug with a wooden handle", "Product description")
	dryRun := flag.Bool("dry-run", false, "Print requests without calling providers")
	resume := flag.String("resume", "", "Resume polling from an existing evidence.json; never resubmit")
	flag.Parse()
	if err := loadEnv(".env"); err != nil {
		return err
	}
	llmModel, imageModel := os.Getenv("OPENROUTER_MODEL"), os.Getenv("FAL_MODEL")
	if *resume == "" && (llmModel == "" || imageModel == "") {
		return fmt.Errorf("set OPENROUTER_MODEL and FAL_MODEL in .env; models are never guessed")
	}
	if *dryRun {
		if *resume != "" {
			return fmt.Errorf("choose -dry-run or -resume")
		}
		return printJSON(map[string]any{"llm_request": direct.PromptRequest(llmModel, *input), "fal_model": imageModel, "fal_request": direct.ImagePayload("<OpenRouter response prompt>"), "network_calls": 0})
	}
	if os.Getenv("FAL_KEY") == "" || (*resume == "" && os.Getenv("OPENROUTER_API_KEY") == "") {
		return fmt.Errorf("fill local .env keys FAL_KEY and OPENROUTER_API_KEY before a real run")
	}
	client := direct.New(os.Getenv("OPENROUTER_API_KEY"), os.Getenv("FAL_KEY"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var record evidence
	var path string
	if *resume != "" {
		path = *resume
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(payload, &record); err != nil {
			return err
		}
		if record.ImageResult != nil {
			return fmt.Errorf("this experiment already has a result; no new call was sent")
		}
		if record.Queue == nil {
			return fmt.Errorf("no persisted fal request to resume")
		}
		if record.LLMResponse == nil || record.ImageRequest == nil || record.ImageModel == "" {
			return fmt.Errorf("incomplete experiment metadata; refusing to resume")
		}
		if err := client.ValidateHandle(*record.Queue); err != nil {
			return err
		}
		record.Error = ""
	} else {
		record = evidence{ID: time.Now().UTC().Format("20060102T150405.000000000Z"), StartedAt: time.Now().UTC(), Input: *input, ImageModel: imageModel, LLMRequest: direct.PromptRequest(llmModel, *input)}
		path = filepath.Join("artifacts", "direct", record.ID, "evidence.json")
		if err := save(path, record); err != nil {
			return err
		}
	}
	logger := slog.With("run_id", record.ID, "attempt", 1)
	// The final error is persisted without exposing the keys, including provider error bodies.
	finishError := func(err error) error {
		record.Error = redact(err.Error())
		if saveErr := save(path, record); saveErr != nil {
			return errors.Join(err, saveErr)
		}
		return err
	}
	if *resume == "" {
		logger.Info("generating prompt", "node_id", "llm", "model", llmModel)
		response, err := client.GeneratePrompt(ctx, record.LLMRequest)
		record.LLMResponse = &response
		if err != nil {
			return finishError(err)
		}
		payload := direct.ImagePayload(strings.TrimSpace(response.Choices[0].Message.Content))
		record.ImageRequest = &payload
		if err := save(path, record); err != nil {
			return err
		}
		logger.Info("submitting image request", "node_id", "image", "model", record.ImageModel)
		handle, err := client.Submit(ctx, record.ImageModel, payload)
		// Persist the ID even if a returned tracking URL is malformed.
		if handle.RequestID != "" {
			record.Queue = &handle
		}
		if err != nil {
			return finishError(err)
		}
		if err := save(path, record); err != nil {
			return err
		}
		logger.Info("image request persisted", "node_id", "image", "provider_request_id", handle.RequestID, "evidence", path)
	}
	for {
		status, err := client.Status(ctx, *record.Queue)
		record.QueueStatus = &status
		if err != nil {
			return finishError(err)
		}
		if err := save(path, record); err != nil {
			return err
		}
		logger.Info("image queue status", "node_id", "image", "status", status.Status, "provider_request_id", record.Queue.RequestID)
		if status.Status == "COMPLETED" {
			break
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return finishError(ctx.Err())
		case <-timer.C:
		}
	}
	result, err := client.Result(ctx, *record.Queue)
	if err != nil {
		return finishError(err)
	}
	record.ImageResult = &result
	finishedAt := time.Now().UTC()
	record.FinishedAt = &finishedAt
	if err := save(path, record); err != nil {
		return err
	}
	logger.Info("direct experiment completed", "node_id", "image", "model", record.ImageModel, "duration_ms", finishedAt.Sub(record.StartedAt).Milliseconds())
	return printJSON(map[string]any{"evidence": path, "llm_model": record.LLMResponse.Model, "image_model": record.ImageModel, "prompt": record.ImageRequest.Prompt, "image_url": result.Images[0].URL, "provider_request_id": record.Queue.RequestID})
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func save(path string, record evidence) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	// Redact defensively if a provider echoes a secret in a response.
	payload = []byte(redact(string(payload)))
	temporary, err := os.CreateTemp(filepath.Dir(path), ".evidence-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temporary.Name()
	defer os.Remove(tempPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(payload); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func redact(value string) string {
	for _, name := range []string{"OPENROUTER_API_KEY", "FAL_KEY"} {
		if key := os.Getenv(name); key != "" {
			value = strings.ReplaceAll(value, key, "[REDACTED]")
		}
	}
	return value
}

func loadEnv(path string) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf(".env expects KEY=value lines")
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
