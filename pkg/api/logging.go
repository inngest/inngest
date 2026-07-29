package api

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

// requestLogSamplePercent is the percentage of non-200 HTTP responses logged.
const requestLogSamplePercent = 5

type LoggingMiddleware interface {
	Middleware(next http.Handler) http.Handler
}

type LoggingAttrsFunc func(context.Context) []any

type LoggingMiddlewareOpt func(*loggingMiddleware)

type loggingMiddleware struct {
	attrs         LoggingAttrsFunc
	samplePercent int
}

func NewLoggingMiddleware(opts ...LoggingMiddlewareOpt) loggingMiddleware {
	m := loggingMiddleware{samplePercent: requestLogSamplePercent}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func WithLoggingAttrs(fn LoggingAttrsFunc) LoggingMiddlewareOpt {
	return func(m *loggingMiddleware) {
		m.attrs = fn
	}
}

func WithLoggingSamplePercent(percent int) LoggingMiddlewareOpt {
	return func(m *loggingMiddleware) {
		m.samplePercent = percent
	}
}

func (m loggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			// The handler wrote a body without calling WriteHeader, which
			// implies a 200.
			status = http.StatusOK
		}
		if status == http.StatusOK {
			return
		}
		if m.samplePercent <= 0 || rand.Intn(100) >= m.samplePercent {
			return
		}

		args := []any{
			"method", util.SanitizeLogField(r.Method),
			"route", util.SanitizeLogField(chi.RouteContext(r.Context()).RoutePattern()),
			"path", util.SanitizeLogField(r.URL.Path),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes_written", ww.BytesWritten(),
		}
		if m.attrs != nil {
			args = append(args, m.attrs(r.Context())...)
		}

		logger.StdlibLogger(r.Context()).Warn("http api request returned non-200 (sampled)", args...)
	})
}
