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

func getHTTPPaths(method protoreflect.MethodDescriptor) []string {
	httpRule := getHTTPRule(method)
	if httpRule == nil {
		return nil
	}

	paths := []string{}
	if path := getHTTPPathFromRule(httpRule); path != "" {
		paths = append(paths, path)
	}
	for _, binding := range httpRule.AdditionalBindings {
		if path := getHTTPPathFromRule(binding); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func getHTTPPathFromRule(httpRule *annotations.HttpRule) string {
	switch pattern := httpRule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return pattern.Get
	case *annotations.HttpRule_Post:
		return pattern.Post
	case *annotations.HttpRule_Put:
		return pattern.Put
	case *annotations.HttpRule_Delete:
		return pattern.Delete
	case *annotations.HttpRule_Patch:
		return pattern.Patch
	default:
		return ""
	}
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

// getAuthzPermission returns the permission declared on a method's authz annotation.
func getAuthzPermission(method protoreflect.MethodDescriptor) string {
	opts := method.Options()
	if !proto.HasExtension(opts, apiv2.E_Authz) {
		return ""
	}

	authzOpts := proto.GetExtension(opts, apiv2.E_Authz).(*apiv2.AuthzOptions)
	return authzOpts.Permission
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
	case codes.FailedPrecondition:
		return http.StatusUnprocessableEntity
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
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
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
			// Get HTTP paths from google.api.http annotation, including additional bindings.
			for _, path := range getHTTPPaths(method) {
				authzPaths[path] = true
			}
		}
	}

	return authzPaths
}

// BuildAuthzPermissionPathMap inspects protobuf annotations to determine each path's required permission.
func BuildAuthzPermissionPathMap() map[string]string {
	authzPaths := make(map[string]string)

	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		return authzPaths
	}

	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		if !hasAuthzAnnotation(method) {
			continue
		}

		for _, path := range getHTTPPaths(method) {
			authzPaths[path] = getAuthzPermission(method)
		}
	}

	return authzPaths
}
