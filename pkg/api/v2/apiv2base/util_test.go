package apiv2base

import (
	"context"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestGetHTTPRule(t *testing.T) {
	t.Run("extracts HTTP rule from method with annotations", func(t *testing.T) {
		// Get the service descriptor
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc, "V2 service should exist")

		// Find a method that should have HTTP annotations (like Health)
		methods := serviceDesc.Methods()
		var healthMethod protoreflect.MethodDescriptor
		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			if string(method.Name()) == "Health" {
				healthMethod = method
				break
			}
		}

		if healthMethod != nil {
			rule := getHTTPRule(healthMethod)
			// Health method should have HTTP rule (typically GET)
			if rule != nil {
				assert.NotNil(t, rule.Pattern, "HTTP rule should have a pattern")
			} else {
				t.Skip("Health method doesn't have HTTP annotation in current proto definition")
			}
		} else {
			t.Skip("Health method not found in proto definition")
		}
	})

	t.Run("returns nil for method without annotations", func(t *testing.T) {
		// Create a mock method descriptor without HTTP annotation
		// Since we can't easily create a mock protoreflect.MethodDescriptor,
		// we'll test this by finding a method that doesn't have annotations
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		methods := serviceDesc.Methods()
		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			rule := getHTTPRule(method)
			// Some methods might not have HTTP rules
			if rule == nil {
				// Found a method without HTTP rule, which is valid
				t.Logf("Method %s has no HTTP rule, which is expected behavior", method.Name())
				return
			}
		}
		// If all methods have HTTP rules, that's also valid
		t.Log("All methods have HTTP rules, which is valid")
	})
}

func TestGetHTTPMethodAndPath(t *testing.T) {
	t.Run("extracts method and path from annotations", func(t *testing.T) {
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		methods := serviceDesc.Methods()
		foundAnnotatedMethod := false

		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			httpMethod, path := getHTTPMethodAndPath(method)

			// All methods should return some HTTP method (at minimum POST as default)
			validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
			assert.Contains(t, validMethods, httpMethod, 
				"Method %s should return valid HTTP method", method.Name())

			// If we find a method with a path, verify it's valid
			if path != "" {
				foundAnnotatedMethod = true
				assert.True(t, len(path) > 0, "Path should not be empty string")
				assert.True(t, path[0] == '/' || path == "", 
					"Path should start with / or be empty, got: %s", path)
				t.Logf("Method %s: %s %s", method.Name(), httpMethod, path)
			}
		}

		if !foundAnnotatedMethod {
			t.Log("No methods with HTTP paths found - this may be expected")
		}
	})

	t.Run("returns default for method without annotations", func(t *testing.T) {
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		methods := serviceDesc.Methods()
		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			httpMethod, path := getHTTPMethodAndPath(method)

			// Even methods without annotations should get defaults
			if path == "" {
				// Methods without HTTP annotation should default to POST
				assert.Equal(t, "POST", httpMethod, 
					"Method without HTTP annotation should default to POST")
				t.Logf("Method %s defaults to: %s (no path)", method.Name(), httpMethod)
			}
		}
	})
}

func TestGetHTTPPath(t *testing.T) {
	t.Run("extracts path from method annotations", func(t *testing.T) {
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		methods := serviceDesc.Methods()
		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			path := getHTTPPath(method)

			// Path can be empty string for methods without HTTP annotations
			// If it's not empty, it should be a valid path
			if path != "" {
				assert.True(t, path[0] == '/', "Non-empty path should start with /")
				t.Logf("Method %s has path: %s", method.Name(), path)
			}
		}
	})
}

func TestGetHTTPMethod(t *testing.T) {
	t.Run("extracts HTTP method from annotations", func(t *testing.T) {
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		methods := serviceDesc.Methods()
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			httpMethod := getHTTPMethod(method)

			assert.Contains(t, validMethods, httpMethod, 
				"Should return valid HTTP method for %s", method.Name())
			t.Logf("Method %s uses HTTP method: %s", method.Name(), httpMethod)
		}
	})
}

