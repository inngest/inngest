package state

import (
	"context"
)

var (
	groupCtxVal = groupIDValType{}
)

// WithGroupID returns a context that stores the given group ID inside.
func WithGroupID(ctx context.Context, groupID string) context.Context {
	return context.WithValue(ctx, groupCtxVal, groupID)
}

// GroupIDFromContext returns the group ID given the current context, or an
// empty string if there's no group ID.
func GroupIDFromContext(ctx context.Context) string {
	str, _ := ctx.Value(groupCtxVal).(string)
	return str
}

type groupIDValType struct{}

type metadataSizeDeltaKeyType struct{}

var metadataSizeDeltaKey = metadataSizeDeltaKeyType{}

// WithMetadataSizeDelta returns a context carrying the metadata size delta
// to be persisted alongside SaveResponse. The delta represents the number
// of bytes of metadata created during the current step execution.
func WithMetadataSizeDelta(ctx context.Context, delta int) context.Context {
	return context.WithValue(ctx, metadataSizeDeltaKey, delta)
}

// MetadataSizeDeltaFromContext returns the metadata size delta from the
// context, or 0 if absent.
func MetadataSizeDeltaFromContext(ctx context.Context) int {
	v, _ := ctx.Value(metadataSizeDeltaKey).(int)
	return v
}
