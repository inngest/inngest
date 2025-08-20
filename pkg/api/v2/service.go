package apiv2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
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

func NewHTTPHandler(ctx context.Context, opts HTTPHandlerOptions) (http.Handler, error) {
	// Create the service
	service := NewService()

	// Create grpc-gateway mux for HTTP REST endpoints
	gwmux := runtime.NewServeMux()
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

// CreateAccount implements a protected endpoint that requires authorization
func (s *Service) CreateAccount(ctx context.Context, req *apiv2.CreateAccountRequest) (*apiv2.CreateAccountResponse, error) {
	now := time.Now()

	return &apiv2.CreateAccountResponse{
		Data: &apiv2.CreateAccountData{
			ApiKey: "IllBeARealKeySomeday",
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(now),
			CachedUntil: nil,
		},
	}, nil
}
