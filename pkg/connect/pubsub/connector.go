package pubsub

type Connector interface {
	RequestForwarder
}

func NewConnector(opts GRPCConnectorOpts) Connector {
	return newGRPCConnector(opts)
}