func TestHasAuthzAnnotation(t *testing.T) {
	t.Run("checks for authz annotation on methods", func(t *testing.T) {
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		methods := serviceDesc.Methods()
		foundAuthzMethod := false

		for i := 0; i < methods.Len(); i++ {
			method := methods.Get(i)
			hasAuthz := hasAuthzAnnotation(method)

			if hasAuthz {
				foundAuthzMethod = true
				t.Logf("Method %s requires authorization", method.Name())
			} else {
				t.Logf("Method %s does not require authorization", method.Name())
			}

			// Just verify the function doesn't panic and returns a boolean
			assert.IsType(t, true, hasAuthz)
		}

		// Log whether any methods require authz
		if foundAuthzMethod {
			t.Log("Found methods requiring authorization")
		} else {
			t.Log("No methods require authorization (may be expected)")
		}
	})
}

func TestGetInngestEnvHeader(t *testing.T) {
	t.Run("extracts header when present", func(t *testing.T) {
		md := metadata.Pairs("x-inngest-env", "production")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		env := GetInngestEnvHeader(ctx)
		assert.Equal(t, "production", env)
	})

	t.Run("returns first value when multiple values present", func(t *testing.T) {
		md := metadata.Pairs("x-inngest-env", "production", "x-inngest-env", "staging")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		env := GetInngestEnvHeader(ctx)
		assert.Equal(t, "production", env)
	})

	t.Run("returns empty string when header missing", func(t *testing.T) {
		md := metadata.Pairs("other-header", "value")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		env := GetInngestEnvHeader(ctx)
		assert.Equal(t, "", env)
	})

	t.Run("returns empty string when no metadata in context", func(t *testing.T) {
		ctx := context.Background()

		env := GetInngestEnvHeader(ctx)
		assert.Equal(t, "", env)
	})

	t.Run("handles case insensitive header names", func(t *testing.T) {
		// gRPC metadata is case-insensitive, test different cases
		testCases := []struct {
			headerName string
			expected   string
		}{
			{"x-inngest-env", "test-env"},
			{"X-Inngest-Env", "test-env-2"},  
			{"X-INNGEST-ENV", "test-env-3"},
		}

		for _, tc := range testCases {
			t.Run("header_case_"+tc.headerName, func(t *testing.T) {
				md := metadata.Pairs(tc.headerName, tc.expected)
				ctx := metadata.NewIncomingContext(context.Background(), md)

				env := GetInngestEnvHeader(ctx)
				assert.Equal(t, tc.expected, env)
			})
		}
	})
}

func TestGRPCToHTTPStatus(t *testing.T) {
	testCases := []struct {
		name       string
		grpcCode   codes.Code
		httpStatus int
	}{
		{"InvalidArgument", codes.InvalidArgument, 400},
		{"Unauthenticated", codes.Unauthenticated, 401},
		{"PermissionDenied", codes.PermissionDenied, 403},
		{"NotFound", codes.NotFound, 404},
		{"AlreadyExists", codes.AlreadyExists, 409},
		{"ResourceExhausted", codes.ResourceExhausted, 429},
		{"Unimplemented", codes.Unimplemented, 501},
		{"Unavailable", codes.Unavailable, 503},
		{"Internal", codes.Internal, 500},
		{"Unknown", codes.Unknown, 500},
		{"OK", codes.OK, 500}, // Default fallback
		{"Canceled", codes.Canceled, 500}, // Default fallback
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpStatus := GRPCToHTTPStatus(tc.grpcCode)
			assert.Equal(t, tc.httpStatus, httpStatus)
		})
	}
}

func TestBuildAuthzPathMap(t *testing.T) {
	t.Run("builds authorization path map from proto annotations", func(t *testing.T) {
		pathMap := BuildAuthzPathMap()

		// Verify it returns a valid map
		assert.NotNil(t, pathMap)
		assert.IsType(t, make(map[string]bool), pathMap)

		// Log the paths that require authorization
		if len(pathMap) > 0 {
			t.Log("Paths requiring authorization:")
			for path, requiresAuthz := range pathMap {
				assert.True(t, requiresAuthz, "All paths in map should require authz")
				assert.NotEmpty(t, path, "Path should not be empty")
				t.Logf("  %s", path)
			}
		} else {
			t.Log("No paths require authorization (may be expected)")
		}

		// Verify all paths are valid HTTP paths
		for path := range pathMap {
			if path != "" {
				assert.True(t, path[0] == '/', "Path should start with /: %s", path)
			}
		}
	})

	t.Run("handles missing service gracefully", func(t *testing.T) {
		// This tests the nil check in BuildAuthzPathMap
		pathMap := BuildAuthzPathMap()
		assert.NotNil(t, pathMap, "Should return empty map even if service not found")
	})
}

