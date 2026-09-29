package queue

import (
	"context"
	"time"
)

var (
	startedAtKey                           = startedAtCtxKey{}
	sojournKey                             = sojournCtxKey{}
	latencyKey                             = latencyCtxKey{}
	archivedWorkspaceAppSemaphoreBypassKey = archivedWorkspaceAppSemaphoreBypassCtxKey{}
)

// startedAtCtxKey is a context key which records when the queue item starts,
// available via context.
type startedAtCtxKey struct{}

// latencyCtxKey is a context key which records when the queue item starts,
// available via context.
type latencyCtxKey struct{}

// sojournCtxKey is a context key which records when the queue item starts,
// available via context.
type sojournCtxKey struct{}

type archivedWorkspaceAppSemaphoreBypassCtxKey struct{}

func GetItemStart(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(startedAtKey).(time.Time)
	return t, ok
}

func GetItemSystemLatency(ctx context.Context) (time.Duration, bool) {
	t, ok := ctx.Value(latencyKey).(time.Duration)
	return t, ok
}

func GetItemConcurrencyLatency(ctx context.Context) (time.Duration, bool) {
	t, ok := ctx.Value(sojournKey).(time.Duration)
	return t, ok
}

// WithArchivedWorkspaceAppSemaphoreBypass marks an item that was leased without
// its app semaphore so consumers can restrict it to cleanup-only processing.
func WithArchivedWorkspaceAppSemaphoreBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, archivedWorkspaceAppSemaphoreBypassKey, true)
}

// ArchivedWorkspaceAppSemaphoreBypassed reports whether the current item was
// leased without its app semaphore for archived-workspace cleanup.
func ArchivedWorkspaceAppSemaphoreBypassed(ctx context.Context) bool {
	bypassed, _ := ctx.Value(archivedWorkspaceAppSemaphoreBypassKey).(bool)
	return bypassed
}
