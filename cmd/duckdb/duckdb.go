// Package duckdb provides the `inngest duckdb` command group for managing
// the DuckDB CLI binary the experimental DuckDB features (dual-write,
// cmd/duckdbseed) spawn as a subprocess.
package duckdb

import (
	"context"
	"fmt"
	"os"

	dbduckdb "github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/util"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "duckdb",
		Usage: "Manage the DuckDB CLI binary used by Inngest's experimental DuckDB features",
		Commands: []*cli.Command{
			downloadCommand(),
		},
	}
}

func downloadCommand() *cli.Command {
	return &cli.Command{
		Name: "download",
		Usage: fmt.Sprintf(
			"Download the pinned DuckDB CLI binary (%s) into the state directory",
			dbduckdb.PinnedVersion,
		),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "state-dir",
				Usage: "Directory for persistent state (shared with `inngest dev --sqlite-dir`); " +
					"the binary is cached under <dir>/bin. Defaults to .inngest.",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Re-download even if a cached binary is already present.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runDownload(ctx, cmd, cmd.String("state-dir"), cmd.Bool("force"))
		},
	}
}

// runDownload is downloadCommand's Action. stateDirFlag is the raw
// --state-dir value ("" meaning "the default"), resolved the same way
// pkg/devserver's setupDualWrite resolves --sqlite-dir, so `inngest duckdb
// download` and `inngest dev --duckdb` agree on where the binary lives.
func runDownload(ctx context.Context, cmd *cli.Command, stateDirFlag string, force bool) error {
	stateDir, err := util.ResolveStateDir(stateDirFlag)
	if err != nil {
		return fmt.Errorf("resolving state directory: %w", err)
	}

	fmt.Fprintf(cmd.Root().Writer, "downloading duckdb %s into %s\n", dbduckdb.PinnedVersion, stateDir)

	path := dbduckdb.BinaryPath(stateDir)
	if err := removeCachedBinaryIfForced(path, force); err != nil {
		return err
	}

	resolved, err := dbduckdb.EnsureBinary(ctx, stateDir)
	if err != nil {
		return fmt.Errorf("downloading duckdb binary: %w", err)
	}

	fmt.Fprintf(cmd.Root().Writer, "duckdb %s ready at %s\n", dbduckdb.PinnedVersion, resolved)
	return nil
}

// removeCachedBinaryIfForced deletes the binary already cached at path when
// force is set, so EnsureBinary's cache-hit short-circuit can't paper over a
// --force request with a stale file. A no-op (not an error) if nothing is
// cached there yet.
func removeCachedBinaryIfForced(path string, force bool) error {
	if !force {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing cached binary at %q: %w", path, err)
	}
	return nil
}
