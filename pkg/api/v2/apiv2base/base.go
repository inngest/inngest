package apiv2base

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

// Base provides core API v2 functionality for error handling, validation, 
// authentication, and HTTP utilities
type Base struct{}

// NewBase creates a new Base instance
func NewBase() *Base {
	return &Base{}
}

// Error handling methods
func (b *Base) NewError(httpCode int, errorCode, message string) error {
	return NewError(httpCode, errorCode, message)
}

func (b *Base) NewErrors(httpCode int, errors ...ErrorItem) error {
	return NewErrors(httpCode, errors...)
}

// Interceptor creation methods
func (b *Base) NewAuthUnaryInterceptor(authnMiddleware, authzMiddleware func(http.Handler) http.Handler) grpc.UnaryServerInterceptor {
	return NewAuthUnaryInterceptor(authnMiddleware, authzMiddleware)
}

func (b *Base) NewAuthStreamInterceptor(authnMiddleware, authzMiddleware func(http.Handler) http.Handler) grpc.StreamServerInterceptor {
	return NewAuthStreamInterceptor(authnMiddleware, authzMiddleware)
}

// Validation methods
func (b *Base) ValidateJSONForProto(jsonData []byte, msg proto.Message) error {
	return ValidateJSONForProto(jsonData, msg)
}

func (b *Base) JSONTypeValidationMiddleware() func(http.Handler) http.Handler {
	return JSONTypeValidationMiddleware()
}

// HTTP utility methods
func (b *Base) GRPCToHTTPStatus(code codes.Code) int {
	return GRPCToHTTPStatus(code)
}

func (b *Base) BuildAuthzPathMap() map[string]bool {
	return BuildAuthzPathMap()
}

func (b *Base) GetInngestEnvHeader(ctx context.Context) string {
	return GetInngestEnvHeader(ctx)
}

func (b *Base) CustomErrorHandler() func(context.Context, *runtime.ServeMux, runtime.Marshaler, http.ResponseWriter, *http.Request, error) {
	return CustomErrorHandler(b)
}

// Additional validation helper methods for testing
func (b *Base) WriteHTTPError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	writeHTTPError(w, statusCode, errorCode, message)
}

func (b *Base) WriteHTTPErrors(w http.ResponseWriter, statusCode int, errors ...ErrorItem) {
	writeHTTPErrors(w, statusCode, errors...)
}