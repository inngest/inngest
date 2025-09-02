package apiv2

import (
	"context"
	"net"
	"testing"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupGRPCTestServer(t testing.TB) (apiv2.V2Client, func()) {
	lis := bufconn.Listen(bufSize)

	server := grpc.NewServer()
	service := NewService(NewServiceOptions(ServiceConfig{}))
	apiv2.RegisterV2Server(server, service)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("gRPC server exited with error: %v", err)
		}
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := apiv2.NewV2Client(conn)

	cleanup := func() {
		conn.Close()
		server.Stop()
		lis.Close()
	}

	return client, cleanup
}

func TestGRPCIntegration_Health(t *testing.T) {
	client, cleanup := setupGRPCTestServer(t)
	defer cleanup()

	t.Run("health check via gRPC", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &apiv2.HealthRequest{}
		resp, err := client.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, "ok", resp.Data.Status)
		require.NotNil(t, resp.Metadata)
		require.NotNil(t, resp.Metadata.FetchedAt)
		require.Nil(t, resp.Metadata.CachedUntil)
	})

	t.Run("multiple concurrent health checks", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		const numRequests = 10
		results := make(chan error, numRequests)

		for range numRequests {
			go func() {
				req := &apiv2.HealthRequest{}
				resp, err := client.Health(ctx, req)
				if err != nil {
					results <- err
					return
				}

				if resp.Data.Status != "ok" {
					results <- err
					return
				}

				results <- nil
			}()
		}

		for range numRequests {
			select {
			case err := <-results:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				require.Fail(t, "timeout waiting for concurrent health checks")
			}
		}
	})

	t.Run("health check with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		req := &apiv2.HealthRequest{}
		resp, err := client.Health(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "context canceled")
	})

	t.Run("health check with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond)

		req := &apiv2.HealthRequest{}
		resp, err := client.Health(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "context deadline exceeded")
	})
}

func TestGRPCIntegration_ServiceImplementation(t *testing.T) {
	t.Run("service implements V2Server interface", func(t *testing.T) {
		service := NewService(NewServiceOptions(ServiceConfig{}))
		require.Implements(t, (*apiv2.V2Server)(nil), service)
	})

	t.Run("service registers successfully with gRPC server", func(t *testing.T) {
		server := grpc.NewServer()
		service := NewService(NewServiceOptions(ServiceConfig{}))

		require.NotPanics(t, func() {
			apiv2.RegisterV2Server(server, service)
		})

		server.Stop()
	})

	t.Run("verify gRPC method descriptors", func(t *testing.T) {
		client, cleanup := setupGRPCTestServer(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req := &apiv2.HealthRequest{}
		resp, err := client.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)

		require.IsType(t, &apiv2.HealthResponse{}, resp)
		require.IsType(t, &apiv2.HealthData{}, resp.Data)
		require.IsType(t, &apiv2.ResponseMetadata{}, resp.Metadata)
	})
}

func TestGRPCIntegration_ErrorHandling(t *testing.T) {
	t.Run("handles server shutdown gracefully", func(t *testing.T) {
		clientLocal, cleanupLocal := setupGRPCTestServer(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		cleanupLocal()
		time.Sleep(100 * time.Millisecond)

		req := &apiv2.HealthRequest{}
		resp, err := clientLocal.Health(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
	})
}

func BenchmarkGRPCIntegration_Health(b *testing.B) {
	client, cleanup := setupGRPCTestServer(b)
	defer cleanup()

	ctx := context.Background()
	req := &apiv2.HealthRequest{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Health(ctx, req)
			if err != nil {
				b.Fatalf("Health check failed: %v", err)
			}
			if resp.Data.Status != "ok" {
				b.Fatalf("Expected status 'ok', got '%s'", resp.Data.Status)
			}
		}
	})
}
