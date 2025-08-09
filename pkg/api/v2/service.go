package v2

import (
	"context"
	"fmt"
	"net"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
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

	rpc *grpc.Server
	log logger.Logger
}

func (a *api) Name() string {
	return "api-v2"
}

func (a *api) Pre(ctx context.Context) error {
	pb.RegisterV2Server(a.rpc, a)

	return nil
}

func (a *api) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", 10551)

	log := a.log.With("addr", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("could not listen on port for v2 api", "error", err)
		return err
	}

	log.Info("start api v2", "addr", addr)
	err = a.rpc.Serve(l)
	if err != nil {
		log.Error("error serving api v2", "error", err)
		return err
	}

	return nil
}

func (a *api) Stop(ctx context.Context) error {
	a.rpc.GracefulStop()
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
