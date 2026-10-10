package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
)

type Client struct {
	HTTP   *http.Client
	Config config.Config
}

func New(c config.Config) *Client {
	return &Client{Config: c, HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

type Completion struct {
	ID     string          `json:"id"`
	Model  string          `json:"model"`
	Prompt string          `json:"prompt"`
	Raw    json.RawMessage `json:"response"`
	Usage  domain.Object   `json:"usage"`
}
type Ticket struct {
	ID          string          `json:"provider_request_id"`
	StatusURL   string          `json:"status_url"`
	ResponseURL string          `json:"response_url"`
	Raw         json.RawMessage `json:"response"`
}
type QueueState struct {
	Status string
	Error  *domain.Failure
	Raw    json.RawMessage
}
type ImageResult struct {
	URL           string
	Width, Height int
	Raw           json.RawMessage
	BillableUnits string
}

func (c *Client) request(ctx context.Context, method, endpoint, key string, payload any, submission bool, fal bool) ([]byte, http.Header, *domain.Failure) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(domain.JSON(payload))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, nil, &domain.Failure{Code: "INVALID_INPUT", Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if fal {
		req.Header.Set("Authorization", "Key "+key)
		req.Header.Set("X-Fal-No-Retry", "1")
		req.Header.Set("x-app-fal-disable-fallback", "true")
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	var wrote atomic.Bool
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{WroteRequest: func(httptrace.WroteRequestInfo) { wrote.Store(true) }}))
	res, err := c.HTTP.Do(req)
	if err != nil {
		failure := &domain.Failure{Code: "PROVIDER_NETWORK_ERROR", Message: "Provider connection failed", ProviderMessage: clean(err.Error()), Retryable: true}
		if key != "" {
			failure.ProviderMessage = strings.ReplaceAll(failure.ProviderMessage, key, "[REDACTED]")
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			failure.Code = "PROVIDER_TIMEOUT"
			failure.Message = "Provider request timed out"
		}
		if submission && wrote.Load() {
			failure.Code = "PROVIDER_SUBMISSION_UNKNOWN"
			failure.Message = "Submission may have been accepted; no safe response was received"
			failure.Uncertain = true
			failure.Retryable = false
		}
		return nil, nil, failure
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return nil, res.Header, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Provider response unreadable or exceeds 2 MiB", Uncertain: submission}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		failure := classify(res.StatusCode, data, res.Header)
		if key != "" {
			failure.ProviderMessage = strings.ReplaceAll(failure.ProviderMessage, key, "[REDACTED]")
		}
		return data, res.Header, failure
	}
	return data, res.Header, nil
}
func clean(s string) string {
	if len(s) > 2048 {
		s = s[:2048]
	}
	return s
}
func classify(status int, data []byte, headers http.Header) *domain.Failure {
	var body domain.Object
	_ = domain.Decode(data, &body)
	msg := string(data)
	code := headers.Get("X-Fal-Error-Type")
	if v, ok := body["error_type"].(string); ok {
		code = v
	}
	if v, ok := body["error"].(map[string]any); ok {
		if m, ok := v["message"].(string); ok {
			msg = m
		}
		if n, ok := v["code"]; ok {
			code = fmt.Sprint(n)
		}
		if meta, ok := v["metadata"].(map[string]any); ok {
			if meta["error_type"] == "refusal" {
				code = "refusal"
			}
			if m, ok := meta["provider_code"].(string); ok && code != "refusal" {
				code = m
			}
		}
	}
	f := &domain.Failure{Code: "INVALID_INPUT", Message: "Provider rejected the request", ProviderCode: code, ProviderMessage: clean(msg), HTTPStatus: status}
	switch {
	case status == 403 && code == "refusal":
		f.Code = "PROVIDER_CONTENT_REJECTED"
	case status == 401 || status == 403:
		f.Code = "PROVIDER_AUTH_ERROR"
	case status == 402:
		f.Code = "PROVIDER_PAYMENT_ERROR"
	case status == 404:
		f.Code = "PROVIDER_MODEL_NOT_FOUND"
	case status == 408 || status == 504 || status == 524:
		f.Code = "PROVIDER_TIMEOUT"
		f.Retryable = true
	case status == 429:
		f.Code = "PROVIDER_RATE_LIMIT"
		f.Retryable = true
	case status >= 500:
		f.Code = "PROVIDER_5XX"
		f.Retryable = true
	}
	if s := headers.Get("Retry-After"); s != "" {
		if seconds, e := strconv.Atoi(s); e == nil {
			f.RetryAfterMS = int64(seconds) * 1000
		} else if at, e := http.ParseTime(s); e == nil {
			f.RetryAfterMS = max(0, time.Until(at).Milliseconds())
		}
	}
	return f
}
func (c *Client) Complete(ctx context.Context, m domain.Model, prompt string, params domain.Object) (Completion, *domain.Failure) {
	body := domain.Object{}
	for k, v := range params {
		if k != "systemPrompt" {
			body[k] = v
		}
	}
	messages := []domain.Object{}
	if s, ok := params["systemPrompt"].(string); ok && s != "" {
		messages = append(messages, domain.Object{"role": "system", "content": s})
	}
	messages = append(messages, domain.Object{"role": "user", "content": prompt})
	body["model"] = m.ModelID
	body["messages"] = messages
	body["stream"] = false
	data, _, failure := c.request(ctx, "POST", c.Config.OpenRouterURL+"/chat/completions", c.Config.OpenRouterKey, body, true, false)
	if failure != nil {
		return Completion{}, failure
	}
	var response struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content json.RawMessage `json:"content"`
				Refusal *string         `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
		Usage domain.Object   `json:"usage"`
		Error json.RawMessage `json:"error"`
	}
	if domain.Decode(data, &response) != nil || len(response.Choices) != 1 || response.ID == "" || response.Model == "" {
		return Completion{}, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Completion lacks one choice, response ID or actual model"}
	}
	result := Completion{ID: response.ID, Model: response.Model, Raw: data, Usage: response.Usage}
	choice := response.Choices[0]
	if choice.Message.Refusal != nil || choice.Finish == "content_filter" {
		return result, &domain.Failure{Code: "PROVIDER_CONTENT_REJECTED", Message: "Model refused the content", ProviderCode: choice.Finish}
	}
	if choice.Finish != "stop" {
		return result, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Completion did not finish normally", ProviderCode: choice.Finish}
	}
	if json.Unmarshal(choice.Message.Content, &result.Prompt) != nil {
		var parts []struct{ Type, Text string }
		if json.Unmarshal(choice.Message.Content, &parts) == nil {
			for _, part := range parts {
				if part.Type == "text" {
					result.Prompt += part.Text
				}
			}
		}
	}
	result.Prompt = strings.TrimSpace(result.Prompt)
	if result.Prompt == "" {
		return result, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Completion produced no prompt text"}
	}
	return result, nil
}
func (c *Client) queueURLSafe(value string) bool {
	u, e := url.Parse(value)
	base, _ := url.Parse(c.Config.FalURL)
	return e == nil && u.Host == base.Host && u.Scheme == base.Scheme && u.User == nil
}
func (c *Client) Submit(ctx context.Context, m domain.Model, prompt string, params domain.Object) (Ticket, *domain.Failure) {
	body := domain.Object{}
	for k, v := range params {
		body[k] = v
	}
	body["prompt"] = prompt
	body["sync_mode"] = false
	data, _, f := c.request(ctx, "POST", c.Config.FalURL+"/"+m.ModelID, c.Config.FalKey, body, true, true)
	if f != nil {
		return Ticket{}, f
	}
	var response struct {
		ID          string `json:"request_id"`
		StatusURL   string `json:"status_url"`
		ResponseURL string `json:"response_url"`
	}
	ticket := Ticket{ID: response.ID, StatusURL: response.StatusURL, ResponseURL: response.ResponseURL, Raw: data}
	if err := domain.Decode(data, &response); err != nil {
		return Ticket{Raw: data}, &domain.Failure{Code: "PROVIDER_SUBMISSION_UNKNOWN", Message: "Queue submission response could not be decoded", Uncertain: true}
	}
	ticket = Ticket{ID: response.ID, StatusURL: response.StatusURL, ResponseURL: response.ResponseURL, Raw: data}
	if response.ID == "" || !c.queueURLSafe(response.StatusURL) || !c.queueURLSafe(response.ResponseURL) {
		return ticket, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Queue response lacks request ID or trusted status/result URLs", Uncertain: response.ID == ""}
	}
	return ticket, nil
}
func (c *Client) Status(ctx context.Context, t Ticket) (QueueState, *domain.Failure) {
	if !c.queueURLSafe(t.StatusURL) {
		return QueueState{}, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Untrusted queue status URL"}
	}
	data, headers, f := c.request(ctx, "GET", t.StatusURL, c.Config.FalKey, nil, false, true)
	if f != nil {
		return QueueState{}, f
	}
	var r struct {
		Status, Error string
		ErrorType     string `json:"error_type"`
	}
	if domain.Decode(data, &r) != nil || (r.Status != "IN_QUEUE" && r.Status != "IN_PROGRESS" && r.Status != "COMPLETED") {
		return QueueState{}, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Unknown queue status"}
	}
	state := QueueState{Status: r.Status, Raw: data}
	if r.Error != "" {
		status := 422
		switch r.ErrorType {
		case "request_timeout", "startup_timeout":
			status = 504
		case "runner_scheduling_failure", "runner_connection_timeout", "runner_disconnected", "runner_connection_refused", "runner_connection_error":
			status = 503
		case "runner_incomplete_response":
			status = 502
		case "runner_server_error", "internal_error":
			status = 500
		}
		state.Error = classify(status, data, headers)
		state.Error.ProviderCode = r.ErrorType
		state.Error.ProviderMessage = clean(r.Error)
	}
	return state, nil
}
func (c *Client) Result(ctx context.Context, t Ticket) (ImageResult, *domain.Failure) {
	if !c.queueURLSafe(t.ResponseURL) {
		return ImageResult{}, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Untrusted queue result URL"}
	}
	data, headers, f := c.request(ctx, "GET", t.ResponseURL, c.Config.FalKey, nil, false, true)
	if f != nil {
		return ImageResult{}, f
	}
	var r struct {
		Images []struct {
			URL           string
			Width, Height int
		} `json:"images"`
	}
	if domain.Decode(data, &r) != nil || len(r.Images) != 1 || r.Images[0].URL == "" || r.Images[0].Width <= 0 || r.Images[0].Height <= 0 {
		return ImageResult{}, &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Expected one image with URL and positive dimensions"}
	}
	return ImageResult{URL: r.Images[0].URL, Width: r.Images[0].Width, Height: r.Images[0].Height, Raw: data, BillableUnits: headers.Get("X-Fal-Billable-Units")}, nil
}
func (c *Client) Estimate(ctx context.Context, m domain.Model) (string, *domain.Failure) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	body := domain.Object{"estimate_type": "historical_api_price", "endpoints": domain.Object{m.ModelID: domain.Object{"call_quantity": 1}}}
	data, _, f := c.request(ctx, "POST", c.Config.FalPricingURL+"/estimate", c.Config.FalKey, body, false, true)
	if f != nil {
		return "", f
	}
	var r struct {
		Cost     json.Number `json:"total_cost"`
		Currency string      `json:"currency"`
	}
	if domain.Decode(data, &r) != nil || r.Cost == "" || r.Currency != "USD" {
		return "", &domain.Failure{Code: "INVALID_PROVIDER_RESPONSE", Message: "Missing USD price estimate"}
	}
	return r.Cost.String(), nil
}
