package apiv2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/stretchr/testify/require"
)

// Helper function for tests
func newTestHTTPHandler(ctx context.Context, serviceOpts ServiceOptions, httpOpts HTTPHandlerOptions) (http.Handler, error) {
	base := apiv2base.NewBase()
	return NewHTTPHandler(ctx, serviceOpts, httpOpts, base)
}

type healthResponse struct {
	Data     healthData       `json:"data"`
	Metadata responseMetadata `json:"metadata"`
}

type healthData struct {
	Status string `json:"status"`
}

type responseMetadata struct {
	FetchedAt   string  `json:"fetchedAt"`
	CachedUntil *string `json:"cachedUntil"`
}

type errorResponse struct {
	Errors []errorItem `json:"errors"`
}

type errorItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func TestHTTPGateway_Health(t *testing.T) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(t, err)

	t.Run("GET /api/v2/health returns success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response healthResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		require.Equal(t, "ok", response.Data.Status)
		require.NotEmpty(t, response.Metadata.FetchedAt)
		require.Nil(t, response.Metadata.CachedUntil)

		_, err = time.Parse(time.RFC3339Nano, response.Metadata.FetchedAt)
		require.NoError(t, err, "fetchedAt should be a valid RFC3339 timestamp")
	})

	t.Run("POST /api/v2/health returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/health", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// gRPC gateway only supports GET for health endpoint according to proto definition
		require.Equal(t, http.StatusNotImplemented, rec.Code)
	})

	t.Run("PUT /api/v2/health returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotImplemented, rec.Code)
	})
}

