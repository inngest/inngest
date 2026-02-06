package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/inngest/inngest/proto/gen/api/v2/apiv2connect"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestNewConnectRpcHandler(t *testing.T) {
	t.Run("creates handler with service", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		require.NotNil(t, handler)
		require.NotNil(t, handler.service)
		require.Same(t, service, handler.service)
	})
}

func TestConnectRpcHandler_ImplementsInterface(t *testing.T) {
	t.Run("implements V2Handler interface", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		require.Implements(t, (*apiv2connect.V2Handler)(nil), handler)
	})
}

func TestConnectRpcHandler_Health(t *testing.T) {
	t.Run("delegates to service and wraps response", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		ctx := context.Background()
		req := connect.NewRequest(&apiv2.HealthRequest{})

		resp, err := handler.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg)
		require.Equal(t, "ok", resp.Msg.Data.Status)
		require.NotNil(t, resp.Msg.Metadata)
		require.NotNil(t, resp.Msg.Metadata.FetchedAt)
	})

	t.Run("handles nil request message", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		ctx := context.Background()
		req := connect.NewRequest[apiv2.HealthRequest](nil)

		resp, err := handler.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Msg.Data.Status)
	})
}

func TestConnectRpcHandler_HTTPIntegration(t *testing.T) {
	t.Run("handler can be mounted and serve health requests", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		path, httpHandler := apiv2connect.NewV2Handler(handler)

		require.Equal(t, "/api.v2.V2/", path)
		require.NotNil(t, httpHandler)

		server := httptest.NewServer(httpHandler)
		defer server.Close()

		client := apiv2connect.NewV2Client(
			http.DefaultClient,
			server.URL,
		)

		ctx := context.Background()
		req := connect.NewRequest(&apiv2.HealthRequest{})

		resp, err := client.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Msg.Data.Status)
	})

	t.Run("handler path follows connect convention", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		path, _ := apiv2connect.NewV2Handler(handler)

		require.Equal(t, "/api.v2.V2/", path)
	})
}

func TestConnectRpcHandler_StreamRun_Validation(t *testing.T) {
	service := NewService(ServiceOptions{})
	handler := NewConnectRpcHandler(service)

	_, httpHandler := apiv2connect.NewV2Handler(handler)
	server := httptest.NewServer(httpHandler)
	defer server.Close()

	client := apiv2connect.NewV2Client(http.DefaultClient, server.URL)
	ctx := context.Background()

	t.Run("returns unimplemented when no provider configured", func(t *testing.T) {
		//
		// Service created without ConnectRPCProvider
		req := connect.NewRequest(&apiv2.StreamRunRequest{
			EnvId: "550e8400-e29b-41d4-a716-446655440000",
			RunId: "01HQZX8N6KCMV9ZQS3R2A1B2C3",
		})

		stream, err := client.StreamRun(ctx, req)
		if err != nil {
			connectErr, ok := err.(*connect.Error)
			require.True(t, ok)
			require.Equal(t, connect.CodeUnimplemented, connectErr.Code())
			return
		}

		ok := stream.Receive()
		require.False(t, ok)
		err = stream.Err()
		require.Error(t, err)
		connectErr, ok := err.(*connect.Error)
		require.True(t, ok)
		require.Equal(t, connect.CodeUnimplemented, connectErr.Code())
	})

	t.Run("returns error for invalid env_id", func(t *testing.T) {
		//
		// Need a service with provider to test validation past the unimplemented check
		mockProvider := &mockConnectRPCProvider{}
		serviceWithProvider := NewService(ServiceOptions{
			ConnectRPCProvider: mockProvider,
		})
		handlerWithProvider := NewConnectRpcHandler(serviceWithProvider)
		_, httpHandlerWithProvider := apiv2connect.NewV2Handler(handlerWithProvider)
		serverWithProvider := httptest.NewServer(httpHandlerWithProvider)
		defer serverWithProvider.Close()
		clientWithProvider := apiv2connect.NewV2Client(http.DefaultClient, serverWithProvider.URL)

		req := connect.NewRequest(&apiv2.StreamRunRequest{
			EnvId: "not-a-uuid",
			RunId: "01HQZX8N6KCMV9ZQS3R2A1B2C3",
		})

		stream, err := clientWithProvider.StreamRun(ctx, req)
		if err != nil {
			connectErr, ok := err.(*connect.Error)
			require.True(t, ok)
			require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
			require.Contains(t, err.Error(), "invalid env_id")
			return
		}

		ok := stream.Receive()
		require.False(t, ok)
		err = stream.Err()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid env_id")
	})

	t.Run("returns error for invalid run_id", func(t *testing.T) {
		mockProvider := &mockConnectRPCProvider{}
		serviceWithProvider := NewService(ServiceOptions{
			ConnectRPCProvider: mockProvider,
		})
		handlerWithProvider := NewConnectRpcHandler(serviceWithProvider)
		_, httpHandlerWithProvider := apiv2connect.NewV2Handler(handlerWithProvider)
		serverWithProvider := httptest.NewServer(httpHandlerWithProvider)
		defer serverWithProvider.Close()
		clientWithProvider := apiv2connect.NewV2Client(http.DefaultClient, serverWithProvider.URL)

		req := connect.NewRequest(&apiv2.StreamRunRequest{
			EnvId: "550e8400-e29b-41d4-a716-446655440000",
			RunId: "not-a-ulid",
		})

		stream, err := clientWithProvider.StreamRun(ctx, req)
		if err != nil {
			connectErr, ok := err.(*connect.Error)
			require.True(t, ok)
			require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
			require.Contains(t, err.Error(), "invalid run_id")
			return
		}

		ok := stream.Receive()
		require.False(t, ok)
		err = stream.Err()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid run_id")
	})

	t.Run("returns unauthenticated when no account_id in context", func(t *testing.T) {
		mockProvider := &mockConnectRPCProvider{}
		serviceWithProvider := NewService(ServiceOptions{
			ConnectRPCProvider: mockProvider,
		})
		handlerWithProvider := NewConnectRpcHandler(serviceWithProvider)
		_, httpHandlerWithProvider := apiv2connect.NewV2Handler(handlerWithProvider)
		serverWithProvider := httptest.NewServer(httpHandlerWithProvider)
		defer serverWithProvider.Close()
		clientWithProvider := apiv2connect.NewV2Client(http.DefaultClient, serverWithProvider.URL)

		req := connect.NewRequest(&apiv2.StreamRunRequest{
			EnvId: "550e8400-e29b-41d4-a716-446655440000",
			RunId: "01HQZX8N6KCMV9ZQS3R2A1B2C3",
		})

		stream, err := clientWithProvider.StreamRun(ctx, req)
		if err != nil {
			connectErr, ok := err.(*connect.Error)
			require.True(t, ok)
			require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
			return
		}

		ok := stream.Receive()
		require.False(t, ok)
		err = stream.Err()
		require.Error(t, err)
		connectErr, ok := err.(*connect.Error)
		require.True(t, ok)
		require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	})
}

