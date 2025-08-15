package v2

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func NewAPIV2(o Opts) service.Service {
	return &api{
		rpc:      grpc.NewServer(grpc.UnaryInterceptor(basicAuthInterceptor(o.Username, o.Password))),
		log:      o.Log.With("caller", "api-v2"),
		username: o.Username,
		password: o.Password,
	}
}

type Opts struct {
	Log      logger.Logger
	Username string
	Password string
}

type api struct {
	pb.V2Server

	rpc      *grpc.Server
	gateway  *http.Server
	log      logger.Logger
	username string
	password string
}

func (a *api) Name() string {
	return "api-v2"
}

// basicAuthInterceptor creates a gRPC interceptor for basic authentication
func basicAuthInterceptor(username, password string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth if no credentials configured
		if username == "" && password == "" {
			return handler(ctx, req)
		}

		// Skip auth for Hello method
		if info.FullMethod == "/api.v2.V2/Hello" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		auth := md.Get("authorization")
		if len(auth) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		if !validateBasicAuth(auth[0], username, password) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}

		return handler(ctx, req)
	}
}

// basicAuthMiddleware creates an HTTP middleware for basic authentication
func (a *api) basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no credentials configured
		if a.username == "" && a.password == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for /v2/hello and /v3/hello endpoints
		if r.URL.Path == "/v2/hello" || r.URL.Path == "/v3/hello" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="API v2"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !validateBasicAuth(auth, a.username, a.password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="API v2"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateBasicAuth validates basic auth credentials
func validateBasicAuth(auth, expectedUsername, expectedPassword string) bool {
	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	encoded := strings.TrimPrefix(auth, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	username, password := credentials[0], credentials[1]
	
	// Use subtle.ConstantTimeCompare to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) == 1 &&
		   subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1
}

func (a *api) Pre(ctx context.Context) error {
	pb.RegisterV2Server(a.rpc, a)

	// Setup gateway
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterV2HandlerFromEndpoint(ctx, mux, "localhost:10551", opts)
	if err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	// Apply basic auth middleware to HTTP gateway
	handler := a.basicAuthMiddleware(mux)

	a.gateway = &http.Server{
		Addr:    ":10552",
		Handler: handler,
	}

	return nil
}

func (a *api) Run(ctx context.Context) error {
	grpcAddr := fmt.Sprintf(":%d", 10551)
	httpAddr := ":10552"

	grpcLog := a.log.With("addr", grpcAddr, "proto", "grpc")
	httpLog := a.log.With("addr", httpAddr, "proto", "http")

	// Start gRPC server
	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		grpcLog.Error("could not listen on port for v2 api", "error", err)
		return err
	}

	// Start gRPC server in goroutine
	go func() {
		grpcLog.Info("start grpc api v2")
		if err := a.rpc.Serve(l); err != nil {
			grpcLog.Error("error serving grpc api v2", "error", err)
		}
	}()

	// Start HTTP gateway server
	httpLog.Info("start http gateway api v2")
	if err := a.gateway.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		httpLog.Error("error serving http gateway api v2", "error", err)
		return err
	}

	return nil
}

func (a *api) Stop(ctx context.Context) error {
	a.rpc.GracefulStop()
	if a.gateway != nil {
		if err := a.gateway.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

// APIs
func (a *api) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{
		Msg: "Hello from API v2!",
	}, nil
}

func (a *api) Greetings(ctx context.Context, req *pb.GreetRequest) (*pb.GreetResponse, error) {
	msg := "Hello, World!"
	if req.Name != "" {
		msg = fmt.Sprintf("Hello, %s!", req.Name)
	}

	return &pb.GreetResponse{
		Msg: msg,
	}, nil
}

func (a *api) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	// Extract HTTP headers from gRPC metadata
	headers := make(map[string]string)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for key, values := range md {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}

	return &pb.EchoResponse{
		Data:    req.Data,
		Headers: headers,
	}, nil
}
