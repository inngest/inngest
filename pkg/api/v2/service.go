package apiv2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServiceOptions defines the complete interface that mirrors router.Opts.
// This interface uses empty interfaces to be fully portable with zero external dependencies.
// All methods return 'any' type allowing external applications to provide their own implementations.
type ServiceOptions interface {
	GetActionConfig() any
	GetAuthn() any
	GetBigQuery() any
	GetCancellationReadWriter() any
	GetClerk() any
	GetConditionalConnectTracer() any
	GetConnectGatewayRetriever() any
	GetConnectHistoryRPC() any
	GetConnectRequestAuther() any
	GetConnectRequestStateManager() any
	GetConnectTokenSigner() any
	GetDB() any
	GetDBCache() any
	GetEncryptor() any
	GetEntitlementProvider() any
	GetEntitlements() any
	GetEventReader() any
	GetExecutor() any
	GetFDB() any
	GetFnReader() any
	GetHistoryReader() any
	GetLog() any
	GetMetrics() any
	GetMetricsRPC() any
	GetPublisher() any
	GetQueueShards() any
	GetReadiness() any
	GetReadOnlyDB() any
	GetRedis() any
	GetReplayStore() any
	GetUnshardedClient() any
}

// Service implements the V2 API service for gRPC with grpc-gateway
type Service struct {
	apiv2.UnimplementedV2Server
	opts ServiceOptions
}

func NewService(opts ServiceOptions) *Service {
	return &Service{opts: opts}
}

// GRPCServerOptions contains options for configuring the gRPC server
type GRPCServerOptions struct {
	AuthnMiddleware func(http.Handler) http.Handler
	AuthzMiddleware func(http.Handler) http.Handler
	Opts            ServiceOptions
}

// NewGRPCServer creates a new gRPC server with the V2 service and optional interceptors
func NewGRPCServer(serviceOpts ServiceOptions, grpcOpts GRPCServerOptions) *grpc.Server {
	var serverOpts []grpc.ServerOption

	// Add authentication and authorization interceptors if any middleware is provided
	if grpcOpts.AuthnMiddleware != nil || grpcOpts.AuthzMiddleware != nil {
		serverOpts = append(serverOpts,
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(grpcOpts.AuthnMiddleware, grpcOpts.AuthzMiddleware)),
			grpc.StreamInterceptor(NewAuthStreamInterceptor(grpcOpts.AuthnMiddleware, grpcOpts.AuthzMiddleware)),
		)
	}

	server := grpc.NewServer(serverOpts...)
	service := NewService(serviceOpts)
	apiv2.RegisterV2Server(server, service)

	return server
}

// NewGRPCServerFromHTTPOptions creates a gRPC server using HTTP middleware options
func NewGRPCServerFromHTTPOptions(serviceOpts ServiceOptions, httpOpts HTTPHandlerOptions) *grpc.Server {
	grpcOpts := GRPCServerOptions{
		AuthnMiddleware: httpOpts.AuthnMiddleware,
		AuthzMiddleware: httpOpts.AuthzMiddleware,
		Opts:            serviceOpts,
	}
	return NewGRPCServer(serviceOpts, grpcOpts)
}

type HTTPHandlerOptions struct {
	AuthnMiddleware func(http.Handler) http.Handler
	AuthzMiddleware func(http.Handler) http.Handler
	MountPoint      string // Mount point: "/api/v2" (default) or "/v2"
}

// customErrorHandler converts gRPC errors to our API error format
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	const fallback = `{"errors":[{"code":"internal_server_error","message":"An unexpected error occurred"}]}`

	w.Header().Set("Content-Type", "application/json")

	// Extract gRPC status from error
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, return 500
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fallback))
		return
	}

	// Map gRPC codes to HTTP status codes
	httpCode := grpcToHTTPStatus(st.Code())

	// Try to parse the error message as our error format
	message := st.Message()

	// If the message looks like our JSON format, use it directly
	if strings.HasPrefix(message, `{"errors":`) {
		w.WriteHeader(httpCode)
		_, _ = w.Write([]byte(message))
		return
	}

	// Otherwise, create a single error response
	errorResponse := ErrorResponse{
		Errors: []ErrorItem{
			{
				Code:    "api_error", // Generic code for non-structured errors
				Message: message,
			},
		},
	}

	jsonData, jsonErr := json.Marshal(errorResponse)
	if jsonErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fallback))
		return
	}

	w.WriteHeader(httpCode)
	_, _ = w.Write(jsonData)
}

// grpcToHTTPStatus maps gRPC codes back to HTTP status codes
func grpcToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.Internal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func NewHTTPHandler(ctx context.Context, serviceOpts ServiceOptions, httpOpts HTTPHandlerOptions) (http.Handler, error) {
	// Create the service
	service := NewService(serviceOpts)

	// Create grpc-gateway mux for HTTP REST endpoints with custom error handler
	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(customErrorHandler),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			// forward standard headers
			if strings.HasPrefix(strings.ToLower(key), "x-") || key == "authorization" {
				return strings.ToLower(key), true
			}
			return "", false
		}),
	)
	if err := apiv2.RegisterV2HandlerServer(ctx, gwmux, service); err != nil {
		return nil, fmt.Errorf("failed to register v2 gateway handler: %w", err)
	}

	// Build map of paths that require authorization
	authzPaths := buildAuthzPathMap()

	r := chi.NewRouter()

	// Add authentication middleware first
	if httpOpts.AuthnMiddleware != nil {
		r.Use(httpOpts.AuthnMiddleware)
	}

	r.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Strip mount point prefix and forward to gateway
		mountPoint := httpOpts.MountPoint
		if mountPoint == "" {
			mountPoint = "/api/v2" // Default to current behavior
		}
		
		originalPath := req.URL.Path
		if after, ok := strings.CutPrefix(req.URL.Path, mountPoint); ok {
			req.URL.Path = after
		}

		// Apply authorization middleware if this path requires it
		if requiresAuthz := authzPaths[req.URL.Path]; requiresAuthz && httpOpts.AuthzMiddleware != nil {
			authzHandler := httpOpts.AuthzMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Add JSON validation after authorization for protected paths
				validationHandler := JSONTypeValidationMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gwmux.ServeHTTP(w, r)
				}))
				validationHandler.ServeHTTP(w, r)
			}))
			authzHandler.ServeHTTP(w, req)
		} else {
			// Add JSON validation for unprotected paths
			validationHandler := JSONTypeValidationMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gwmux.ServeHTTP(w, r)
			}))
			validationHandler.ServeHTTP(w, req)
		}

		// Restore original path for logging
		req.URL.Path = originalPath
	}))

	return r, nil
}

// buildAuthzPathMap inspects protobuf annotations to determine which paths require authorization
func buildAuthzPathMap() map[string]bool {
	authzPaths := make(map[string]bool)

	// Get the service descriptor
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		return authzPaths
	}

	// Iterate through all methods in the service
	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)

		// Check if method has authz annotation
		if hasAuthzAnnotation(method) {
			// Get the HTTP path from google.api.http annotation
			if path := getHTTPPath(method); path != "" {
				authzPaths[path] = true
			}
		}
	}

	return authzPaths
}
