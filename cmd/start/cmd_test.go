package start

import (
	"testing"

	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestCommandConnectGRPCFlags(t *testing.T) {
	flags := map[string]cli.Flag{}
	for _, flag := range Command().Flags {
		for _, name := range flag.Names() {
			flags[name] = flag
		}
	}

	tests := []struct {
		name  string
		value any
	}{
		{name: "connect-gateway-grpc-ip", value: connectgrpc.DefaultConnectGRPCIP},
		{name: "connect-executor-grpc-ip", value: connectgrpc.DefaultConnectGRPCIP},
		{name: "connect-gateway-grpc-port", value: devserver.DefaultConnectGatewayGRPCPort},
		{name: "connect-executor-grpc-port", value: devserver.DefaultConnectExecutorGRPCPort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, ok := flags[tt.name]
			require.True(t, ok)
			require.Equal(t, tt.value, flag.Get())
		})
	}
}
