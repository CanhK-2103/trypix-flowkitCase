package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Object map[string]any

func Decode(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}
func JSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

type Pricing struct {
	Currency         string    `json:"currency"`
	Unit             string    `json:"unit"`
	UnitPrice        string    `json:"unitPrice,omitempty"`
	InputPerMillion  string    `json:"inputPerMillion,omitempty"`
	OutputPerMillion string    `json:"outputPerMillion,omitempty"`
	Source           string    `json:"source"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
type Model struct {
	Alias        string   `json:"alias"`
	Provider     string   `json:"provider"`
	ModelID      string   `json:"modelId"`
	InputProfile string   `json:"inputProfile"`
	Defaults     Object   `json:"defaults"`
	Enabled      bool     `json:"enabled"`
	Pricing      *Pricing `json:"pricing,omitempty"`
}
type Node struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	ModelRef       string `json:"modelRef,omitempty"`
	Params         Object `json:"params,omitempty"`
	PromptTemplate string `json:"promptTemplate,omitempty"`
}
type Edge struct {
	From     string `json:"from"`
	FromPort string `json:"fromPort"`
	To       string `json:"to"`
	ToPort   string `json:"toPort"`
}
type Definition struct {
	SchemaVersion int    `json:"schemaVersion"`
	Nodes         []Node `json:"nodes"`
	Edges         []Edge `json:"edges"`
}
type Workflow struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Definition  Definition `json:"definition"`
	Revision    int        `json:"revision"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
type Finding struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Path      string `json:"path"`
	NodeID    string `json:"node_id,omitempty"`
	EdgeIndex *int   `json:"edge_index,omitempty"`
}
type Failure struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	ProviderCode    string `json:"provider_error_code,omitempty"`
	ProviderMessage string `json:"provider_error_message,omitempty"`
	HTTPStatus      int    `json:"provider_http_status,omitempty"`
	Retryable       bool   `json:"retryable"`
	RetryAfterMS    int64  `json:"retry_after_ms,omitempty"`
	Uncertain       bool   `json:"submission_uncertain,omitempty"`
}

func (f *Failure) Error() string { return fmt.Sprintf("%s: %s", f.Code, f.Message) }

type Cost struct {
	Estimated    *string `json:"estimated_cost_usd"`
	Reported     *string `json:"reported_cost_usd"`
	Actual       *string `json:"actual_cost_usd"`
	ActualStatus string  `json:"actual_cost_status"`
	ActualSource string  `json:"actual_cost_source,omitempty"`
	Status       string  `json:"cost_status"`
	Source       string  `json:"pricing_source,omitempty"`
}
type QuoteNode struct {
	NodeID      string   `json:"node_id"`
	ModelID     string   `json:"model_id,omitempty"`
	Cost        Cost     `json:"cost"`
	Assumptions []string `json:"assumptions"`
}
type Quote struct {
	WorkflowRevision int         `json:"workflow_revision"`
	CreatedAt        time.Time   `json:"created_at"`
	Nodes            []QuoteNode `json:"nodes"`
	Cost             Cost        `json:"cost"`
}
type Billing struct {
	Attempts    int             `json:"checks"`
	NextCheckAt *time.Time      `json:"next_check_at,omitempty"`
	CheckedAt   *time.Time      `json:"checked_at,omitempty"`
	Error       *Failure        `json:"error,omitempty"`
	Raw         json.RawMessage `json:"response,omitempty"`
}
type ProviderRequest struct {
	ID                string          `json:"id"`
	ProviderRequestID string          `json:"provider_request_id,omitempty"`
	Provider          string          `json:"provider"`
	RequestedModelID  string          `json:"requested_model_id"`
	ActualModelID     string          `json:"actual_model_id,omitempty"`
	IdentitySource    string          `json:"model_identity_source,omitempty"`
	Phase             string          `json:"phase"`
	StatusURL         string          `json:"status_url,omitempty"`
	ResponseURL       string          `json:"response_url,omitempty"`
	Raw               json.RawMessage `json:"response,omitempty"`
	Usage             Object          `json:"usage,omitempty"`
	Pricing           *Pricing        `json:"pricing_snapshot,omitempty"`
	Cost              Cost            `json:"cost"`
	Billing           Billing         `json:"billing"`
	CreatedAt         time.Time       `json:"created_at"`
	FinishedAt        *time.Time      `json:"finished_at,omitempty"`
}
type Attempt struct {
	Number             int        `json:"attempt"`
	Phase              string     `json:"phase"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	DurationMS         int64      `json:"duration_ms"`
	ProviderRequestRef string     `json:"provider_request_ref,omitempty"`
	Error              *Failure   `json:"error,omitempty"`
}
type NodeRun struct {
	Node           Node              `json:"node"`
	Model          *Model            `json:"model_snapshot,omitempty"`
	Status         string            `json:"status"`
	Input          Object            `json:"input,omitempty"`
	Output         Object            `json:"output,omitempty"`
	RenderedPrompt string            `json:"rendered_prompt,omitempty"`
	StartedAt      *time.Time        `json:"started_at,omitempty"`
	FinishedAt     *time.Time        `json:"finished_at,omitempty"`
	DeadlineAt     *time.Time        `json:"deadline_at,omitempty"`
	NextRetryAt    *time.Time        `json:"next_retry_at,omitempty"`
	DurationMS     int64             `json:"duration_ms"`
	Attempts       []Attempt         `json:"attempts"`
	Requests       []ProviderRequest `json:"provider_requests"`
	Error          *Failure          `json:"error,omitempty"`
	Cost           Cost              `json:"cost"`
}
type Run struct {
	ID              string     `json:"run_id"`
	Workflow        Workflow   `json:"workflow_snapshot"`
	Input           Object     `json:"input"`
	Status          string     `json:"status"`
	RequestID       string     `json:"request_id"`
	JobID           int64      `json:"job_id"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	QueueWaitMS     int64      `json:"queue_wait_ms"`
	DurationMS      int64      `json:"duration_ms"`
	Nodes           []NodeRun  `json:"nodes"`
	FailedNodeID    string     `json:"failed_node_id,omitempty"`
	Error           *Failure   `json:"error,omitempty"`
	Output          Object     `json:"output,omitempty"`
	Cost            Cost       `json:"cost"`
	InitialEstimate *Quote     `json:"initial_estimate,omitempty"`
}

func (r *Run) Terminal() bool { return r.Status == "succeeded" || r.Status == "failed" }

type Artifact struct {
	ID             string `json:"artifact_id"`
	RunID          string `json:"run_id"`
	NodeID         string `json:"node_id"`
	RequestID      string `json:"provider_request_ref"`
	URL            string `json:"url"`
	SourceURL      string `json:"source_url"`
	FileName       string `json:"-"`
	MIME           string `json:"content_type"`
	Size           int64  `json:"size_bytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ActualModelID  string `json:"actual_model_id"`
	IdentitySource string `json:"model_identity_source"`
}
