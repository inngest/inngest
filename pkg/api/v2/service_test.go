package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestService_Health(t *testing.T) {
	service := NewService()

	t.Run("returns health status with timestamp", func(t *testing.T) {
		ctx := context.Background()
		req := &apiv2.HealthRequest{}

		before := time.Now()
		resp, err := service.Health(ctx, req)
		after := time.Now()

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, "ok", resp.Data.Status)
		require.NotNil(t, resp.Metadata)
		require.NotNil(t, resp.Metadata.FetchedAt)
		require.Nil(t, resp.Metadata.CachedUntil)

		fetchedTime := resp.Metadata.FetchedAt.AsTime()
		require.True(t, fetchedTime.After(before) || fetchedTime.Equal(before))
		require.True(t, fetchedTime.Before(after) || fetchedTime.Equal(after))
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		req := &apiv2.HealthRequest{}
		resp, err := service.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Data.Status)
	})
}

func TestNewService(t *testing.T) {
	t.Run("creates new service instance", func(t *testing.T) {
		service := NewService()
		require.NotNil(t, service)
		require.IsType(t, &Service{}, service)
	})
}

func TestNewHTTPHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("creates HTTP handler without auth middleware", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := NewHTTPHandler(ctx, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("creates HTTP handler with auth middleware", func(t *testing.T) {
		authMiddlewareCalled := false
		authMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authMiddlewareCalled = true
				next.ServeHTTP(w, r)
			})
		}

		opts := HTTPHandlerOptions{
			AuthnMiddleware: authMiddleware,
		}
		handler, err := NewHTTPHandler(ctx, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, authMiddlewareCalled)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("handles health endpoint correctly", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := NewHTTPHandler(ctx, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		require.Contains(t, rec.Body.String(), `"status":"ok"`)
		require.Contains(t, rec.Body.String(), `"fetchedAt"`)
	})

	t.Run("strips /api/v2 prefix correctly", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := NewHTTPHandler(ctx, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestService_HealthResponse_Structure(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	req := &apiv2.HealthRequest{}

	resp, err := service.Health(ctx, req)
	require.NoError(t, err)

	t.Run("validates response structure", func(t *testing.T) {
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Metadata)
	})

	t.Run("validates data fields", func(t *testing.T) {
		require.Equal(t, "ok", resp.Data.Status)
	})

	t.Run("validates metadata fields", func(t *testing.T) {
		require.NotNil(t, resp.Metadata.FetchedAt)
		require.Nil(t, resp.Metadata.CachedUntil)

		fetchedAt := resp.Metadata.FetchedAt.AsTime()
		require.False(t, fetchedAt.IsZero())
		require.True(t, fetchedAt.Before(time.Now().Add(time.Second)))
	})
}

func TestService_HealthRequest_Validation(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	t.Run("accepts nil request", func(t *testing.T) {
		resp, err := service.Health(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Data.Status)
	})

	t.Run("accepts empty request", func(t *testing.T) {
		req := &apiv2.HealthRequest{}
		resp, err := service.Health(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Data.Status)
	})
}

func TestService_Metadata_Timestamp(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	req := &apiv2.HealthRequest{}

	t.Run("timestamps are consistent and recent", func(t *testing.T) {
		start := time.Now()

		resp1, err := service.Health(ctx, req)
		require.NoError(t, err)

		resp2, err := service.Health(ctx, req)
		require.NoError(t, err)

		end := time.Now()

		time1 := resp1.Metadata.FetchedAt.AsTime()
		time2 := resp2.Metadata.FetchedAt.AsTime()

		require.True(t, time1.After(start) || time1.Equal(start))
		require.True(t, time1.Before(end) || time1.Equal(end))
		require.True(t, time2.After(start) || time2.Equal(start))
		require.True(t, time2.Before(end) || time2.Equal(end))
		require.True(t, time2.After(time1) || time2.Equal(time1))
	})

	t.Run("timestamp format is valid protobuf timestamp", func(t *testing.T) {
		resp, err := service.Health(ctx, req)
		require.NoError(t, err)

		timestamp := resp.Metadata.FetchedAt
		require.NotNil(t, timestamp)

		require.True(t, timestamp.IsValid())

		asTime := timestamp.AsTime()
		require.False(t, asTime.IsZero())

		fromTime := timestamppb.New(asTime)
		require.Equal(t, timestamp.Seconds, fromTime.Seconds)
		require.Equal(t, timestamp.Nanos, fromTime.Nanos)
	})
}
