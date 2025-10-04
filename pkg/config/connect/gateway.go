package connect

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/spf13/viper"
)

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

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

const GatewayIPKey = "connect.gateway.grpc.ip"
const GatewayPortKey = "connect.gateway.grpc.port"

func getWithDefault[T any](key string, defaultValue T, getter func(string) T) T {
	if viper.IsSet(key) {
		return getter(key)
	}
	return defaultValue
}

func Gateway(ctx context.Context) ConnectGateway {
	configOnce.Do(func() {

		ipStr := getWithDefault(GatewayIPKey, "127.0.0.1", viper.GetString)
		port := getWithDefault(GatewayPortKey, uint32(50052), viper.GetUint32)

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

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