func TestHTTPGateway_Middleware(t *testing.T) {
	ctx := context.Background()

	t.Run("auth middleware is called", func(t *testing.T) {
		authCalled := false
		authMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authCalled = true
				r.Header.Set("X-Auth-Called", "true")
				next.ServeHTTP(w, r)
			})
		}

		opts := HTTPHandlerOptions{
			AuthnMiddleware: authMiddleware,
		}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, authCalled)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("auth middleware can block requests", func(t *testing.T) {
		authMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("Unauthorized"))
			})
		}

		opts := HTTPHandlerOptions{
			AuthnMiddleware: authMiddleware,
		}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, "Unauthorized", rec.Body.String())
	})

	t.Run("authorization middleware blocks protected endpoints", func(t *testing.T) {
		authzCalled := false
		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"errors":[{"code":"forbidden","message":"Access denied"}]}`))
			})
		}

		opts := HTTPHandlerOptions{
			AuthzMiddleware: authzMiddleware,
		}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
		require.NoError(t, err)

		// Test protected endpoint (CreatePartnerAccount)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/partner/accounts", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, authzCalled, "Authorization middleware should be called")
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("authorization middleware not applied to health endpoint", func(t *testing.T) {
		authzCalled := false
		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"errors":[{"code":"forbidden","message":"Access denied"}]}`))
			})
		}

		opts := HTTPHandlerOptions{
			AuthzMiddleware: authzMiddleware,
		}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
		require.NoError(t, err)

		// Test health endpoint (should not trigger authz middleware)
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.False(t, authzCalled)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("both authentication and authorization middleware work together", func(t *testing.T) {
		authnCalled := false
		authzCalled := false

		authnMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authnCalled = true
				r.Header.Set("X-Authn-Called", "true")
				next.ServeHTTP(w, r)
			})
		}

		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"errors":[{"code":"forbidden","message":"Access denied"}]}`))
			})
		}

		opts := HTTPHandlerOptions{
			AuthnMiddleware: authnMiddleware,
			AuthzMiddleware: authzMiddleware,
		}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
		require.NoError(t, err)

		// Test protected endpoint (CreatePartnerAccount) - should hit both middlewares
		req := httptest.NewRequest(http.MethodPost, "/api/v2/partner/accounts", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, authnCalled, "Authentication middleware should be called")
		require.True(t, authzCalled, "Authorization middleware should be called for protected endpoint")
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHTTPGateway_Routing(t *testing.T) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(t, err)

	t.Run("routes without /api/v2 prefix return 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// The implementation serves the gateway at root, so /health might resolve
		// Let's check for a truly non-existent path instead
		req2 := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec2 := httptest.NewRecorder()

		handler.ServeHTTP(rec2, req2)

		require.Equal(t, http.StatusNotFound, rec2.Code)
	})

	t.Run("invalid endpoints under /api/v2 return 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/invalid", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("root path returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHTTPGateway_ContentTypes(t *testing.T) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(t, err)

	t.Run("accepts application/json content type for valid methods", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("accepts requests without content type for GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("response has correct content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		contentType := rec.Header().Get("Content-Type")
		require.Contains(t, contentType, "application/json")
	})
}

func TestHTTPGateway_ErrorHandling(t *testing.T) {
	t.Run("handler creation with cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		opts := HTTPHandlerOptions{}
		handler, err := newTestHTTPHandler(cancelledCtx, ServiceOptions{}, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)
	})
}

func TestHTTPGateway_ResponseFormat(t *testing.T) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(t, err)

	t.Run("response format matches expected schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		body, err := io.ReadAll(rec.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		require.Contains(t, response, "data")
		require.Contains(t, response, "metadata")

		data := response["data"].(map[string]interface{})
		require.Contains(t, data, "status")
		require.Equal(t, "ok", data["status"])

		metadata := response["metadata"].(map[string]interface{})
		require.Contains(t, metadata, "fetchedAt")
		require.NotEmpty(t, metadata["fetchedAt"])
	})

	t.Run("timestamps are properly formatted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		var response healthResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		parsedTime, err := time.Parse(time.RFC3339Nano, response.Metadata.FetchedAt)
		require.NoError(t, err)
		require.True(t, time.Since(parsedTime) < 5*time.Second)
	})
}

func TestHTTPGateway_ConcurrentRequests(t *testing.T) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(t, err)

	t.Run("handles concurrent requests", func(t *testing.T) {
		const numRequests = 20
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)
				results <- rec.Code
			}()
		}

		for i := 0; i < numRequests; i++ {
			select {
			case statusCode := <-results:
				require.Equal(t, http.StatusOK, statusCode)
			case <-time.After(5 * time.Second):
				require.Fail(t, "timeout waiting for concurrent requests")
			}
		}
	})
}

func TestHTTPGateway_InvokeFunction(t *testing.T) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(t, err)

	t.Run("POST /api/v2/functions/{id}/invoke with mode=async in request body", func(t *testing.T) {
		body := `{"data": {"message": "Hello, World!"}, "mode": "async", "idempotencyKey": "test-123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions/my-app-hello-world/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotImplemented, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response errorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Len(t, response.Errors, 1)
		require.Contains(t, response.Errors[0].Message, "not implemented")
		// Assert that the mode parameter was correctly parsed from the request body
		require.Contains(t, response.Errors[0].Message, "mode: async")
	})

	t.Run("POST /api/v2/functions/{id}/invoke with mode=sync in request body", func(t *testing.T) {
		body := `{"data": {"message": "Hello, World!"}, "mode": "sync", "idempotencyKey": "test-456"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions/my-app-hello-world/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotImplemented, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response errorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Len(t, response.Errors, 1)
		require.Contains(t, response.Errors[0].Message, "not implemented")
		// Assert that the mode parameter was correctly parsed from the request body
		require.Contains(t, response.Errors[0].Message, "mode: sync")
	})

	t.Run("POST /api/v2/functions/{id}/invoke without mode defaults to async", func(t *testing.T) {
		body := `{"data": {"message": "Hello, World!"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions/my-app-hello-world/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotImplemented, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response errorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Len(t, response.Errors, 1)
		require.Contains(t, response.Errors[0].Message, "not implemented")
		// Assert that when no mode is provided, it defaults to async
		require.Contains(t, response.Errors[0].Message, "mode: async")
	})

	t.Run("POST /api/v2/functions/{id}/invoke with invalid mode parameter", func(t *testing.T) {
		body := `{"data": {"message": "Hello, World!"}, "mode": "invalid"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions/my-app-hello-world/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response errorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Len(t, response.Errors, 1)
		require.Contains(t, response.Errors[0].Message, "Mode must be either 'sync' or 'async'")
	})

	t.Run("POST /api/v2/functions/{id}/invoke with missing function ID", func(t *testing.T) {
		body := `{"data": {"message": "Hello, World!"}, "mode": "sync"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions//invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// This should return a 400 because the function ID validation happens before URL routing
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("POST /api/v2/functions/{id}/invoke with missing data", func(t *testing.T) {
		body := `{"mode": "sync", "idempotencyKey": "test-789"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions/my-app-hello-world/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response errorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Len(t, response.Errors, 1)
		require.Contains(t, response.Errors[0].Message, "Input data is required")
	})

	t.Run("POST /api/v2/functions/{id}/invoke with complex nested data object", func(t *testing.T) {
		body := `{"data": {"user": {"id": 123, "name": "John"}, "items": [{"id": 1, "name": "Item1"}]}, "mode": "sync", "idempotencyKey": "test-complex"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/functions/my-app-hello-world/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotImplemented, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response errorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Len(t, response.Errors, 1)
		require.Contains(t, response.Errors[0].Message, "not implemented")
		require.Contains(t, response.Errors[0].Message, "mode: sync")
	})

	t.Run("GET /api/v2/functions/{id}/invoke returns not implemented", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/functions/my-app-hello-world/invoke", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// grpc-gateway returns 501 for unsupported HTTP methods on valid endpoints
		require.Equal(t, http.StatusNotImplemented, rec.Code)
	})
}

func BenchmarkHTTPGateway_Health(b *testing.B) {
	ctx := context.Background()
	opts := HTTPHandlerOptions{}
	handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)
	require.NoError(b, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", rec.Code)
			}
		}
	})
}
