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
	"github.com/inngest/inngest/pkg/consts"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the V2 API service for gRPC with grpc-gateway
type Service struct {
	apiv2.UnimplementedV2Server
	signingKeys SigningKeysProvider
}

// ServiceOptions contains configuration for the V2 service
type ServiceOptions struct {
	SigningKeysProvider SigningKeysProvider
}

func NewService(opts ServiceOptions) *Service {
	return &Service{
		signingKeys: opts.SigningKeysProvider,
	}
}

// GRPCServerOptions contains options for configuring the gRPC server
type GRPCServerOptions struct {
	AuthnMiddleware func(http.Handler) http.Handler
	AuthzMiddleware func(http.Handler) http.Handler
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
	return NewGRPCServer(serviceOpts, GRPCServerOptions(httpOpts))
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
		// Strip /api/v2 prefix and forward to gateway
		originalPath := req.URL.Path
		if after, ok := strings.CutPrefix(req.URL.Path, "/api/v2"); ok {
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

// Health implements the health check endpoint for gRPC (used by grpc-gateway)
func (s *Service) Health(ctx context.Context, req *apiv2.HealthRequest) (*apiv2.HealthResponse, error) {
	return &apiv2.HealthResponse{
		Data: &apiv2.HealthData{
			Status: "ok",
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
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
	// Validate required fields
	if req.Name == "" {
		return nil, NewError(http.StatusBadRequest, ErrorMissingField, "Environment name is required")
	}

	// For now, return not implemented since this is OSS
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Environments not implemented in OSS")
}

func (s *Service) FetchPartnerAccounts(ctx context.Context, req *apiv2.FetchAccountsRequest) (*apiv2.FetchAccountsResponse, error) {
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Accounts not implemented in OSS")
}

func (s *Service) FetchAccount(ctx context.Context, req *apiv2.FetchAccountRequest) (*apiv2.FetchAccountResponse, error) {
	// First commit date: 2021-05-13 09:30:04 -0700
	firstCommitTime, err := time.Parse("2006-01-02 15:04:05", "2021-05-13 09:30:04")
	if err != nil {
		return nil, err // NewError something something
	}

	// Return the default dev server account
	account := &apiv2.Account{
		Id:        consts.DevServerAccountID.String(),
		Email:     "dev@inngest.local",
		Name:      "Dev Server",
		CreatedAt: timestamppb.New(firstCommitTime),
		UpdatedAt: timestamppb.New(firstCommitTime),
	}

	return &apiv2.FetchAccountResponse{
		Data: account,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
	}, nil
}

func (s *Service) FetchAccountEventKeys(ctx context.Context, req *apiv2.FetchAccountEventKeysRequest) (*apiv2.FetchAccountEventKeysResponse, error) {
	// Extract environment from X-Inngest-Env header
	envName := GetInngestEnvHeader(ctx)

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
	// Note: envName can be used to filter by environment when implemented
	_ = envName
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Account event keys not implemented in OSS")
}

func (s *Service) FetchAccountEnvs(ctx context.Context, req *apiv2.FetchAccountEnvsRequest) (*apiv2.FetchAccountEnvsResponse, error) {
	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 250 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 250")
		}
	}

	// First commit date: 2021-05-13 09:30:04 -0700
	firstCommitTime, err := time.Parse("2006-01-02 15:04:05", "2021-05-13 09:30:04")
	if err != nil {
		return nil, err // NewError something something
	}

	// Return the default dev server environment
	defaultEnv := &apiv2.Env{
		Id:        consts.DevServerEnvID.String(),
		Name:      "dev",
		Type:      apiv2.EnvType_TEST,
		CreatedAt: timestamppb.New(firstCommitTime),
	}

	return &apiv2.FetchAccountEnvsResponse{
		Data: []*apiv2.Env{defaultEnv},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
		Page: &apiv2.Page{
			HasMore: false,
		},
	}, nil
}

func (s *Service) FetchAccountSigningKeys(ctx context.Context, req *apiv2.FetchAccountSigningKeysRequest) (*apiv2.FetchAccountSigningKeysResponse, error) {
	// Extract environment from X-Inngest-Env header
	envName := GetInngestEnvHeader(ctx)

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// If no signing keys provider is configured, return empty list
	// This happens in dev mode where signing keys aren't required
	if s.signingKeys == nil {
		return &apiv2.FetchAccountSigningKeysResponse{
			Data: []*apiv2.SigningKey{},
			Metadata: &apiv2.ResponseMetadata{
				FetchedAt:   timestamppb.New(time.Now()),
				CachedUntil: nil,
			},
			Page: &apiv2.Page{
				HasMore: false,
			},
		}, nil
	}

	// Get signing keys from the provider
	keys, err := s.signingKeys.GetSigningKeys(ctx)
	if err != nil {
		return nil, NewError(http.StatusInternalServerError, ErrorInternalError, "Failed to fetch signing keys")
	}

	// Filter by environment if specified
	var filteredKeys []*apiv2.SigningKey
	for _, key := range keys {
		if envName == "" || key.Environment == envName {
			filteredKeys = append(filteredKeys, key)
		}
	}

	// For now, return all keys without pagination
	// In a real implementation, you'd handle cursor-based pagination here

	return &apiv2.FetchAccountSigningKeysResponse{
		Data: filteredKeys,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
		Page: &apiv2.Page{
			HasMore: false,
		},
	}, nil
}
