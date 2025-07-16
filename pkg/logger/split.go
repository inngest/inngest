package logger

import (
	"context"
	"log/slog"
)

type splitHandler struct {
	handlers []slog.Handler
}

func (s splitHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, handler := range s.handlers {
		if !handler.Enabled(ctx, l) {
			return false
		}
	}
	return true
}

func (s splitHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range s.handlers {
		// might not need to clone but let's be safe
		err := handler.Handle(ctx, record.Clone())
		if err != nil {
			return err
		}
	}

	return nil
}

func (s splitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	for i, handler := range s.handlers {
		s.handlers[i] = handler.WithAttrs(attrs)
	}

	return s
}

func (s splitHandler) WithGroup(name string) slog.Handler {
	for i, handler := range s.handlers {
		s.handlers[i] = handler.WithGroup(name)
	}

	return s
}

func NewSplitHandler(handler ...slog.Handler) slog.Handler {
	return &splitHandler{handlers: handler}
}
