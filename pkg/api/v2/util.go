package apiv2

import (
	"net/http"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/genproto/googleapis/api/annotations"
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