package checkpoint

<<<<<<< HEAD
import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

type SyncCheckpoint struct {
	RunID ulid.ULID               `json:"run_id"`
	FnID  uuid.UUID               `json:"fn_id"`
	AppID uuid.UUID               `json:"app_id"`
	Steps []state.GeneratorOpcode `json:"steps"`

	// Plus auth data added from auth.  This is never exposed via JSON,
	// as this type us unmarshalled.
	AccountID uuid.UUID `json:"-"`
	EnvID     uuid.UUID `json:"-"`

	// Optional metadata.  If not provided this will be loaded.
	// This is never exposed via JSON, as this type us unmarshalled.
	Metadata *state.Metadata `json:"-"`
}

func (s SyncCheckpoint) ID() state.ID {
	return state.ID{
		RunID:      s.RunID,
		FunctionID: s.FnID,
		Tenant: state.Tenant{
			AccountID: s.AccountID,
			EnvID:     s.EnvID,
			AppID:     s.AppID,
		},
	}
}

type AsyncCheckpoint struct {
	RunID ulid.ULID               `json:"run_id"`
	FnID  uuid.UUID               `json:"fn_id"`
	Steps []state.GeneratorOpcode `json:"steps"`
	// QueueItemRef represents the queue item ID that's currently leased while
	// executing the SDK.
	QueueItemRef string `json:"qi_id"`

	// Plus auth data added from auth.  This is never exposed via JSON
	// for security.
	AccountID uuid.UUID `json:"-"`
	EnvID     uuid.UUID `json:"-"`
}

func (s AsyncCheckpoint) ID() state.ID {
	return state.ID{
		RunID:      s.RunID,
		FunctionID: s.FnID,
		Tenant: state.Tenant{
			AccountID: s.AccountID,
			EnvID:     s.EnvID,
		},
	}
}

// APIResult represents the final result of a sync-based API function call
type APIResult struct {
	// StatusCode represents the status code for the API result
	StatusCode int `json:"status_code"`
	// Headers represents any response headers sent in the server response
	Headers map[string]string `json:"headers"`
	// Body represents the API response.  This may be nil by default.  It is only
	// captured when you manually specify that you want to track the result.
	Body []byte `json:"body,omitempty"`
	// Duration represents the overall time that it took for the API to execute.
	Duration time.Duration `json:"duration"`
}

// MetricCardinality represents base IDs used within checkpoint metrics.
type MetricCardinality struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	AppID     uuid.UUID
	FnID      uuid.UUID
}

// MetricsProvider represents an interface for recording metrics
// on checkpoint APIs.
type MetricsProvider interface {
	OnFnScheduled(ctx context.Context, m MetricCardinality)
	OnStepFinished(ctx context.Context, m MetricCardinality, status enums.StepStatus)
	OnFnFinished(ctx context.Context, m MetricCardinality, status enums.RunStatus)
}

type nilCheckpointMetrics struct{}

func (nilCheckpointMetrics) OnFnScheduled(ctx context.Context, m MetricCardinality) {
}

func (nilCheckpointMetrics) OnStepFinished(ctx context.Context, m MetricCardinality, status enums.StepStatus) {
}

func (nilCheckpointMetrics) OnFnFinished(ctx context.Context, m MetricCardinality, status enums.RunStatus) {
}
