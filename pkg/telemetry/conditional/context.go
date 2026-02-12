package conditional

import (
	"context"

	"github.com/google/uuid"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

func init() {
	// Register the conditional logging check with the logger package.
	// This enables logger.From() to automatically check conditional scopes.
	logger.RegisterConditionalCheck(conditionalLoggingCheck)
}

// conditionalLoggingCheck is called by logger.From() to determine if logging
// should be enabled for the current context. Returns true if logging should
// proceed, false if it should be skipped.
func conditionalLoggingCheck(ctx context.Context) bool {
	scope, ok := ScopeFromContext(ctx)
	if !ok {
		// No scope set, logging proceeds normally
		return true
	}
	// Scope is set, check if logging is enabled for this scope
	return IsLoggingEnabled(ctx, scope)
}

type contextKey struct{}

var ffContextKey = contextKey{}

// scopeContextKey is the context key for conditional scope.
type scopeContextKey struct{}

var conditionalScopeKey = scopeContextKey{}

// ContextOption is a functional option for configuring FeatureFlagContext.
type ContextOption func(*FeatureFlagContext)

// WithAccountID sets the AccountID in the FeatureFlagContext.
func WithAccountID(id uuid.UUID) ContextOption {
	return func(f *FeatureFlagContext) {
		f.AccountID = id
	}
}

// WithEnvID sets the EnvID in the FeatureFlagContext.
func WithEnvID(id uuid.UUID) ContextOption {
	return func(f *FeatureFlagContext) {
		f.EnvID = id
	}
}

// WithFunctionID sets the FunctionID in the FeatureFlagContext.
func WithFunctionID(id uuid.UUID) ContextOption {
	return func(f *FeatureFlagContext) {
		f.FunctionID = id
	}
}

// WithRunID sets the RunID in the FeatureFlagContext.
func WithRunID(id ulid.ULID) ContextOption {
	return func(f *FeatureFlagContext) {
		f.RunID = id
	}
}

// WithEventID sets the EventID in the FeatureFlagContext.
func WithEventID(id ulid.ULID) ContextOption {
	return func(f *FeatureFlagContext) {
		f.EventID = id
	}
}

// WithBillingPlan sets the BillingPlan in the FeatureFlagContext.
func WithBillingPlan(plan string) ContextOption {
	return func(f *FeatureFlagContext) {
		f.BillingPlan = plan
	}
}

// WithExtra sets a custom key-value pair in the Extra map.
func WithExtra(key string, value any) ContextOption {
	return func(f *FeatureFlagContext) {
		if f.Extra == nil {
			f.Extra = make(map[string]any)
		}
		f.Extra[key] = value
	}
}

// FromStateID returns ContextOptions to populate a FeatureFlagContext from a statev2.ID.
func FromStateID(id statev2.ID) []ContextOption {
	return []ContextOption{
		WithAccountID(id.Tenant.AccountID),
		WithEnvID(id.Tenant.EnvID),
		WithFunctionID(id.FunctionID),
		WithRunID(id.RunID),
	}
}

// FromIdentifier returns ContextOptions to populate a FeatureFlagContext from a statev1.Identifier.
func FromIdentifier(id statev1.Identifier) []ContextOption {
	opts := []ContextOption{
		WithAccountID(id.AccountID),
		WithEnvID(id.WorkspaceID),
		WithFunctionID(id.WorkflowID),
		WithRunID(id.RunID),
	}
	// Only set EventID if it's non-zero
	if id.EventID.Compare(ulid.ULID{}) != 0 {
		opts = append(opts, WithEventID(id.EventID))
	}
	return opts
}

// FromMetadata returns ContextOptions to populate a FeatureFlagContext from statev2.Metadata.
func FromMetadata(md statev2.Metadata) []ContextOption {
	opts := FromStateID(md.ID)
	eventID := md.Config.EventID()
	if eventID.Compare(ulid.ULID{}) != 0 {
		opts = append(opts, WithEventID(eventID))
	}
	return opts
}

// WithContext creates a new context with a fresh FeatureFlagContext configured with the given options.
// Use this at request/job entry points to establish the observability context.
func WithContext(ctx context.Context, opts ...ContextOption) context.Context {
	ffCtx := FeatureFlagContext{}
	for _, opt := range opts {
		opt(&ffCtx)
	}
	return context.WithValue(ctx, ffContextKey, &ffCtx)
}

// AddToContext merges additional options into an existing FeatureFlagContext in the context.
// If no FeatureFlagContext exists, it creates a new one.
// Use this when you need to add more identifiers later in the request flow.
func AddToContext(ctx context.Context, opts ...ContextOption) context.Context {
	existing := GetFromContext(ctx)
	// Clone to avoid mutating the original
	ffCtx := existing.Clone()
	for _, opt := range opts {
		opt(&ffCtx)
	}
	return context.WithValue(ctx, ffContextKey, &ffCtx)
}

// GetFromContext retrieves the FeatureFlagContext from the context.
// Returns an empty FeatureFlagContext if none is set.
func GetFromContext(ctx context.Context) FeatureFlagContext {
	v := ctx.Value(ffContextKey)
	if v == nil {
		return FeatureFlagContext{}
	}
	if ffCtx, ok := v.(*FeatureFlagContext); ok {
		return *ffCtx
	}
	return FeatureFlagContext{}
}

// HasContext returns true if a FeatureFlagContext is present in the context.
func HasContext(ctx context.Context) bool {
	return ctx.Value(ffContextKey) != nil
}

// WithScope returns a context with conditional scope set.
// When this context is passed to logger.From() or metrics functions,
// they will check if the scope is enabled before logging/recording.
//
// Usage:
//
//	ctx = conditional.WithScope(ctx, "queue.Process")
//	logger.From(ctx).Debug("conditional log")  // Only logs if enabled for scope
//
// Or for one-off calls:
//
//	logger.From(conditional.WithScope(ctx, "queue.Process")).Debug("debug info")
func WithScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, conditionalScopeKey, scope)
}

// ScopeFromContext returns the conditional scope if set.
// Returns the scope and true if a scope is set, or empty string and false otherwise.
func ScopeFromContext(ctx context.Context) (string, bool) {
	scope, ok := ctx.Value(conditionalScopeKey).(string)
	return scope, ok
}

// HasScope returns true if a conditional scope is set in context.
func HasScope(ctx context.Context) bool {
	_, ok := ScopeFromContext(ctx)
	return ok
}
