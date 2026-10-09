package driver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// withRestrictExternalAccess sets restrictExternalAccess on a process being
// constructed — see Options.RestrictExternalAccess.
func withRestrictExternalAccess(restrict bool) processOption {
	return func(p *process) { p.restrictExternalAccess = restrict }
}

// bootstrapSandboxLocked assumes mu is already held. It is a no-op unless the
// caller set Options.RestrictExternalAccess. It must run last in
// initSessionLocked: once enable_external_access is off, INSTALL/LOAD, a new
// ATTACH, and changing extension/home directories all fail, so every other
// bootstrap phase has to be done by then.
func (p *process) bootstrapSandboxLocked(ctx context.Context) error {
	if !p.restrictExternalAccess {
		return nil
	}
	stmts, err := sandboxStmts(p.duckLake)
	if err != nil {
		return err
	}
	return p.bootstrapExecLocked(ctx, "sandbox", stmts)
}

// sandboxStmts builds the statements that turn off DuckDB's external access
// for the rest of the subprocess's life, keeping only what the DuckLake
// catalog itself needs reachable. Every setting involved is GLOBAL in DuckDB
// (there is no per-connection scope), so this applies to every connection —
// dual-write's included — not just Insights'. Verified against the real
// binary:
//
//   - enable_external_access=false blocks read_text/read_csv/glob/ATTACH/
//     COPY/INSTALL/LOAD against anything outside allowed_directories/
//     allowed_paths, and getenv() outright.
//   - It is one-way: DuckDB refuses to turn it back on while the database is
//     running, and refuses any further change to allowed_directories/
//     allowed_paths once it is off. So no later statement can widen it.
//   - The database file's own temp directory is added to allowed_directories
//     automatically, and an already-attached catalog (DuckDB or SQLite
//     file, Postgres, quack) keeps working; only DuckLake's DATA_PATH needs
//     listing, since DuckLake opens Parquet files under it on demand.
//   - lock_configuration is deliberately NOT set: DuckLake changes settings
//     internally, so locking the configuration makes every DuckLake
//     statement fail ("Failed to query most recent snapshot").
func sandboxStmts(duckLake *DuckLakeOptions) ([]string, error) {
	var dirs []string
	if duckLake != nil && duckLake.DataPath != "" {
		abs, err := filepath.Abs(duckLake.DataPath)
		if err != nil {
			return nil, fmt.Errorf("duckdb: resolving DuckLake data path %q: %w", duckLake.DataPath, err)
		}
		dirs = append(dirs, strings.TrimSuffix(abs, string(filepath.Separator))+string(filepath.Separator))
	}

	var stmts []string
	if len(dirs) > 0 {
		lit, err := encodeLiteral(dirs)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding allowed directories: %w", err)
		}
		stmts = append(stmts, fmt.Sprintf("SET allowed_directories=%s;", lit))
	}
	stmts = append(stmts, "SET enable_external_access=false;")
	return stmts, nil
}
