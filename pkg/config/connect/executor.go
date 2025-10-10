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

type ConnectExecutor struct {
	GRPCIP   net.IP
	GRPCPort uint16
}

var (
	executorConfig     ConnectExecutor
	executorConfigOnce sync.Once
)

const ExecutorIPKey = "connect.executor.grpc.ip"
const ExecutorPortKey = "connect.executor.grpc.port"

func Executor(ctx context.Context) ConnectExecutor {
	executorConfigOnce.Do(func() {

		ipStr := getWithDefault(ExecutorIPKey, "127.0.0.1", viper.GetString)
		port := getWithDefault(ExecutorPortKey, uint32(50053), viper.GetUint32)

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
