package connect

import (
	"context"
	"net"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/spf13/viper"
)

type ConnectExecutor struct {
	GRPCIP   net.IP
	GRPCPort uint16
}

var (
	executorConfig ConnectExecutor
	executorConfigOnce    sync.Once
)

func Executor(ctx context.Context) ConnectExecutor {
	executorConfigOnce.Do(func() {
		ipKey := "connect.executor.grpc.ip"
		portKey := "connect.executor.grpc.port"

		viper.SetDefault(ipKey, getOutboundIP())
		viper.SetDefault(portKey, 50053)

		ipStr := viper.GetString(ipKey)
		port := viper.GetUint32(portKey)

		ip := net.ParseIP(ipStr)
		if ip == nil {
			logger.StdlibLogger(ctx).Error("invalid connect executor IP", "ip", ipStr)
		}

		executorConfig = ConnectExecutor{
			GRPCIP:   ip,
			GRPCPort: uint16(port),
		}
	})
	return executorConfig
}

