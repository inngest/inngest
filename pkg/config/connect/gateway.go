package connect

import (
	"context"
	"net"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/spf13/viper"
)

type ConnectGateway struct {
	// To be used only by the connect-gateway service. Executors will get the IPs dynamically
	// from the Gateway metadata.
	GRPCIP   net.IP
	GRPCPort uint16
}

var (
	gatewayConfig ConnectGateway
	configOnce    sync.Once
)

func Gateway(ctx context.Context) ConnectGateway {
	configOnce.Do(func() {
		ipKey := "connect.gateway.grpc.ip"
		portKey := "connect.gateway.grpc.port"

		viper.SetDefault(ipKey, getOutboundIP())
		viper.SetDefault(portKey, 50052)

		ipStr := viper.GetString(ipKey)
		port := viper.GetUint32(portKey)

		ip := net.ParseIP(ipStr)
		if ip == nil {
			logger.StdlibLogger(ctx).Error("invalid connect gateway IP", "ip", ipStr)
		}

		gatewayConfig = ConnectGateway{
			GRPCIP:   ip,
			GRPCPort: uint16(port),
		}
	})
	return gatewayConfig
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
