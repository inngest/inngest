package internal

import (
	"context"

	"github.com/inngest/inngestgo/internal/middleware"
)

type eventSenderCtxKeyType struct{}

var eventSenderCtxKey = eventSenderCtxKeyType{}

type eventSender interface {
	Send(ctx context.Context, evt any) (string, error)
	SendMany(ctx context.Context, evt []any) ([]string, error)
}

func ContextWithEventSender(ctx context.Context, sender eventSender) context.Context {
	return context.WithValue(ctx, eventSenderCtxKey, sender)
}

func EventSenderFromContext(ctx context.Context) (eventSender, bool) {
	sender, ok := ctx.Value(eventSenderCtxKey).(eventSender)
	return sender, ok
}

type middlewareManagerCtxKeyType struct{}

var middlewareManagerCtxKey = middlewareManagerCtxKeyType{}

func ContextWithMiddlewareManager(ctx context.Context, mgr *middleware.MiddlewareManager) context.Context {
	return context.WithValue(ctx, middlewareManagerCtxKey, mgr)
}

func MiddlewareManagerFromContext(ctx context.Context) (*middleware.MiddlewareManager, bool) {
	mgr, ok := ctx.Value(middlewareManagerCtxKey).(*middleware.MiddlewareManager)
	return mgr, ok
}
