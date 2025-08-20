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
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGRPCInterceptors(t *testing.T) {
	ctx := context.Background()

	t.Run("authorization interceptor blocks protected methods", func(t *testing.T) {
		authzCalled := false
		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden)
			})
		}

		// Create gRPC server with authorization interceptor
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(nil, authzMiddleware)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test protected method (CreateAccount)
		_, err = client.CreateAccount(ctx, &apiv2.CreateAccountRequest{})
		require.Error(t, err)
		require.True(t, authzCalled)
		
		// Check that it's a permission denied error
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("authorization interceptor allows unprotected methods", func(t *testing.T) {
		authzCalled := false
		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden)
			})
		}

		// Create gRPC server with authorization interceptor
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(nil, authzMiddleware)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test unprotected method (Health)
		resp, err := client.Health(ctx, &apiv2.HealthRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, authzCalled, "Authorization function should not be called for unprotected methods")
	})

	t.Run("authorization interceptor with nil middleware blocks protected methods", func(t *testing.T) {
		// Create gRPC server with authorization interceptor but no authorization middleware
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(nil, nil)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test protected method (CreateAccount)
		_, err = client.CreateAccount(ctx, &apiv2.CreateAccountRequest{})
		require.Error(t, err)
		
		// Check that it's a permission denied error with specific message
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
		require.Contains(t, st.Message(), "authorization not configured")
	})

	t.Run("authentication interceptor blocks all methods when authentication fails", func(t *testing.T) {
		authnCalled := false
		authnMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authnCalled = true
				w.WriteHeader(http.StatusUnauthorized)
			})
		}

		// Create gRPC server with authentication interceptor
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(authnMiddleware, nil)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test unprotected method (Health) - should be blocked by authentication
		_, err = client.Health(ctx, &apiv2.HealthRequest{})
		require.Error(t, err)
		require.True(t, authnCalled)
		
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("both authentication and authorization interceptors work together", func(t *testing.T) {
		authnCalled := false
		authzCalled := false

		authnMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authnCalled = true
				// Allow authentication to pass
				next.ServeHTTP(w, r)
			})
		}

		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden)
			})
		}

		// Create gRPC server with both interceptors
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(authnMiddleware, authzMiddleware)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test protected method (CreateAccount) - should hit both middlewares
		_, err = client.CreateAccount(ctx, &apiv2.CreateAccountRequest{})
		require.Error(t, err)
		require.True(t, authnCalled, "Authentication middleware should be called")
		require.True(t, authzCalled, "Authorization middleware should be called")
		
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("authentication passes but authorization blocks", func(t *testing.T) {
		authnCalled := false
		authzCalled := false

		authnMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authnCalled = true
				next.ServeHTTP(w, r) // Pass authentication
			})
		}

		authzMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authzCalled = true
				w.WriteHeader(http.StatusForbidden) // Block authorization
			})
		}

		// Create gRPC server with both interceptors
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(authnMiddleware, authzMiddleware)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test unprotected method (Health) - authn should pass, authz not called
		resp, err := client.Health(ctx, &apiv2.HealthRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, authnCalled, "Authentication should be called for all methods")
		require.False(t, authzCalled, "Authorization should not be called for unprotected methods")
	})

	t.Run("uses correct HTTP method from protobuf annotations", func(t *testing.T) {
		receivedMethod := ""
		methodCheckMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				next.ServeHTTP(w, r)
			})
		}

		// Create gRPC server with method-checking middleware for both authn and authz
		server := grpc.NewServer(
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(methodCheckMiddleware, methodCheckMiddleware)),
		)
		service := NewService()
		apiv2.RegisterV2Server(server, service)

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

		// Test Health method (should use GET based on annotation)
		_, err = client.Health(ctx, &apiv2.HealthRequest{})
		require.NoError(t, err)
		require.Equal(t, http.MethodGet, receivedMethod, "Health method should use GET")

		// Reset for next test
		receivedMethod = ""

		// Test CreateAccount method (should use POST based on annotation)
		_, err = client.CreateAccount(ctx, &apiv2.CreateAccountRequest{})
		require.NoError(t, err)
		require.Equal(t, http.MethodPost, receivedMethod, "CreateAccount method should use POST")
	})
}

func TestParseMethodName(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		expected   string
	}{
		{
			name:       "standard v2 method",
			fullMethod: "/api.v2.V2/Health",
			expected:   "Health",
		},
		{
			name:       "create account method",
			fullMethod: "/api.v2.V2/CreateAccount",
			expected:   "CreateAccount",
		},
		{
			name:       "empty string",
			fullMethod: "",
			expected:   "",
		},
		{
			name:       "no slash",
			fullMethod: "Health",
			expected:   "",
		},
		{
			name:       "only slash",
			fullMethod: "/",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMethodName(tt.fullMethod)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRequiresAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		expected   bool
	}{
		{
			name:       "health method should not require authorization",
			fullMethod: "/api.v2.V2/Health",
			expected:   false,
		},
		{
			name:       "create account method should require authorization",
			fullMethod: "/api.v2.V2/CreateAccount",
			expected:   true,
		},
		{
			name:       "unknown method should not require authorization",
			fullMethod: "/api.v2.V2/UnknownMethod",
			expected:   false,
		},
		{
			name:       "invalid method path should not require authorization",
			fullMethod: "invalid",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := requiresAuthorization(tt.fullMethod)
			require.Equal(t, tt.expected, result)
		})
	}
}