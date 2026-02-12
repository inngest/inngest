package conditional

import (
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// ObservabilityType represents the type of observability being conditionally enabled.
type ObservabilityType string

const (
	// ObservabilityTypeLogs represents conditional logging.
	ObservabilityTypeLogs ObservabilityType = "logs"
	// ObservabilityTypeMetrics represents conditional metrics.
	ObservabilityTypeMetrics ObservabilityType = "metrics"
	// ObservabilityTypeTraces represents conditional tracing.
	ObservabilityTypeTraces ObservabilityType = "traces"
)

// String returns the string representation of the ObservabilityType.
func (o ObservabilityType) String() string {
	return string(o)
}

// FeatureFlagContext contains all identifiers for feature flag evaluation.
// Uses a struct to allow easy extension with new fields without breaking API.
type FeatureFlagContext struct {
	// Core identifiers (always present when available)
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	FunctionID uuid.UUID

	// Extended identifiers (optional, set via options)
	RunID       ulid.ULID // For run-specific enablement
	EventID     ulid.ULID // For event-specific enablement
	BillingPlan string    // For plan-based enablement (e.g., "enterprise", "pro")

	// Extra provides an escape hatch for future identifiers without struct changes.
	// Use this for experimental identifiers, one-off custom identifiers,
	// or future identifiers that haven't been added to the struct yet.
	Extra map[string]any
}

// GetExtra returns the value for the given key from the Extra map, or nil if not present.
func (f FeatureFlagContext) GetExtra(key string) any {
	if f.Extra == nil {
		return nil
	}
	return f.Extra[key]
}

// GetExtraString returns the string value for the given key from the Extra map,
// or an empty string if not present or not a string.
func (f FeatureFlagContext) GetExtraString(key string) string {
	v := f.GetExtra(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// HasAccountID returns true if the AccountID is set (non-zero).
func (f FeatureFlagContext) HasAccountID() bool {
	return f.AccountID != uuid.Nil
}

// HasEnvID returns true if the EnvID is set (non-zero).
func (f FeatureFlagContext) HasEnvID() bool {
	return f.EnvID != uuid.Nil
}

// HasFunctionID returns true if the FunctionID is set (non-zero).
func (f FeatureFlagContext) HasFunctionID() bool {
	return f.FunctionID != uuid.Nil
}

// HasRunID returns true if the RunID is set (non-zero).
func (f FeatureFlagContext) HasRunID() bool {
	return f.RunID.Compare(ulid.ULID{}) != 0
}

// HasEventID returns true if the EventID is set (non-zero).
func (f FeatureFlagContext) HasEventID() bool {
	return f.EventID.Compare(ulid.ULID{}) != 0
}

// Clone returns a deep copy of the FeatureFlagContext.
func (f FeatureFlagContext) Clone() FeatureFlagContext {
	clone := f
	if f.Extra != nil {
		clone.Extra = make(map[string]any, len(f.Extra))
		for k, v := range f.Extra {
			clone.Extra[k] = v
		}
	}
	return clone
}
