package pubsub

import (
	"context"
	"github.com/inngest/inngest/pkg/logger"

	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func WithNoop() ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		return noopConnector{}, nil
	}
}

// noopConnector is a blank implementation of the Connector interface
type noopConnector struct{}

func (noopConnector) Proxy(ctx, traceCtx context.Context, opts ProxyOpts) (*connpb.SDKResponse, error) {
	logger.StdlibLogger(ctx).Error("using no-op connector to proxy message", "opts", opts)

	return &connpb.SDKResponse{}, nil
}

