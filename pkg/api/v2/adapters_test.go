package apiv2

import (
	"context"
	"net"
	"net/http"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestHTTPMiddlewareToGRPCInterceptor(t *testing.T) {
	ctx := context.Background()

	t.Run("allows request when middleware allows", func(t *testing.T) {
		allowMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if request has required header
				if r.Header.Get("authorization") == "Bearer valid-token" {
					next.ServeHTTP(w, r)
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}
			})
		}

		authzFunc := HTTPMiddlewareToGRPCInterceptor(allowMiddleware)

		// Create context with valid authorization header
		md := metadata.Pairs("authorization", "Bearer valid-token")
		ctx := metadata.NewIncomingContext(ctx, md)

		err := authzFunc(ctx, "/api.v2.V2/CreatePartnerAccount")
		require.NoError(t, err)
	})

	t.Run("blocks request when middleware blocks", func(t *testing.T) {
		blockMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			})
		}

		authzFunc := HTTPMiddlewareToGRPCInterceptor(blockMiddleware)

		err := authzFunc(ctx, "/api.v2.V2/CreatePartnerAccount")
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("checks authorization header from gRPC metadata", func(t *testing.T) {
		headerCheckMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("authorization") == "Bearer invalid-token" {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					next.ServeHTTP(w, r)
				}
			})
		}

		authzFunc := HTTPMiddlewareToGRPCInterceptor(headerCheckMiddleware)

		// Create context with invalid authorization header
		md := metadata.Pairs("authorization", "Bearer invalid-token")
		ctx := metadata.NewIncomingContext(ctx, md)

		err := authzFunc(ctx, "/api.v2.V2/CreatePartnerAccount")
		require.Error(t, err)
	})
}

func TestNewGRPCServerFromHTTPOptions(t *testing.T) {
	ctx := context.Background()

	t.Run("uses HTTP middleware for gRPC authorization", func(t *testing.T) {
		blockMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			})
		}

		httpOpts := HTTPHandlerOptions{
			AuthzMiddleware: blockMiddleware,
		}

		server := NewGRPCServerFromHTTPOptions(ServiceOptions{}, httpOpts)

		// Setup in-memory connection
		lis := bufconn.Listen(1024 * 1024)
		go func() {
			_ = server.Serve(lis)
		}()
		defer server.Stop()

		// Create client
		conn, err := grpc.NewClient("passthrough://bufnet",
			grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
				return lis.Dial()
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := apiv2.NewV2Client(conn)

		// Test protected method (CreatePartnerAccount) - should be blocked
		_, err = client.CreatePartnerAccount(ctx, &apiv2.CreateAccountRequest{})
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())

		// Test unprotected method (Health) - should work
		resp, err := client.Health(ctx, &apiv2.HealthRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("works without middleware", func(t *testing.T) {
		httpOpts := HTTPHandlerOptions{}

		server := NewGRPCServerFromHTTPOptions(ServiceOptions{}, httpOpts)

		// Setup in-memory connection
		lis := bufconn.Listen(1024 * 1024)
		go func() {
			_ = server.Serve(lis)
		}()
		defer server.Stop()

		// Create client
		conn, err := grpc.NewClient("passthrough://bufnet",
			grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
				return lis.Dial()
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := apiv2.NewV2Client(conn)

		// Both methods should work without middleware
		resp1, err := client.Health(ctx, &apiv2.HealthRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp1)

		resp2, err := client.CreatePartnerAccount(ctx, &apiv2.CreateAccountRequest{})
		require.Error(t, err)
		require.Nil(t, resp2)
	})
}

func TestGRPCServerOptions(t *testing.T) {
	t.Run("creates server with middleware", func(t *testing.T) {
		httpMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}

		opts := GRPCServerOptions{
			AuthnMiddleware: httpMiddleware,
			AuthzMiddleware: httpMiddleware,
		}

		server := NewGRPCServer(ServiceOptions{}, opts)
		require.NotNil(t, server)
	})
}
