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
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/inngest/inngest/pkg/consts"
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
	functionPauser FunctionPauser
	functionConfig FunctionConfigProvider
	runs           RunProvider
	replays        ReplayProvider
	traces         FunctionTraceReader
	executor       FunctionScheduler
	eventPublisher EventPublisher
	eventSender    EventSender
	maxEventSize   int
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
	FunctionPauser      FunctionPauser
	FunctionConfig      FunctionConfigProvider
	Runs                RunProvider
	Replays             ReplayProvider
	FunctionTraces      FunctionTraceReader
	Executor            FunctionScheduler
	EventPublisher      EventPublisher
	EventSender         EventSender
	MaxEventSize        int
	Scores              ScoreProvider
	RateLimitProvider   RateLimitProvider
}

func NewService(opts ServiceOptions) *Service {
	rateLimiter := opts.RateLimitProvider
	if rateLimiter == nil {
		rateLimiter = noopRateLimitProvider{}
	}
	maxEventSize := opts.MaxEventSize
	if maxEventSize <= 0 {
		maxEventSize = consts.AbsoluteMaxEventSize
	}
	return &Service{
		signingKeys:    opts.SigningKeysProvider,
		eventKeys:      opts.EventKeysProvider,
		apps:           opts.Apps,
		functions:      opts.Functions,
		functionPauser: opts.FunctionPauser,
		functionConfig: opts.FunctionConfig,
		runs:           opts.Runs,
		replays:        opts.Replays,
		traces:         opts.FunctionTraces,
		executor:       opts.Executor,
		eventPublisher: opts.EventPublisher,
		eventSender:    opts.EventSender,
		maxEventSize:   maxEventSize,
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
		runtime.SetQueryParameterParser(&runtime.DefaultQueryParser{RejectUnknownFields: true}),
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

	// Build map of paths that require authorization
	authzPaths := base.BuildAuthzPathMap()
	authzRoutes := chi.NewRouter()
	for path := range authzPaths {
		authzRoutes.Handle(path, gwmux)
	}

	r := chi.NewRouter()
	r.Use(RequestBodyLimitMiddleware(consts.AbsoluteMaxEventSize))

	// Add authentication middleware first
	if httpOpts.AuthnMiddleware != nil {
		r.Use(httpOpts.AuthnMiddleware)
	}

	// Add metrics middleware
	if httpOpts.MetricsMiddleware != nil {
		r.Use(httpOpts.MetricsMiddleware.Middleware)
	}

	r.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Authorization, request validation, and routing must use the same path.
		originalPath := req.URL.Path
		originalRawPath := req.URL.RawPath
		req.URL.Path = normalizedV2Path(req.URL.Path)
		if req.URL.RawPath != "" {
			req.URL.RawPath = normalizedV2Path(req.URL.RawPath)
		}
		defer func() {
			req.URL.Path = originalPath
			req.URL.RawPath = originalRawPath
		}()

		// Apply authorization middleware if this path requires it
		if httpOpts.AuthzMiddleware != nil && authzRoutes.Match(chi.NewRouteContext(), req.Method, req.URL.EscapedPath()) {
			authzHandler := httpOpts.AuthzMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Add JSON validation after authorization for protected paths
				validationHandler := base.JSONTypeValidationMiddleware()(gwmux)
				validationHandler.ServeHTTP(w, r)
			}))
			authzHandler.ServeHTTP(w, req)
		} else {
			// Add JSON validation for unprotected paths
			validationHandler := base.JSONTypeValidationMiddleware()(gwmux)
			validationHandler.ServeHTTP(w, req)
		}

	}))

	return r, nil
}

func normalizedV2Path(path string) string {
	for _, prefix := range []string{"/api/v2", "/v2"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	if path == "" {
		return "/"
	}
	return path
}

// RequestBodyLimitMiddleware rejects oversized bodies before validation reads them.
func RequestBodyLimitMiddleware(maxEventBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		type route struct{ method, path string }
		limits := map[string]int64{"SendEvent": maxEventBytes, "CreateReplay": 64 << 10}
		handlers := map[route]http.Handler{}
		paths := chi.NewRouter()
		for _, endpoint := range apiv2endpoint.Discover() {
			if limit, ok := limits[endpoint.MethodName]; ok {
				paths.Method(endpoint.HTTPMethod, endpoint.Path, next)
				handlers[route{endpoint.HTTPMethod, endpoint.Path}] = apiv2base.MaxRequestBodyBytesMiddleware(limit)(next)
			}
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rctx := chi.NewRouteContext()
			if paths.Match(rctx, r.Method, strings.TrimRight(normalizedV2Path(r.URL.EscapedPath()), "/")) {
				limited := handlers[route{r.Method, rctx.RoutePattern()}]
				limited.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
