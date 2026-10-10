// Package direct is the exploratory provider client, before migration to FlowKit.
package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const SystemPrompt = "Write a concise English image-generation prompt for a commercial product photograph. Preserve the described product. Describe composition, lighting and a simple studio background. Do not include text, logos, commentary or markdown. Return only the prompt, at most 100 words."

type Client struct {
	HTTP           *http.Client
	OpenRouterBase string
	FalBase        string
	OpenRouterKey  string
	FalKey         string
}

func New(openRouterKey, falKey string) *Client {
	return &Client{
		HTTP:           &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		OpenRouterBase: "https://openrouter.ai/api/v1",
		FalBase:        "https://queue.fal.run",
		OpenRouterKey:  openRouterKey, FalKey: falKey,
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Provider    map[string]bool `json:"provider"`
}

func PromptRequest(model, input string) LLMRequest {
	return LLMRequest{Model: model, Messages: []Message{{Role: "system", Content: SystemPrompt}, {Role: "user", Content: input}}, MaxTokens: 160, Temperature: 0.4, Provider: map[string]bool{"allow_fallbacks": false}}
}

type LLMResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage json.RawMessage `json:"usage,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

func (c *Client) GeneratePrompt(ctx context.Context, request LLMRequest) (LLMResponse, error) {
	var response LLMResponse
	err := c.request(ctx, http.MethodPost, c.OpenRouterBase+"/chat/completions", "Bearer "+c.OpenRouterKey, request, &response)
	if err != nil {
		return response, err
	}
	if len(response.Error) > 0 && string(response.Error) != "null" {
		return response, fmt.Errorf("OpenRouter returned an error: %s", response.Error)
	}
	if response.Model == "" || len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return response, fmt.Errorf("OpenRouter returned no model or prompt")
	}
	if response.Choices[0].FinishReason != "stop" {
		return response, fmt.Errorf("OpenRouter prompt did not finish normally: %s", response.Choices[0].FinishReason)
	}
	return response, nil
}

type ImageSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ImageRequest struct {
	Prompt       string    `json:"prompt"`
	ImageSize    ImageSize `json:"image_size"`
	NumImages    int       `json:"num_images"`
	OutputFormat string    `json:"output_format"`
}

func ImagePayload(prompt string) ImageRequest {
	return ImageRequest{Prompt: prompt, ImageSize: ImageSize{Width: 512, Height: 512}, NumImages: 1, OutputFormat: "jpeg"}
}

type QueueHandle struct {
	RequestID   string `json:"request_id"`
	StatusURL   string `json:"status_url"`
	ResponseURL string `json:"response_url"`
}

func (c *Client) Submit(ctx context.Context, model string, request ImageRequest) (QueueHandle, error) {
	var handle QueueHandle
	if strings.ContainsAny(model, "?#") || strings.HasPrefix(model, "/") || strings.Contains(model, "..") || strings.TrimSpace(model) == "" {
		return handle, fmt.Errorf("invalid fal model path")
	}
	err := c.request(ctx, http.MethodPost, c.FalBase+"/"+model, "Key "+c.FalKey, request, &handle)
	if err != nil {
		return handle, err
	}
	if handle.RequestID == "" {
		return handle, fmt.Errorf("fal submit returned no request ID")
	}
	if err := c.ValidateHandle(handle); err != nil {
		return handle, err
	}
	return handle, nil
}

// Follow URLs returned by fal: request retrieval paths need not equal submit paths.
// Restrict the origin to avoid sending credentials to an untrusted host.
func (c *Client) ValidateHandle(handle QueueHandle) error {
	if handle.RequestID == "" {
		return fmt.Errorf("missing fal request ID")
	}
	base, err := url.Parse(c.FalBase)
	if err != nil {
		return err
	}
	for _, address := range []string{handle.StatusURL, handle.ResponseURL} {
		u, err := url.Parse(address)
		if err != nil || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil || u.Path == "" || u.Fragment != "" {
			return fmt.Errorf("untrusted fal queue URL")
		}
	}
	return nil
}

type QueueStatus struct {
	Status    string          `json:"status"`
	Error     json.RawMessage `json:"error,omitempty"`
	ErrorType string          `json:"error_type,omitempty"`
}

func (c *Client) Status(ctx context.Context, handle QueueHandle) (QueueStatus, error) {
	var status QueueStatus
	if err := c.ValidateHandle(handle); err != nil {
		return status, err
	}
	err := c.request(ctx, http.MethodGet, handle.StatusURL, "Key "+c.FalKey, nil, &status)
	if err != nil {
		return status, err
	}
	if len(status.Error) > 0 && string(status.Error) != "null" {
		return status, fmt.Errorf("fal terminal error (%s): %s", status.ErrorType, status.Error)
	}
	switch status.Status {
	case "IN_QUEUE", "IN_PROGRESS", "COMPLETED":
		return status, nil
	default:
		return status, fmt.Errorf("unknown fal status: %q", status.Status)
	}
}

type ImageResult struct {
	Images []struct {
		URL         string `json:"url"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		ContentType string `json:"content_type"`
	} `json:"images"`
	Seed   int64  `json:"seed"`
	Prompt string `json:"prompt"`
}

func (c *Client) Result(ctx context.Context, handle QueueHandle) (ImageResult, error) {
	var result ImageResult
	if err := c.ValidateHandle(handle); err != nil {
		return result, err
	}
	err := c.request(ctx, http.MethodGet, handle.ResponseURL, "Key "+c.FalKey, nil, &result)
	if err != nil {
		return result, err
	}
	if len(result.Images) == 0 || result.Images[0].URL == "" {
		return result, fmt.Errorf("fal returned no image")
	}
	return result, nil
}

type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("provider HTTP %d: %s", e.Status, e.Body) }

func (c *Client) request(ctx context.Context, method, address, auth string, body, target any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, address, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	response, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	const maxBody = 2 * 1024 * 1024
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return err
	}
	if len(payload) > maxBody {
		return fmt.Errorf("provider response too large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &HTTPError{Status: response.StatusCode, Body: string(payload)}
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("invalid provider JSON: %w", err)
	}
	return nil
}
