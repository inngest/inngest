package connect

import (
	"context"
	"net"

	"github.com/inngest/inngest/pkg/logger"
)

type GRPCConfig struct {
	IP   net.IP
	Port int
}

type ConnectGRPCConfig struct {
	Gateway  GRPCConfig
	Executor GRPCConfig
}

// NewGRPCConfig creates a new GRPC configuration with proper IP parsing and error logging
func NewGRPCConfig(ctx context.Context, gatewayIP string, gatewayPort int, executorIP string, executorPort int) ConnectGRPCConfig {
	parsedGatewayIP := net.ParseIP(gatewayIP)
	if parsedGatewayIP == nil {
		logger.StdlibLogger(ctx).Error("invalid connect gateway IP", "ip", gatewayIP)
		parsedGatewayIP = net.ParseIP("127.0.0.1") // fallback to localhost
	}

	parsedExecutorIP := net.ParseIP(executorIP)
	if parsedExecutorIP == nil {
		logger.StdlibLogger(ctx).Error("invalid connect executor IP", "ip", executorIP)
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
