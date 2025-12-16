package apiv2base

import (
	"context"
	"net/http"
	"net/http/httptest"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// applyAuth applies authentication and authorization middleware to a gRPC method
func applyAuth(ctx context.Context, fullMethod string, authnMiddleware, authzMiddleware func(http.Handler) http.Handler) error {
	// Apply authentication middleware if provided (applies to all methods)
	if authnMiddleware != nil {
		authnFunc := HTTPMiddlewareToGRPCInterceptor(authnMiddleware)
		if err := authnFunc(ctx, fullMethod); err != nil {
			return status.Error(codes.Unauthenticated, "authentication failed")
		}
	}

	// Apply authorization middleware if this method requires it
	if requiresAuthorization(fullMethod) {
		if authzMiddleware != nil {
			authzFunc := HTTPMiddlewareToGRPCInterceptor(authzMiddleware)
			if err := authzFunc(ctx, fullMethod); err != nil {
				return status.Error(codes.PermissionDenied, "access denied")
			}
		} else {
			// No authorization middleware provided but authorization required
			return status.Error(codes.PermissionDenied, "authorization not configured")
		}
	}

	return nil
}

// NewAuthUnaryInterceptor creates a unary gRPC interceptor that applies authentication
// and authorization middleware based on protobuf annotations
func NewAuthUnaryInterceptor(authnMiddleware, authzMiddleware func(http.Handler) http.Handler) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := applyAuth(ctx, info.FullMethod, authnMiddleware, authzMiddleware); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// NewAuthStreamInterceptor creates a streaming gRPC interceptor that applies authentication
// and authorization middleware based on protobuf annotations
func NewAuthStreamInterceptor(authnMiddleware, authzMiddleware func(http.Handler) http.Handler) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := applyAuth(ss.Context(), info.FullMethod, authnMiddleware, authzMiddleware); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

// requiresAuthorization checks if a gRPC method requires authorization based on protobuf annotations
func requiresAuthorization(fullMethod string) bool {
	// Parse the method name from the full method path
	// Full method format: "/package.service/MethodName"
	methodName := parseMethodName(fullMethod)
	if methodName == "" {
		return false
	}

	// Get the service descriptor
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		return false
	}

	// Find the method descriptor
	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		if string(method.Name()) == methodName {
			return hasAuthzAnnotation(method)
		}
	}

	return false
}

// parseMethodName extracts the method name from a full gRPC method path
func parseMethodName(fullMethod string) string {
	// Full method format: "/api.v2.V2/MethodName"
	// Find the last slash and extract everything after it
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return ""
}

// HTTPMiddlewareToGRPCInterceptor converts an HTTP middleware function to a gRPC interceptor function
func HTTPMiddlewareToGRPCInterceptor(middleware func(http.Handler) http.Handler) func(ctx context.Context, method string) error {
	return func(ctx context.Context, method string) error {
		// Create a test handler that will succeed if middleware allows the request
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Determine HTTP method from gRPC method and protobuf annotations
		httpMethod := getHTTPMethodForGRPCMethod(method)

		// Create a test request with the appropriate HTTP method
		req := httptest.NewRequest(httpMethod, "/test", nil)
		req = req.WithContext(ctx)

		// Copy gRPC metadata to HTTP headers
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			for key, values := range md {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
		}

		// Create a response recorder
		rec := httptest.NewRecorder()

		// Apply the middleware
		wrappedHandler := middleware(testHandler)
		wrappedHandler.ServeHTTP(rec, req)

		// Check if middleware blocked the request
		if rec.Code != http.StatusOK {
			return status.Error(codes.PermissionDenied, "authorization failed")
		}

		return nil
	}
}

// getHTTPMethodForGRPCMethod determines the HTTP method for a gRPC method by reading protobuf annotations
func getHTTPMethodForGRPCMethod(fullMethod string) string {
	// Parse the method name from the full method path
	methodName := parseMethodName(fullMethod)
	if methodName == "" {
		return http.MethodPost // Default fallback
	}

	// Get the service descriptor
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		return http.MethodPost // Default fallback
	}

	// Find the method descriptor and extract HTTP method from annotations
	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		if string(method.Name()) == methodName {
			return getHTTPMethod(method) // Use refactored function from util.go
		}
	}

	return http.MethodPost // Default fallback
}
