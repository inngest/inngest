package apiv2base

import (
	"context"
	"net/http"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// getHTTPRule extracts the HttpRule from google.api.http annotation
func getHTTPRule(method protoreflect.MethodDescriptor) *annotations.HttpRule {
	opts := method.Options()
	if !proto.HasExtension(opts, annotations.E_Http) {
		return nil
	}

	httpRule := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	return httpRule
}

// getHTTPMethodAndPath extracts both HTTP method and path from google.api.http annotation
func getHTTPMethodAndPath(method protoreflect.MethodDescriptor) (httpMethod, path string) {
	httpRule := getHTTPRule(method)
	if httpRule == nil {
		return http.MethodPost, "" // Default for gRPC
	}

	// Extract both method and path from the annotation pattern
	switch pattern := httpRule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet, pattern.Get
	case *annotations.HttpRule_Post:
		return http.MethodPost, pattern.Post
	case *annotations.HttpRule_Put:
		return http.MethodPut, pattern.Put
	case *annotations.HttpRule_Delete:
		return http.MethodDelete, pattern.Delete
	case *annotations.HttpRule_Patch:
		return http.MethodPatch, pattern.Patch
	default:
		return http.MethodPost, "" // Default fallback
	}
}

// getHTTPPath extracts the HTTP path from google.api.http annotation
func getHTTPPath(method protoreflect.MethodDescriptor) string {
	_, path := getHTTPMethodAndPath(method)
	return path
}

// getHTTPMethod extracts the HTTP method from google.api.http annotation
func getHTTPMethod(method protoreflect.MethodDescriptor) string {
	httpMethod, _ := getHTTPMethodAndPath(method)
	return httpMethod
}

// hasAuthzAnnotation checks if a method has the authz annotation requiring authorization
func hasAuthzAnnotation(method protoreflect.MethodDescriptor) bool {
	opts := method.Options()
	if !proto.HasExtension(opts, apiv2.E_Authz) {
		return false
	}

	authzOpts := proto.GetExtension(opts, apiv2.E_Authz).(*apiv2.AuthzOptions)
	return authzOpts.RequireAuthz
}

// GetInngestEnvHeader extracts the X-Inngest-Env header value from the gRPC context.
// Returns an empty string if the header is not present.
func GetInngestEnvHeader(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-inngest-env"); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// GRPCToHTTPStatus maps gRPC codes back to HTTP status codes
func GRPCToHTTPStatus(code codes.Code) int {
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

// BuildAuthzPathMap inspects protobuf annotations to determine which paths require authorization
func BuildAuthzPathMap() map[string]bool {
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
