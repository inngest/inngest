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
