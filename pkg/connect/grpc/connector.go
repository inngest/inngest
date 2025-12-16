package grpc

import "context"

type Connector interface {
	RequestForwarder
}

func NewConnector(ctx context.Context, opts GRPCConnectorOpts, options ...GRPCConnectorOption) Connector {
	return newGRPCConnector(ctx, opts, options...)
}
