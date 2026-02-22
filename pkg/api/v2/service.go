package apiv2

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/cqrs"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
)

// Service implements the V2 API service for gRPC with grpc-gateway
type Service struct {
	apiv2.UnimplementedV2Server
	signingKeys SigningKeysProvider
	eventKeys   EventKeysProvider
	data        cqrs.Manager
	base        *apiv2base.Base
}

// ServiceOptions contains configuration for the V2 service
type ServiceOptions struct {
	SigningKeysProvider SigningKeysProvider
	EventKeysProvider   EventKeysProvider
	DataManager         cqrs.Manager
}

func NewService(opts ServiceOptions) *Service {
	return &Service{
		signingKeys: opts.SigningKeysProvider,
		eventKeys:   opts.EventKeysProvider,
		data:        opts.DataManager,
		base:        apiv2base.NewBase(),
	}
}

// GRPCServerOptions contains options for configuring the gRPC server
type GRPCServerOptions struct {
	AuthnMiddleware func(http.Handler) http.Handler
	AuthzMiddleware func(http.Handler) http.Handler
}

// NewGRPCServer creates a new gRPC server with the V2 service and optional interceptors
func NewGRPCServer(serviceOpts ServiceOptions, grpcOpts GRPCServerOptions, base *apiv2base.Base) *grpc.Server {
	var serverOpts []grpc.ServerOption

	// Add authentication and authorization interceptors if any middleware is provided
	if grpcOpts.AuthnMiddleware != nil || grpcOpts.AuthzMiddleware != nil {
		serverOpts = append(serverOpts,
			grpc.UnaryInterceptor(base.NewAuthUnaryInterceptor(grpcOpts.AuthnMiddleware, grpcOpts.AuthzMiddleware)),
			grpc.StreamInterceptor(base.NewAuthStreamInterceptor(grpcOpts.AuthnMiddleware, grpcOpts.AuthzMiddleware)),
		)
	}

	server := grpc.NewServer(serverOpts...)
	service := NewService(serviceOpts)
	apiv2.RegisterV2Server(server, service)

	return server
}

// NewGRPCServerFromHTTPOptions creates a gRPC server using HTTP middleware options
func NewGRPCServerFromHTTPOptions(serviceOpts ServiceOptions, httpOpts HTTPHandlerOptions, base *apiv2base.Base) *grpc.Server {
	return NewGRPCServer(serviceOpts, GRPCServerOptions{
		AuthnMiddleware: httpOpts.AuthnMiddleware,
		AuthzMiddleware: httpOpts.AuthzMiddleware,
	}, base)
}

type HTTPHandlerOptions struct {
	AuthnMiddleware   func(http.Handler) http.Handler
	AuthzMiddleware   func(http.Handler) http.Handler
	MetricsMiddleware api.MetricsMiddleware
}

func NewHTTPHandler(ctx context.Context, serviceOpts ServiceOptions, httpOpts HTTPHandlerOptions, base *apiv2base.Base) (http.Handler, error) {
	// Create the service
	service := NewService(serviceOpts)

	// Create grpc-gateway mux for HTTP REST endpoints with custom error handler
	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(base.CustomErrorHandler()),
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
	authzPaths := base.BuildAuthzPathMap()

	r := chi.NewRouter()

	// Add authentication middleware first
	if httpOpts.AuthnMiddleware != nil {
		r.Use(httpOpts.AuthnMiddleware)
	}

	// Add metrics middleware
	if httpOpts.MetricsMiddleware != nil {
		r.Use(httpOpts.MetricsMiddleware.Middleware)
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
				validationHandler := base.JSONTypeValidationMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gwmux.ServeHTTP(w, r)
				}))
				validationHandler.ServeHTTP(w, r)
			}))
			authzHandler.ServeHTTP(w, req)
		} else {
			// Add JSON validation for unprotected paths
			validationHandler := base.JSONTypeValidationMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gwmux.ServeHTTP(w, r)
			}))
			validationHandler.ServeHTTP(w, req)
		}

		// Restore original path for logging
		req.URL.Path = originalPath
	}))

	return r, nil
}
