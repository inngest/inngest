package connect

import (
	"context"
	"net"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/spf13/viper"
)

type ConnectExecutor struct {
	GRPCIP net.IP
}

var (
	executorConfig     ConnectExecutor
	executorConfigOnce sync.Once
)

const (
	ExecutorIPKey   = "connect.executor.grpc.ip"
	ExecutorPortKey = "connect.executor.grpc.port"
)

func Executor(ctx context.Context) ConnectExecutor {
	executorConfigOnce.Do(func() {
		ipStr := getWithDefault(ExecutorIPKey, "127.0.0.1", viper.GetString)

		ip := net.ParseIP(ipStr)
		if ip == nil {
			logger.StdlibLogger(ctx).Error("invalid connect executor IP", "ip", ipStr)
		}

		executorConfig = ConnectExecutor{
			GRPCIP: ip,
		}
	})
	return executorConfig
}

// SetConfig is used for testing
func SetConfig(ctx context.Context, config ConnectExecutor) {
	// Make sure to initialize config to avoid overriding it with sync.Once
	Executor(ctx)

	// Override the config
	executorConfig = config
}
