package connect

import (
	"context"
	"github.com/inngest/inngest/pkg/service"
	"net/http"
)

type connectGatewaySvc struct{}

func NewConnectService() (service.Service, http.Handler) {
	svc := &connectGatewaySvc{}

	return svc
}

func (c connectGatewaySvc) Name() string {
	return "connect-gateway"
}

func (c connectGatewaySvc) Pre(ctx context.Context) error {
	return nil
}

func (c connectGatewaySvc) Run(ctx context.Context) error {
	return nil
}

func (c connectGatewaySvc) Stop(ctx context.Context) error {
	return nil
}
