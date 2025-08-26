package apiv2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the V2 API service for gRPC with grpc-gateway
type Service struct {
	apiv2.UnimplementedV2Server
}

func NewService() *Service {
	return &Service{}
}

// GRPCServerOptions contains options for configuring the gRPC server
type GRPCServerOptions struct {
	AuthnMiddleware func(http.Handler) http.Handler
	AuthzMiddleware func(http.Handler) http.Handler
}

// NewGRPCServer creates a new gRPC server with the V2 service and optional interceptors
func NewGRPCServer(opts GRPCServerOptions) *grpc.Server {
	var serverOpts []grpc.ServerOption

	// Add authentication and authorization interceptors if any middleware is provided
	if opts.AuthnMiddleware != nil || opts.AuthzMiddleware != nil {
		serverOpts = append(serverOpts,
			grpc.UnaryInterceptor(NewAuthUnaryInterceptor(opts.AuthnMiddleware, opts.AuthzMiddleware)),
			grpc.StreamInterceptor(NewAuthStreamInterceptor(opts.AuthnMiddleware, opts.AuthzMiddleware)),
		)
	}

	server := grpc.NewServer(serverOpts...)
	service := NewService()
	apiv2.RegisterV2Server(server, service)

	return server
}

// NewGRPCServerFromHTTPOptions creates a gRPC server using HTTP middleware options
func NewGRPCServerFromHTTPOptions(httpOpts HTTPHandlerOptions) *grpc.Server {
	return NewGRPCServer(GRPCServerOptions(httpOpts))
}

type HTTPHandlerOptions struct {
	AuthnMiddleware func(http.Handler) http.Handler
	AuthzMiddleware func(http.Handler) http.Handler
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

func NewHTTPHandler(ctx context.Context, opts HTTPHandlerOptions) (http.Handler, error) {
	// Create the service
	service := NewService()

	// Create grpc-gateway mux for HTTP REST endpoints with custom error handler
	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(customErrorHandler),
	)
	if err := apiv2.RegisterV2HandlerServer(ctx, gwmux, service); err != nil {
		return nil, fmt.Errorf("failed to register v2 gateway handler: %w", err)
	}

	// Build map of paths that require authorization
	authzPaths := buildAuthzPathMap()

	r := chi.NewRouter()

	if opts.AuthnMiddleware != nil {
		r.Use(opts.AuthnMiddleware)
	}

	r.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Strip /api/v2 prefix and forward to gateway
		originalPath := req.URL.Path
		if after, ok := strings.CutPrefix(req.URL.Path, "/api/v2"); ok {
			req.URL.Path = after
		}

		// Apply authorization middleware if this path requires it
		if requiresAuthz := authzPaths[req.URL.Path]; requiresAuthz && opts.AuthzMiddleware != nil {
			authzHandler := opts.AuthzMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gwmux.ServeHTTP(w, r)
			}))
			authzHandler.ServeHTTP(w, req)
		} else {
			gwmux.ServeHTTP(w, req)
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

// Health implements the health check endpoint for gRPC (used by grpc-gateway)
func (s *Service) Health(ctx context.Context, req *apiv2.HealthRequest) (*apiv2.HealthResponse, error) {
	now := time.Now()

	return &apiv2.HealthResponse{
		Data: &apiv2.HealthData{
			Status: "ok",
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(now),
			CachedUntil: nil, // Health responses are not cached
		},
	}, nil
}

// CreatePartnerAccount implements a protected endpoint that requires authorization
func (s *Service) CreatePartnerAccount(ctx context.Context, req *apiv2.CreateAccountRequest) (*apiv2.CreateAccountResponse, error) {
	// Return multiple errors for the not implemented functionality
	return nil, NewErrors(http.StatusNotImplemented,
		ErrorItem{Code: ErrorNotImplemented, Message: "Accounts not implemented in OSS"},
		ErrorItem{Code: ErrorNotImplemented, Message: "Partners not implemented in OSS"},
	)
}

func (s *Service) CreateEnv(ctx context.Context, req *apiv2.CreateEnvRequest) (*apiv2.CreateEnvResponse, error) {
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Environments not implemented in OSS")
}

func (s *Service) FetchPartnerAccounts(ctx context.Context, req *apiv2.FetchAccountsRequest) (*apiv2.FetchAccountsResponse, error) {
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Accounts not implemented in OSS")
}

func (s *Service) FetchAccount(ctx context.Context, req *apiv2.FetchAccountRequest) (*apiv2.FetchAccountResponse, error) {
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Accounts not implemented in OSS")
}

func (s *Service) FetchAccountEventKeys(ctx context.Context, req *apiv2.FetchAccountEventKeysRequest) (*apiv2.FetchAccountEventKeysResponse, error) {
	// Validate required fields
	if req.AccountId == "" {
		return nil, NewError(http.StatusBadRequest, ErrorMissingField, "Account ID is required")
	}

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// For now, return not implemented since this is OSS
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Account event keys not implemented in OSS")
}

func (s *Service) FetchAccountEnvs(ctx context.Context, req *apiv2.FetchAccountEnvsRequest) (*apiv2.FetchAccountEnvsResponse, error) {
	// Validate required fields
	if req.AccountId == "" {
		return nil, NewError(http.StatusBadRequest, ErrorMissingField, "Account ID is required")
	}

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 250 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 250")
		}
	}

	// For now, return not implemented since this is OSS
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Account environments not implemented in OSS")
}
