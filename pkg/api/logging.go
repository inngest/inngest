package api

import (
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

type loggingMiddleware struct{}

func NewLoggingMiddleware() loggingMiddleware {
	return loggingMiddleware{}
}

func (loggingMiddleware) Middleware(next http.Handler) http.Handler {
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
		if rand.Intn(100) >= requestLogSamplePercent {
			return
		}

		logger.StdlibLogger(r.Context()).Warn(
			"http api request returned non-200 (sampled)",
			"method", util.SanitizeLogField(r.Method),
			"route", util.SanitizeLogField(chi.RouteContext(r.Context()).RoutePattern()),
			"path", util.SanitizeLogField(r.URL.Path),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes_written", ww.BytesWritten(),
		)
	})
}
