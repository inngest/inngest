package conditional

import (
	"context"
	"sync"
)

// FeatureFlagFn is the function signature for feature flag evaluation.
// It receives the context, all identifiers in FeatureFlagContext, the observability type,
// and the scope, and returns whether the observability should be enabled.
type FeatureFlagFn func(
	ctx context.Context,
	ffCtx FeatureFlagContext,
	observabilityType ObservabilityType,
	scope string,
) bool

var (
	globalFeatureFlagFn FeatureFlagFn
	globalMu            sync.RWMutex
)

// RegisterFeatureFlag registers a global feature flag function for conditional observability.
// This should be called once at application startup.
// The function is thread-safe and can be called multiple times to replace the existing function.
func RegisterFeatureFlag(fn FeatureFlagFn) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalFeatureFlagFn = fn
}

// GetFeatureFlagFn returns the currently registered feature flag function.
// Returns nil if no function has been registered.
func GetFeatureFlagFn() FeatureFlagFn {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalFeatureFlagFn
}

// IsEnabled checks if observability is enabled for the given parameters.
// If no feature flag function is registered, returns false.
func IsEnabled(
	ctx context.Context,
	ffCtx FeatureFlagContext,
	observabilityType ObservabilityType,
	scope string,
) bool {
	globalMu.RLock()
	fn := globalFeatureFlagFn
	globalMu.RUnlock()

	if fn == nil {
		return false
	}
	return fn(ctx, ffCtx, observabilityType, scope)
}

// IsEnabledFromContext checks if observability is enabled using the FeatureFlagContext
// stored in the context.Context. If no FeatureFlagContext is present, returns false.
func IsEnabledFromContext(
	ctx context.Context,
	observabilityType ObservabilityType,
	scope string,
) bool {
	ffCtx := GetFromContext(ctx)
	return IsEnabled(ctx, ffCtx, observabilityType, scope)
}

// IsLoggingEnabled is a convenience function that checks if logging is enabled
// for the given scope using the FeatureFlagContext from the context.
func IsLoggingEnabled(ctx context.Context, scope string) bool {
	return IsEnabledFromContext(ctx, ObservabilityTypeLogs, scope)
}

// IsMetricsEnabled is a convenience function that checks if metrics are enabled
// for the given scope using the FeatureFlagContext from the context.
func IsMetricsEnabled(ctx context.Context, scope string) bool {
	return IsEnabledFromContext(ctx, ObservabilityTypeMetrics, scope)
}

// IsTracingEnabled is a convenience function that checks if tracing is enabled
// for the given scope using the FeatureFlagContext from the context.
func IsTracingEnabled(ctx context.Context, scope string) bool {
	return IsEnabledFromContext(ctx, ObservabilityTypeTraces, scope)
}

// AlwaysEnabled is a FeatureFlagFn that always returns true.
// Useful for testing or when you want to enable all conditional observability.
func AlwaysEnabled(_ context.Context, _ FeatureFlagContext, _ ObservabilityType, _ string) bool {
	return true
}

// NeverEnabled is a FeatureFlagFn that always returns false.
// Useful for testing or when you want to disable all conditional observability.
func NeverEnabled(_ context.Context, _ FeatureFlagContext, _ ObservabilityType, _ string) bool {
	return false
}

// ScopeEnabled returns a FeatureFlagFn that enables observability only for specific scopes.
func ScopeEnabled(scopes ...string) FeatureFlagFn {
	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}
	return func(_ context.Context, _ FeatureFlagContext, _ ObservabilityType, scope string) bool {
		return scopeSet[scope]
	}
}

// TypeEnabled returns a FeatureFlagFn that enables observability only for specific types.
func TypeEnabled(types ...ObservabilityType) FeatureFlagFn {
	typeSet := make(map[ObservabilityType]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}
	return func(_ context.Context, _ FeatureFlagContext, obsType ObservabilityType, _ string) bool {
		return typeSet[obsType]
	}
}

// ClearFeatureFlag removes the registered feature flag function.
// Primarily useful for testing.
func ClearFeatureFlag() {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalFeatureFlagFn = nil
}
