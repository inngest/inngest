package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/libduckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver/internal/worker"
)

// WorkerServerOpts configures ServeWorker.
type WorkerServerOpts struct {
	// SocketPath is the Unix domain socket path to listen on (required).
	SocketPath string
	// LibPath is an optional path to libduckdb. If empty, EnsureLibrary lazily
	// downloads and verifies libduckdb into StateDir.
	LibPath string
	// DBPath is the DuckDB database path ("" or ":memory:" for in-memory).
	DBPath string
	// StateDir is the cache directory used when LibPath is empty.
	StateDir string
	// Config holds optional DuckDB database configuration parameters.
	Config map[string]string
}

// ServeWorker loads libduckdb via purego (downloading it lazily if LibPath is
// empty), opens the DuckDB database, and serves the Unix domain socket worker
// protocol until ctx is cancelled or a Shutdown RPC is received.
func ServeWorker(ctx context.Context, opts WorkerServerOpts) error {
	if opts.SocketPath == "" {
		return fmt.Errorf("duckdb worker: SocketPath is required")
	}

	resolvedLib := opts.LibPath
	if resolvedLib == "" {
		dir := opts.StateDir
		if dir == "" {
			cacheDir, err := os.UserCacheDir()
			if err != nil {
				return fmt.Errorf("duckdb worker: resolving user cache dir: %w", err)
			}
			dir = filepath.Join(cacheDir, "inngest", "duckdb")
		}
		var err error
		resolvedLib, err = EnsureLibrary(ctx, dir)
		if err != nil {
			return fmt.Errorf("duckdb worker: ensuring libduckdb: %w", err)
		}
	}

	lib, err := libduckdb.Load(resolvedLib)
	if err != nil {
		return fmt.Errorf("duckdb worker: loading libduckdb: %w", err)
	}
	defer lib.Close()

	db, err := lib.Open(opts.DBPath, opts.Config)
	if err != nil {
		return fmt.Errorf("duckdb worker: opening database: %w", err)
	}
	defer db.Close()

	srv := worker.NewServer(db, lib.Version())
	return srv.ListenAndServe(ctx, opts.SocketPath)
}
