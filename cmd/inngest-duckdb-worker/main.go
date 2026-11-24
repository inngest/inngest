package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/inngest/inngest/pkg/duckdb/worker/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "inngest-duckdb-worker: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("inngest-duckdb-worker", flag.ContinueOnError)
	listenAddr := fs.String("listen", "", "Address to listen on: Unix socket path (/path.sock, unix:///path.sock) or TCP address (:9090, tcp://0.0.0.0:9090)")
	socketPath := fs.String("socket", "", "Alias for --listen (Unix domain socket path)")
	healthAddr := fs.String("health-addr", "", "Optional HTTP listen address for GET /healthz (e.g. :8080)")
	libPath := fs.String("lib", "", "Path to libduckdb shared library (if empty, lazily downloaded into --state-dir)")
	dbPath := fs.String("db", "", "DuckDB database file path (empty for in-memory)")
	stateDir := fs.String("state-dir", "", "Directory for cached libduckdb downloads")
	authToken := fs.String("auth-token", "", "Shared auth token required on RPCs (falls back to INNGEST_DUCKDB_WORKER_AUTH_TOKEN; required for non-loopback TCP)")
	allowUnsigned := fs.Bool("allow-unsigned-extensions", false, "Allow loading unsigned DuckDB extensions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	addr := *listenAddr
	if addr == "" {
		addr = *socketPath
	}
	if addr == "" {
		return fmt.Errorf("--listen (or --socket) is required")
	}

	token := *authToken
	if token == "" {
		token = os.Getenv("INNGEST_DUCKDB_WORKER_AUTH_TOKEN")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return server.Serve(ctx, server.Opts{
		ListenAddr:              addr,
		HealthAddr:              *healthAddr,
		AuthToken:               token,
		LibPath:                 *libPath,
		DBPath:                  *dbPath,
		StateDir:                *stateDir,
		AllowUnsignedExtensions: *allowUnsigned,
	})
}