func TestNewConnectRPCHTTPHandler(t *testing.T) {
	t.Run("returns correct path", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		path, httpHandler := NewConnectRPCHTTPHandler(handler)

		require.Equal(t, "/api.v2.V2/", path)
		require.NotNil(t, httpHandler)
	})

	t.Run("injects dev server context values", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		path, httpHandler := NewConnectRPCHTTPHandler(handler)

		require.Equal(t, "/api.v2.V2/", path)
		require.NotNil(t, httpHandler)

		//
		// The health endpoint should work with the injected context
		server := httptest.NewServer(httpHandler)
		defer server.Close()

		client := apiv2connect.NewV2Client(http.DefaultClient, server.URL)

		ctx := context.Background()
		req := connect.NewRequest(&apiv2.HealthRequest{})

		resp, err := client.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Msg.Data.Status)
	})
}

func TestConnectRpcHandler_ConcurrentHealth(t *testing.T) {
	t.Run("handles concurrent health requests", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		handler := NewConnectRpcHandler(service)

		_, httpHandler := apiv2connect.NewV2Handler(handler)
		server := httptest.NewServer(httpHandler)
		defer server.Close()

		client := apiv2connect.NewV2Client(http.DefaultClient, server.URL)

		const numRequests = 20
		results := make(chan error, numRequests)

		for range numRequests {
			go func() {
				ctx := context.Background()
				req := connect.NewRequest(&apiv2.HealthRequest{})
				resp, err := client.Health(ctx, req)
				if err != nil {
					results <- err
					return
				}
				if resp.Msg.Data.Status != "ok" {
					results <- err
					return
				}
				results <- nil
			}()
		}

		for range numRequests {
			err := <-results
			require.NoError(t, err)
		}
	})
}

//
// Mock provider for testing
type mockConnectRPCProvider struct{}

func (m *mockConnectRPCProvider) GetRunData(ctx context.Context, accountID uuid.UUID, envID uuid.UUID, runID ulid.ULID) (*apiv2.RunData, error) {
	return &apiv2.RunData{
		Id:     "test-run",
		Status: "RUNNING",
	}, nil
}
