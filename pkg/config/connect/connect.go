package connect

import (
	"context"
	"net"
	"strconv"

	"github.com/inngest/inngest/pkg/logger"
)

type GRPCConfig struct {
	IP   net.IP
	Port int
}

type ConnectGRPCConfig struct {
	Gateway  GRPCConfig
	Executor GRPCConfig

	// BindHost is the host the connect listeners bind to. Empty binds to all
	// interfaces.
	BindHost string
}

// ListenAddr returns the address a connect listener should bind to for the
// given port.
func (c ConnectGRPCConfig) ListenAddr(port int) string {
	return net.JoinHostPort(c.BindHost, strconv.Itoa(port))
}

// NewGRPCConfig creates a new GRPC configuration with proper IP parsing and error logging
func NewGRPCConfig(ctx context.Context, gatewayIP string, gatewayPort int, executorIP string, executorPort int) ConnectGRPCConfig {
	parsedGatewayIP := net.ParseIP(gatewayIP)
	if parsedGatewayIP == nil {
		logger.StdlibLogger(ctx).Warn("connect gateway IP not set, this is completely fine if the instance is not a connect gateway", "ip", gatewayIP)
		parsedGatewayIP = net.ParseIP("127.0.0.1") // fallback to localhost
	}

	parsedExecutorIP := net.ParseIP(executorIP)
	if parsedExecutorIP == nil {
		logger.StdlibLogger(ctx).Warn("invalid connect executor IP not set, this is completely fine if the instance is not an executor", "ip", executorIP)
		parsedExecutorIP = net.ParseIP("127.0.0.1") // fallback to localhost
	}

	return ConnectGRPCConfig{
		Gateway: GRPCConfig{
			IP:   parsedGatewayIP,
			Port: gatewayPort,
		},
		Executor: GRPCConfig{
			IP:   parsedExecutorIP,
			Port: executorPort,
		},
	}
}
