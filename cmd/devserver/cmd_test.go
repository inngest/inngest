package devserver

import (
	"testing"

	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestCommandConnectGRPCIPFlags(t *testing.T) {
	flags := map[string]cli.Flag{}
	for _, flag := range Command().Flags {
		for _, name := range flag.Names() {
			flags[name] = flag
		}
	}

	for _, name := range []string{"connect-gateway-grpc-ip", "connect-executor-grpc-ip"} {
		t.Run(name, func(t *testing.T) {
			flag, ok := flags[name]
			require.True(t, ok)
			require.Equal(t, connectgrpc.DefaultConnectGRPCIP, flag.Get())
		})
	}
}