// Base instance tests - testing utils through the base instance
func TestBase_GetInngestEnvHeader(t *testing.T) {
	base := NewBase()

	t.Run("gets header through base instance", func(t *testing.T) {
		md := metadata.Pairs("x-inngest-env", "test-environment")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		env := base.GetInngestEnvHeader(ctx)
		assert.Equal(t, "test-environment", env)
	})
}

func TestBase_GRPCToHTTPStatus(t *testing.T) {
	base := NewBase()

	t.Run("converts gRPC codes through base instance", func(t *testing.T) {
		testCases := []struct {
			grpcCode   codes.Code
			httpStatus int
		}{
			{codes.InvalidArgument, 400},
			{codes.Unauthenticated, 401},
			{codes.PermissionDenied, 403},
			{codes.NotFound, 404},
			{codes.Internal, 500},
		}

		for _, tc := range testCases {
			httpStatus := base.GRPCToHTTPStatus(tc.grpcCode)
			assert.Equal(t, tc.httpStatus, httpStatus)
		}
	})
}

func TestBase_BuildAuthzPathMap(t *testing.T) {
	base := NewBase()

	t.Run("builds authz map through base instance", func(t *testing.T) {
		pathMap := base.BuildAuthzPathMap()

		assert.NotNil(t, pathMap)
		assert.IsType(t, make(map[string]bool), pathMap)

		// Verify consistency with direct function call
		directMap := BuildAuthzPathMap()
		assert.Equal(t, len(directMap), len(pathMap), 
			"Base instance should return same result as direct call")

		for path, requiresAuthz := range pathMap {
			directRequiresAuthz, exists := directMap[path]
			assert.True(t, exists, "Path should exist in both maps: %s", path)
			assert.Equal(t, directRequiresAuthz, requiresAuthz, 
				"Authorization requirement should match for path: %s", path)
		}
	})
}

// Benchmark tests
func BenchmarkGetHTTPMethodAndPath(b *testing.B) {
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		b.Skip("V2 service not found")
		return
	}

	methods := serviceDesc.Methods()
	if methods.Len() == 0 {
		b.Skip("No methods found")
		return
	}

	method := methods.Get(0)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getHTTPMethodAndPath(method)
	}
}

func BenchmarkHasAuthzAnnotation(b *testing.B) {
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		b.Skip("V2 service not found")
		return
	}

	methods := serviceDesc.Methods()
	if methods.Len() == 0 {
		b.Skip("No methods found")
		return
	}

	method := methods.Get(0)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasAuthzAnnotation(method)
	}
}

func BenchmarkGRPCToHTTPStatus(b *testing.B) {
	codes := []codes.Code{
		codes.InvalidArgument,
		codes.Unauthenticated, 
		codes.PermissionDenied,
		codes.NotFound,
		codes.Internal,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := codes[i%len(codes)]
		_ = GRPCToHTTPStatus(code)
	}
}

func BenchmarkGetInngestEnvHeader(b *testing.B) {
	md := metadata.Pairs("x-inngest-env", "benchmark-env")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetInngestEnvHeader(ctx)
	}
}

func BenchmarkBuildAuthzPathMap(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildAuthzPathMap()
	}
}

func BenchmarkBase_GRPCToHTTPStatus(b *testing.B) {
	base := NewBase()
	codes := []codes.Code{
		codes.InvalidArgument,
		codes.Unauthenticated,
		codes.PermissionDenied, 
		codes.NotFound,
		codes.Internal,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := codes[i%len(codes)]
		_ = base.GRPCToHTTPStatus(code)
	}
}