package apiv2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the V2 API service for gRPC with grpc-gateway
type Service struct {
	apiv2.UnimplementedV2Server
}

func NewService() *Service {
	return &Service{}
}

type HTTPHandlerOptions struct {
	AuthMiddleware func(http.Handler) http.Handler
}

func NewHTTPHandler(ctx context.Context, opts HTTPHandlerOptions) (http.Handler, error) {
	// Create the service
	service := NewService()

	// Create grpc-gateway mux for HTTP REST endpoints
	gwmux := runtime.NewServeMux()
	if err := apiv2.RegisterV2HandlerServer(ctx, gwmux, service); err != nil {
		return nil, fmt.Errorf("failed to register v2 gateway handler: %w", err)
	}

	r := chi.NewRouter()

	if opts.AuthMiddleware != nil {
		r.Use(opts.AuthMiddleware)
	}

	r.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip /api/v2 prefix and forward to gateway
		if after, ok := strings.CutPrefix(r.URL.Path, "/api/v2"); ok {
			r.URL.Path = after
		}
		gwmux.ServeHTTP(w, r)
	}))

	return r, nil
}

// Health implements the health check endpoint for gRPC (used by grpc-gateway)
func (s *Service) Health(ctx context.Context, req *apiv2.HealthRequest) (*apiv2.HealthResponse, error) {
	now := time.Now()

	return &apiv2.HealthResponse{
		Data: &apiv2.HealthData{
			Status: "ok",
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(now),
			CachedUntil: nil, // Health responses are not cached
		},
	}, nil
}
