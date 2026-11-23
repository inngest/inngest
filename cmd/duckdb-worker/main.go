package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/inngest/inngest/pkg/duckdb/driver"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "duckdb-worker: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("duckdb-worker", flag.ContinueOnError)
	socketPath := fs.String("socket", "", "Unix domain socket path to listen on (required)")
	libPath := fs.String("lib", "", "Path to libduckdb shared library (if empty, lazily downloaded into --state-dir)")
	dbPath := fs.String("db", "", "DuckDB database file path (empty for in-memory)")
	stateDir := fs.String("state-dir", "", "Directory for cached libduckdb downloads")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *socketPath == "" {
		return fmt.Errorf("--socket is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return driver.ServeWorker(ctx, driver.WorkerServerOpts{
		SocketPath: *socketPath,
		LibPath:    *libPath,
		DBPath:     *dbPath,
		StateDir:   *stateDir,
	})
}
