package devserver

import (
	"fmt"
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
		APIGRPCPort:  8290,
	}

	inUse := map[int]bool{
		50051: true,
		7777:  true,
		7778:  true,
		8288:  true,
		8289:  true,
		8290:  true,
		50052: true,
		50053: true,
	}

	resolved, changes, err := resolvePortConflicts(opts, func(_ string, port int) bool {
		return !inUse[port]
	})
	require.NoError(t, err)

	require.Equal(t, 8291, resolved.Config.EventAPI.Port)
	require.Equal(t, 8291, resolved.Config.CoreAPI.Port)
	require.Equal(t, 8292, resolved.ConnectGatewayPort)
	require.Equal(t, 50054, resolved.ConnectGRPCConfig.Gateway.Port)
	require.Equal(t, 50055, resolved.ConnectGRPCConfig.Executor.Port)
	require.Equal(t, 8293, resolved.APIGRPCPort)
	require.Equal(t, 7779, resolved.DebugAPIPort)
	require.Len(t, changes, 6)
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
		APIGRPCPort:  9005,
	}

	resolved, changes, err := resolvePortConflicts(opts, func(_ string, _ int) bool {
		return true
	})
	require.NoError(t, err)

	require.Equal(t, opts, resolved)
	require.Empty(t, changes)
}

func TestResolvePortConflictsChecksWildcardPortsForConnectServices(t *testing.T) {
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
		APIGRPCPort: 8290,
	}

	seen := map[string]bool{}

	resolved, changes, err := resolvePortConflicts(opts, func(addr string, port int) bool {
		seen[fmt.Sprintf("%s:%d", addr, port)] = true
		return true
	})
	require.NoError(t, err)

	require.Equal(t, opts, resolved)
	require.Empty(t, changes)
	require.True(t, seen["127.0.0.1:8288"])
	require.True(t, seen[":8289"])
	require.True(t, seen[":8290"])
	require.True(t, seen[":50052"])
	require.True(t, seen[":50053"])
}

func TestResolvePortConflictsSkipsDisabledAPIGRPCPort(t *testing.T) {
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
		APIGRPCPort: -1,
	}

	seen := map[int]bool{}

	resolved, changes, err := resolvePortConflicts(opts, func(_ string, port int) bool {
		seen[port] = true
		return true
	})
	require.NoError(t, err)

	require.Equal(t, -1, resolved.APIGRPCPort)
	require.Empty(t, changes)
	require.False(t, seen[-1])
}
