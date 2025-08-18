package apiv2

import (
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// getHTTPPath extracts the HTTP path from google.api.http annotation
func getHTTPPath(method protoreflect.MethodDescriptor) string {
	opts := method.Options()
	if !proto.HasExtension(opts, annotations.E_Http) {
		return ""
	}
	
	httpRule := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	if httpRule == nil {
		return ""
	}
	
	// Extract path from the appropriate HTTP method
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
	}
	
	return ""
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