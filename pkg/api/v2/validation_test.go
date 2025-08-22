package apiv2

import (
	"context"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidateCreateAccountRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *apiv2.CreateAccountRequest
		wantErr     bool
		wantCode    codes.Code
		description string
	}{
		{
			name: "valid email",
			request: &apiv2.CreateAccountRequest{
				Email: "test@example.com",
				Name:  stringPtr("Test User"),
			},
			wantErr:     false,
			description: "should accept valid email format",
		},
		{
			name: "invalid email format",
			request: &apiv2.CreateAccountRequest{
				Email: "invalid-email",
				Name:  stringPtr("Test User"),
			},
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			description: "should reject invalid email format",
		},
		{
			name: "empty email",
			request: &apiv2.CreateAccountRequest{
				Email: "",
				Name:  stringPtr("Test User"),
			},
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			description: "should reject empty email",
		},
		{
			name: "email with spaces",
			request: &apiv2.CreateAccountRequest{
				Email: "test @example.com",
				Name:  stringPtr("Test User"),
			},
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			description: "should reject email with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(tt.request)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRequest() expected error but got none for %s", tt.description)
					return
				}
				
				// Check that it's a gRPC status error with the expected code
				if st, ok := status.FromError(err); ok {
					if st.Code() != tt.wantCode {
						t.Errorf("validateRequest() error code = %v, want %v for %s", st.Code(), tt.wantCode, tt.description)
					}
					t.Logf("Got expected validation error: %v", st.Message())
				} else {
					t.Errorf("validateRequest() error is not a gRPC status error: %v for %s", err, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("validateRequest() error = %v, wantErr %v for %s", err, tt.wantErr, tt.description)
				}
			}
		})
	}
}

func TestValidationInterceptor(t *testing.T) {
	interceptor := NewValidationUnaryInterceptor()
	
	// Mock handler that just returns the request
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return req, nil
	}
	
	// Mock server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/api.v2.V2/CreateAccount",
	}
	
	// Test with valid request
	validReq := &apiv2.CreateAccountRequest{
		Email: "test@example.com",
		Name:  stringPtr("Test User"),
	}
	
	_, err := interceptor(context.Background(), validReq, info, handler)
	if err != nil {
		t.Errorf("interceptor with valid request failed: %v", err)
	}
	
	// Test with invalid request
	invalidReq := &apiv2.CreateAccountRequest{
		Email: "invalid-email",
		Name:  stringPtr("Test User"),
	}
	
	_, err = interceptor(context.Background(), invalidReq, info, handler)
	if err == nil {
		t.Error("interceptor with invalid request should have failed but didn't")
	}
	
	// Verify it's a gRPC InvalidArgument error
	if st, ok := status.FromError(err); ok {
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", st.Code())
		}
	} else {
		t.Errorf("expected gRPC status error, got %T: %v", err, err)
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}