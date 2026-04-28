package ociauth

import (
	"context"
)

type scopeKey struct{}

// ContextWithScope returns ctx annotated with the given
// scope. When the ociauth transport receives a request with a scope in the context,
// it will treat it as "desired authorization scope"; new authorization tokens
// will be acquired with that scope as well as any scope required by
// the operation.
func ContextWithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

// ScopeFromContext returns any scope associated with the context
// by [ContextWithScope].
func ScopeFromContext(ctx context.Context) Scope {
	s, _ := ctx.Value(scopeKey{}).(Scope)
	return s
}

type requestInfoKey struct{}

// RequestInfo provides information about the OCI request that
// is currently being made. It is expected to be attached to an HTTP
// request context. The [ociclient] package will add this to all
// requests that is makes.
type RequestInfo struct {
	// RequiredScope holds the authorization scope that's required
	// by the request. The ociauth logic will reuse any available
	// auth token that has this scope. When acquiring a new token,
	// it will add any scope found in [ScopeFromContext] too.
	RequiredScope Scope
}

// ContextWithRequestInfo returns ctx annotated with the given
// request informaton. When ociclient receives a request with
// this attached, it will respect info.RequiredScope to determine
// what auth tokens to reuse. When it acquires a new token,
// it will ask for the union of info.RequiredScope [ScopeFromContext].
func ContextWithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

// RequestInfoFromContext returns any request information associated with the context
// by [ContextWithRequestInfo].
func RequestInfoFromContext(ctx context.Context) RequestInfo {
	info, _ := ctx.Value(requestInfoKey{}).(RequestInfo)
	return info
}
