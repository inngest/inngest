package v2

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAPIV2(o Opts) service.Service {
	return &api{
		rpc: grpc.NewServer(),
		log: o.Log.With("caller", "api-v2"),
	}
}

type Opts struct {
	Log logger.Logger
}

type api struct {
	pb.V2Server

	rpc     *grpc.Server
	gateway *http.Server
	log     logger.Logger
}

func (a *api) Name() string {
	return "api-v2"
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

	a.gateway = &http.Server{
		Addr:    ":10552",
		Handler: mux,
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
	return &pb.EchoResponse{
		Data: []byte("Echo response"),
	}, nil
}
