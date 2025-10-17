package stephttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/oklog/ulid/v2"
)

// CheckpointRun represents the response payload for a successful run creation,
// and is used to resume any checkpointed run (async or sync).
type CheckpointRun struct {
	// FnID represents the ID of the function that the checkpoint run relates to.
	// This is required to be passed back in future step and response checkpoint calls
	// for proper tracking.
	FnID uuid.UUID `json:"fn_id"`
	// AppID represents the ID of the app that the checkpoint run relates to.
	// This is required to be passed back in future step and response checkpoint calls
	// for proper tracking.
	AppID uuid.UUID `json:"app_id"`
	// RunID is the function run ID created for this execution.
	RunID ulid.ULID `json:"run_id"`
	// Token is the token to use when redirecting if this is an async checkpoint.
	Token string `json:"token,omitempty"`

	// NOTE: The below are not included in the API response when checkpointing
	// new runs.

	// Stack is the current stack, used when resuming requests.
	Stack []string
	// Signature is the signature used when resuming requests.
	Signature string
	// Attempt is the attempt number
	Attempt int
}

// checkpointAPI handles API function runs and step checkpointing
type checkpointAPI interface {
	CheckpointNewRun(ctx context.Context, runID ulid.ULID, input NewAPIRunData, steps ...sdkrequest.GeneratorOpcode) (*CheckpointRun, error)
	CheckpointSteps(ctx context.Context, run CheckpointRun, steps []sdkrequest.GeneratorOpcode) error
	CheckpointResponse(ctx context.Context, run CheckpointRun, result APIResult) error
	GetSteps(ctx context.Context, runID ulid.ULID) (map[string]json.RawMessage, error)
}

// NewAPIRunRequest represents the entire request payload used to create new
// API-based runs.
type CheckpointNewRunRequest struct {
	// RunID represents the run ID for this request.  This is generated on the
	// client, and embeds the timestamp that the request started
	RunID ulid.ULID `json:"run_id"`

	// Idempotency allows the customization of an idempotency key, allowing us to
	// handle API idempotency using Inngest.
	Idempotency string `json:"idempotency"`

	// Event embeds the key request information which is used as the triggering
	// event for API-based runs.
	Event inngestgo.GenericEvent[NewAPIRunData] `json:"event"`

	// Steps are optional steps to send when creating the request.
	Steps []sdkrequest.GeneratorOpcode `json:"steps,omitempty,omitzero"`
}

// NewAPIRunData represents event data stored and used to create new API-based
// runs.
type NewAPIRunData struct {
	// Domain is the domain that served the incoming request.
	Domain string `json:"domain"`
	// Method is the incoming request method.  This is used for RESTful
	// API endpoints.
	Method string `json:"method"`
	// Path is the path for the incoming request.
	Path string `json:"path"` // request path
	// Fn is the optional function slug.  If not present, this is created
	// using a combination of the method and the path: "POST /v1/runs"
	Fn string `json:"fn"`

	// IP is the IP that created the request.
	IP string `json:"ip"` // incoming IP
	// ContentType is the content type for the request.
	ContentType string `json:"content_type"`
	// QueryParams are the query parameters for the request, as a single string
	// without the leading "?".
	//
	// NOTE: This is optional;  we do not require that users store the query params
	// for every request, as this may contain data that users choose not to log.
	QueryParams string `json:"query_params"`
	// Body is the incoming request body.
	//
	// NOTE: This is optional;  we do not require that users store the body for
	// every request, as this may contain data that users choose not to log.
	Body []byte `json:"body"`
}

// APIResult represents the final result of an API function call
type APIResult struct {
	// StatusCode represents the status code for the API result
	StatusCode int `json:"status_code"`
	// Headers represents any response headers sent in the server response
	Headers map[string]string `json:"headers"`
	// Body represents the API response.  This may be nil by default.  It is only
	// captured when you manually specify that you want to track the result.
	Body []byte `json:"body,omitempty"`
	// Duration represents the duration
	Duration time.Duration `json:"duration"`
	// Error represents any error from the API.  This is only for internal errors,
	// eg. when a step permanently fails
	Error string `json:"error,omitempty"`
}

// APIClient handles HTTP requests to the checkpoint API
type APIClient struct {
	baseURL     string
	primaryKey  string
	fallbackKey string
	httpClient  *http.Client
	// useFallback tracks whether to use the fallback key (1) or primary key (0)
	useFallback *atomic.Bool
}

