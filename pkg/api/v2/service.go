package apiv2

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Service implements the V2 API service for gRPC with grpc-gateway
type Service struct {
	apiv2.UnimplementedV2Server
	signingKeys    SigningKeysProvider
	eventKeys      EventKeysProvider
	apps           AppProvider
	functions      FunctionProvider
	functionConfig FunctionConfigProvider
	runs           RunProvider
	traces         FunctionTraceReader
	executor       FunctionScheduler
	eventPublisher EventPublisher
	scores         ScoreProvider
	rateLimiter    RateLimitProvider
	base           *apiv2base.Base
}

// ServiceOptions contains configuration for the V2 service
type ServiceOptions struct {
	SigningKeysProvider SigningKeysProvider
	EventKeysProvider   EventKeysProvider
	Apps                AppProvider
	Functions           FunctionProvider
	FunctionConfig      FunctionConfigProvider
	Runs                RunProvider
	FunctionTraces      FunctionTraceReader
	Executor            FunctionScheduler
	EventPublisher      EventPublisher
	Scores              ScoreProvider
	RateLimitProvider   RateLimitProvider
}

func NewService(opts ServiceOptions) *Service {
	rateLimiter := opts.RateLimitProvider
	if rateLimiter == nil {
		rateLimiter = noopRateLimitProvider{}
	}
	return &Service{
		signingKeys:    opts.SigningKeysProvider,
		eventKeys:      opts.EventKeysProvider,
		apps:           opts.Apps,
		functions:      opts.Functions,
		functionConfig: opts.FunctionConfig,
		runs:           opts.Runs,
		traces:         opts.FunctionTraces,
		executor:       opts.Executor,
		eventPublisher: opts.EventPublisher,
		scores:         opts.Scores,
		rateLimiter:    rateLimiter,
		base:           apiv2base.NewBase(),
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
		runtime.WithMarshalerOption(runtime.MIMEWildcard, newResponseEnumMarshaler()),
		runtime.WithErrorHandler(base.CustomErrorHandler()),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			// forward standard headers
			if strings.HasPrefix(strings.ToLower(key), "x-") || key == "authorization" {
				return strings.ToLower(key), true
			}
			return "", false
		}),
		// Allow handlers to override the HTTP status code by setting a
		// "x-http-code" gRPC header via grpc.SetHeader.  This runs before
		// the response body is written, so WriteHeader takes effect.
		runtime.WithForwardResponseOption(func(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
			md, ok := runtime.ServerMetadataFromContext(ctx)
			if !ok {
				return nil
			}
			if vals := md.HeaderMD.Get("x-http-code"); len(vals) > 0 {
				if code, err := strconv.Atoi(vals[0]); err == nil {
					// Remove the metadata key so it doesn't leak as an HTTP header.
					delete(md.HeaderMD, "x-http-code")
					w.Header().Del("Grpc-Metadata-X-Http-Code")
					w.WriteHeader(code)
				}
			}
			return nil
		}),
	)
	if err := apiv2.RegisterV2HandlerServer(ctx, gwmux, service); err != nil {
		return nil, fmt.Errorf("failed to register v2 gateway handler: %w", err)
	}

	// Built once; the matcher compiles the path templates with grpc-gateway's
	// own compiler.
	authzMatcher, err := apiv2base.NewAuthzMatcher(apiv2base.BuildAuthzRoutes())
	if err != nil {
		return nil, fmt.Errorf("failed to build v2 authz route matcher: %w", err)
	}

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
		// Strip supported v2 prefixes and forward to gateway
		originalPath := req.URL.Path
		if after, ok := strings.CutPrefix(req.URL.Path, "/api/v2"); ok {
			req.URL.Path = after
		} else if after, ok := strings.CutPrefix(req.URL.Path, "/v2"); ok {
			req.URL.Path = after
		}

		// Authorization runs here rather than as router middleware because the
		// /api/v2 and /v2 prefixes are only stripped above, so a middleware added
		// via r.Use would see the prefixed path and match no route template.
		//
		// The middleware is invoked only when the route requires a grant: if it is
		// called, authorization is required and the route is on the context.
		serve := base.JSONTypeValidationMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gwmux.ServeHTTP(w, r)
		}))

		route, matched := authzMatcher.Match(req.Method, req.URL.Path)
		switch {
		case !matched:
			// A request the matcher cannot identify is never handed to the
			// gateway. Both route sets are built from the same service
			// descriptor, so a miss is either a genuine 404 or the two
			// disagreeing about the path, and forwarding the second case would
			// run a handler whose grant nobody checked. Answering 404 still gives
			// a mistyped URL a 404, without betting the two can never disagree.
			base.WriteHTTPError(w, http.StatusNotFound, apiv2base.ErrorNotFound,
				"the requested endpoint does not exist")

		case route.Exempt:
			serve.ServeHTTP(w, req)

		case httpOpts.AuthzMiddleware == nil:
			serve.ServeHTTP(w, req)

		default:
			req = req.WithContext(apiv2base.WithAuthzRoute(req.Context(), route))
			httpOpts.AuthzMiddleware(serve).ServeHTTP(w, req)
		}

		// Restore original path for logging
		req.URL.Path = originalPath
	}))

	return r, nil
}
