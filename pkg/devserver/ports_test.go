package devserver

import (
	"net"
	"testing"

	"github.com/inngest/inngest/pkg/config"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	"github.com/stretchr/testify/require"
)

func TestResolvePortConflictsUsesNextAvailablePorts(t *testing.T) {
	t.Setenv("DEBUG", "1")

	opts := StartOpts{
		Config: config.Config{
			EventAPI: config.EventAPI{
				Addr: "127.0.0.1",
				Port: 8288,
			},
			CoreAPI: config.CoreAPI{
				Addr: "127.0.0.1",
				Port: 8288,
			},
		},
		ConnectGatewayPort: 8289,
		ConnectGatewayHost: "127.0.0.1",
		ConnectGRPCConfig: connectConfig.ConnectGRPCConfig{
			Gateway: connectConfig.GRPCConfig{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 50052,
			},
			Executor: connectConfig.GRPCConfig{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 50053,
			},
		},
		DebugAPIPort: 7777,
	}

	inUse := map[int]bool{
		7777:  true,
		7778:  true,
		8288:  true,
		8289:  true,
		50052: true,
		50053: true,
	}

	resolved, changes, err := resolvePortConflicts(opts, func(_ string, port int) bool {
		return !inUse[port]
	})
	require.NoError(t, err)

	require.Equal(t, 8290, resolved.Config.EventAPI.Port)
	require.Equal(t, 8290, resolved.Config.CoreAPI.Port)
	require.Equal(t, 8291, resolved.ConnectGatewayPort)
	require.Equal(t, 50054, resolved.ConnectGRPCConfig.Gateway.Port)
	require.Equal(t, 50055, resolved.ConnectGRPCConfig.Executor.Port)
	require.Equal(t, 7779, resolved.DebugAPIPort)
	require.Len(t, changes, 5)
}

func TestResolvePortConflictsLeavesAvailablePortsUnchanged(t *testing.T) {
	opts := StartOpts{
		Config: config.Config{
			EventAPI: config.EventAPI{
				Addr: "127.0.0.1",
				Port: 9000,
			},
			CoreAPI: config.CoreAPI{
				Addr: "127.0.0.1",
				Port: 9000,
			},
		},
		ConnectGatewayPort: 9001,
		ConnectGatewayHost: "127.0.0.1",
		ConnectGRPCConfig: connectConfig.ConnectGRPCConfig{
			Gateway: connectConfig.GRPCConfig{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 9002,
			},
			Executor: connectConfig.GRPCConfig{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 9003,
			},
		},
		DebugAPIPort: 9004,
	}

	resolved, changes, err := resolvePortConflicts(opts, func(_ string, _ int) bool {
		return true
	})
	require.NoError(t, err)

	require.Equal(t, opts, resolved)
	require.Empty(t, changes)
}
