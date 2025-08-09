package v2

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
)

func NewAPIV2() service.Service {
	return &api{
		rpc: grpc.NewServer(),
	}
}

type api struct {
	pb.V2Server

	rpc *grpc.Server
}

func (a *api) Name() string {
	return "api-v2"
}

func (a *api) Pre(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (a *api) Run(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (a *api) Stop(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