// NewAPIClient creates a new API client with the given domain and signing keys
func NewAPIClient(baseURL, primaryKey, fallbackKey string) *APIClient {
	return &APIClient{
		baseURL:     baseURL,
		primaryKey:  primaryKey,
		fallbackKey: fallbackKey,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		useFallback: &atomic.Bool{},
	}
}

// CheckpointNewRun creates a new API run checkpoint
func (c *APIClient) CheckpointNewRun(ctx context.Context, runID ulid.ULID, input NewAPIRunData, steps ...sdkrequest.GeneratorOpcode) (*CheckpointRun, error) {
	payload := CheckpointNewRunRequest{
		RunID: runID,
		Event: inngestgo.GenericEvent[NewAPIRunData]{
			Name: "http/run.started",
			Data: input,
		},
		Steps: steps,
	}

	resp := &CheckpointRun{}
	err := c.makeRequest(ctx, "POST", "/v1/http/runs", payload, resp)
	return resp, err
}

// CheckpointSteps saves step execution state
func (c *APIClient) CheckpointSteps(ctx context.Context, run CheckpointRun, steps []sdkrequest.GeneratorOpcode) error {
	payload := map[string]any{
		"run_id": run.RunID,
		"fn_id":  run.FnID,
		"app_id": run.AppID,
		"steps":  steps,
	}
	return c.makeRequest(ctx, "POST", fmt.Sprintf("/v1/http/runs/%s/steps", run.RunID.String()), payload, nil)
}

// CheckpointResponse saves the final API response
func (c *APIClient) CheckpointResponse(ctx context.Context, run CheckpointRun, result APIResult) error {
	payload := map[string]any{
		"run_id": run.RunID,
		"fn_id":  run.FnID,
		"app_id": run.AppID,
		"result": result,
	}
	return c.makeRequest(ctx, "POST", fmt.Sprintf("/v1/http/runs/%s/response", run.RunID.String()), payload, nil)
}

func (c *APIClient) GetSteps(ctx context.Context, runID ulid.ULID) (map[string]json.RawMessage, error) {
	byt, err := c.do(ctx, "GET", fmt.Sprintf("/v0/runs/%s/actions", runID.String()), nil)
	if err != nil {
		return nil, fmt.Errorf("error making api request: %w", err)
	}
	steps := map[string]json.RawMessage{}
	err = json.Unmarshal(byt, &steps)
	return steps, err
}

// makeRequest performs an authenticated HTTP request to the API
func (c *APIClient) makeRequest(ctx context.Context, method, path string, payload any, response any) error {
	byt, err := c.do(ctx, method, path, payload)
	if err != nil {
		return fmt.Errorf("error making api request: %w", err)
	}

	if response != nil {
		// The API response always has a wrapper.
		wrapped := wrapper{Data: response}
		if err := json.Unmarshal(byt, &wrapped); err != nil {
			return fmt.Errorf("error unmarshalling response: %w", err)
		}
	}

	return nil
}

type wrapper struct {
	Data any `json:"data"`
}

// do performs an authenticated HTTP request to the API with automatic retries.
// Retries up to 5 times with exponential backoff (50ms, 100ms, 200ms, 400ms, 400ms).
// Returns the response body on success, or the last error encountered on failure.
func (c *APIClient) do(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var lastErr error

	// Retry up to 5 times with exponential backoff, capped at 400ms
	for attempt := 0; attempt < 5; attempt++ {
		byt, err := c.doSingle(ctx, method, path, payload)
		if err == nil {
			return byt, nil
		}

		lastErr = err

		// If not the last attempt, wait and retry
		if attempt < 4 {
			backoff := time.Duration(50*(1<<attempt)) * time.Millisecond
			if backoff > 400*time.Millisecond {
				backoff = 400 * time.Millisecond
			}
			time.Sleep(backoff)
		}
	}

	return nil, lastErr
}

// doSingle performs a single authenticated HTTP request to the API
func (c *APIClient) doSingle(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.getCurrentSigningKey())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// If we get a 401 and have a fallback key, try switching to it
		if resp.StatusCode == 401 && c.fallbackKey != "" && !c.useFallback.Load() {
			c.useFallback.Store(true)
			// Retry the request with the fallback key
			return c.doSingle(ctx, method, path, payload)
		}
		return byt, fmt.Errorf("API request to '%s' failed with status %d (%s)", req.URL, resp.StatusCode, byt)
	}

	return byt, err
}

// getCurrentSigningKey returns the appropriate signing key based on rotation state
func (c *APIClient) getCurrentSigningKey() string {
	if c.useFallback.Load() {
		return c.fallbackKey
	}
	return c.primaryKey
}
