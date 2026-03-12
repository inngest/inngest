package devserver

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type PortChange struct {
	Name string
	From int
	To   int
}

type portAssignment struct {
	name string
	addr string
	port int
	set  func(*StartOpts, int)
}

func ResolvePortConflicts(opts StartOpts) (StartOpts, []PortChange, error) {
	return resolvePortConflicts(opts, portIsAvailable)
}

func resolvePortConflicts(opts StartOpts, isAvailable func(string, int) bool) (StartOpts, []PortChange, error) {
	resolved := opts
	changes := []PortChange{}
	reserved := map[int]struct{}{}

	assignments := []portAssignment{
		{
			name: "dev server api",
			addr: resolved.Config.EventAPI.Addr,
			port: resolved.Config.EventAPI.Port,
			set: func(opts *StartOpts, port int) {
				opts.Config.EventAPI.Port = port
				opts.Config.CoreAPI.Port = port
			},
		},
		{
			name: "connect gateway",
			addr: resolved.ConnectGatewayHost,
			port: resolved.ConnectGatewayPort,
			set: func(opts *StartOpts, port int) {
				opts.ConnectGatewayPort = port
			},
		},
		{
			name: "connect gateway grpc",
			addr: resolved.ConnectGRPCConfig.Gateway.IP.String(),
			port: resolved.ConnectGRPCConfig.Gateway.Port,
			set: func(opts *StartOpts, port int) {
				opts.ConnectGRPCConfig.Gateway.Port = port
			},
		},
		{
			name: "connect executor grpc",
			addr: resolved.ConnectGRPCConfig.Executor.IP.String(),
			port: resolved.ConnectGRPCConfig.Executor.Port,
			set: func(opts *StartOpts, port int) {
				opts.ConnectGRPCConfig.Executor.Port = port
			},
		},
	}

	if os.Getenv("DEBUG") != "" {
		assignments = append(assignments, portAssignment{
			name: "debug api",
			port: resolved.DebugAPIPort,
			set: func(opts *StartOpts, port int) {
				opts.DebugAPIPort = port
			},
		})
	}

	for _, assignment := range assignments {
		if assignment.port == 0 {
			continue
		}

		port, err := nextAvailablePort(assignment.addr, assignment.port, reserved, isAvailable)
		if err != nil {
			return opts, nil, fmt.Errorf("could not find open port for %s: %w", assignment.name, err)
		}

		reserved[port] = struct{}{}
		assignment.set(&resolved, port)

		if port != assignment.port {
			changes = append(changes, PortChange{
				Name: assignment.name,
				From: assignment.port,
				To:   port,
			})
		}
	}

	return resolved, changes, nil
}

func nextAvailablePort(addr string, start int, reserved map[int]struct{}, isAvailable func(string, int) bool) (int, error) {
	if start <= 0 || start > 65535 {
		return 0, fmt.Errorf("invalid port %d", start)
	}

	for port := start; port <= 65535; port++ {
		if _, ok := reserved[port]; ok {
			continue
		}
		if isAvailable(addr, port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no ports available starting at %d", start)
}

func portIsAvailable(addr string, port int) bool {
	listener, err := net.Listen("tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
