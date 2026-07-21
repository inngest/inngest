package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddlewareLogsSampledNon200Responses(t *testing.T) {
	var buf bytes.Buffer
	ctx := logger.WithStdlib(
		context.Background(),
		logger.StdlibLogger(context.Background(),
			logger.WithHandler(logger.JSONHandler),
			logger.WithLoggerLevel(logger.LevelWarning),
			logger.WithLoggerWriter(&buf),
		),
	)

	r := chi.NewRouter()
	r.Use(NewLoggingMiddleware(
		WithLoggingSamplePercent(100),
		WithLoggingAttrs(func(context.Context) []any {
			return []any{"auth_account_id", "acct-123"}
		}),
	).Middleware)
	r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/abc", nil).WithContext(ctx)
	r.ServeHTTP(httptest.NewRecorder(), req)

	log := buf.String()
	require.Contains(t, log, "http api request returned non-200")
	require.Contains(t, log, `"method":"GET"`)
	require.Contains(t, log, `"route":"/test/{id}"`)
	require.Contains(t, log, `"path":"/test/abc"`)
	require.Contains(t, log, `"status":400`)
	require.Contains(t, log, `"auth_account_id":"acct-123"`)
}

func TestLoggingMiddlewareDoesNotLog200Responses(t *testing.T) {
	var buf bytes.Buffer
	ctx := logger.WithStdlib(
		context.Background(),
		logger.StdlibLogger(context.Background(),
			logger.WithHandler(logger.JSONHandler),
			logger.WithLoggerLevel(logger.LevelWarning),
			logger.WithLoggerWriter(&buf),
		),
	)

	r := chi.NewRouter()
	r.Use(NewLoggingMiddleware(WithLoggingSamplePercent(100)).Middleware)
	r.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil).WithContext(ctx)
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.Empty(t, buf.String())
}

func TestLoggingMiddlewareSanitizesRequestFields(t *testing.T) {
	var buf bytes.Buffer
	ctx := logger.WithStdlib(
		context.Background(),
		logger.StdlibLogger(context.Background(),
			logger.WithHandler(logger.JSONHandler),
			logger.WithLoggerLevel(logger.LevelWarning),
			logger.WithLoggerWriter(&buf),
		),
	)

	h := NewLoggingMiddleware(WithLoggingSamplePercent(100)).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest("GET\n", "/bad", nil).WithContext(ctx)
	req.URL.Path = "/bad\npath"
	routeCtx := chi.NewRouteContext()
	routeCtx.RoutePatterns = []string{"/bad/{id}\n"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	h.ServeHTTP(httptest.NewRecorder(), req)

	log := buf.String()
	require.Contains(t, log, `"method":"GET"`)
	require.Contains(t, log, `"route":"/bad/{id}"`)
	require.Contains(t, log, `"path":"/badpath"`)
	require.False(t, strings.Contains(log, "GET\\n"))
	require.False(t, strings.Contains(log, "/bad\\npath"))
}
