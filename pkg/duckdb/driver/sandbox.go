package driver

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
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
	stmts, err := sandboxStmts(p.iceberg)
	if err != nil {
		return err
	}
	return p.bootstrapExecLocked(ctx, "sandbox", stmts)
}

// sandboxStmts builds the statements that turn off DuckDB's external access
// for the rest of the subprocess's life, keeping only what the Iceberg lake
// itself needs reachable. Every setting involved is GLOBAL in DuckDB (there
// is no per-connection scope), so this applies to every connection —
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
//     automatically. The Iceberg REST calls go through the same check
//     ("Cannot access http://127.0.0.1:.../v1/... - file system operations
//     are disabled by configuration"), so the catalog URI is allowed as a
//     directory prefix, and so is the warehouse's DataPath since data and
//     metadata files are read/written on demand.
//   - NOTE: the catalog's secret stays registered, so SQL in this process
//     can still reach the catalog's REST API w/ its token (IE through the
//     iceberg extension's own functions). The catalog only serves this
//     warehouse, so that's no more than the lake attach can already do.
//   - lock_configuration is deliberately NOT set: the iceberg extension
//     changes settings internally.
func sandboxStmts(iceberg *IcebergOptions) ([]string, error) {
	var dirs []string
	if iceberg != nil {
		dirs = append(dirs, strings.TrimSuffix(iceberg.URI, "/")+"/")
		if iceberg.DataPath != "" {
			abs, err := filepath.Abs(iceberg.DataPath)
			if err != nil {
				return nil, fmt.Errorf("duckdb: resolving Iceberg data path %q: %w", iceberg.DataPath, err)
			}
			dirs = append(dirs, strings.TrimSuffix(abs, string(filepath.Separator))+string(filepath.Separator))
		}
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

// ResourceLimits caps the DuckDB subprocess's resource use -- see
// Options.ResourceLimits. A zero field leaves that setting at DuckDB's own
// default.
type ResourceLimits struct {
	// MemoryLimit is DuckDB's memory_limit (e.g. "2GiB"). NOTE: it only
	// bounds the buffer manager, so it doesn't stop every runaway statement
	// (one huge repeat()/lpad()/range() value is allocated outside it, see
	// pkg/duckdb/insights' size guard), but it does make big joins/sorts/
	// aggregates spill or fail instead of taking the host's RAM.
	MemoryLimit string
	// Threads is DuckDB's threads.
	Threads int
	// MaxTempDirectorySize is DuckDB's max_temp_directory_size (e.g.
	// "4GiB"), bounding how much a statement over MemoryLimit can spill to
	// disk.
	MaxTempDirectorySize string
}

// DefaultResourceLimits returns conservative limits for a subprocess that
// both dual-writes and runs Insights queries. The limits are per-process,
// so a heavy Insights query and dual-write's own writes share them: they're
// sized for dev-scale data, leaving dual-write's small batched INSERTs
// plenty of headroom while keeping a runaway query from taking the whole
// machine. Fixed sizes rather than a percentage of RAM because DuckDB only
// accepts "N%" for memory_limit in newer releases (1.5.5 rejects it).
func DefaultResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		MemoryLimit:          "2GiB",
		Threads:              max(1, min(4, runtime.NumCPU()/2)),
		MaxTempDirectorySize: "4GiB",
	}
}

// withMaxAbandoned sets maxAbandoned on a process being constructed -- see
// Options.MaxAbandonedStatements.
func withMaxAbandoned(n int) processOption {
	return func(p *process) { p.maxAbandoned = n }
}

// withResourceLimits sets resourceLimits on a process being constructed --
// see Options.ResourceLimits.
func withResourceLimits(limits *ResourceLimits) processOption {
	return func(p *process) { p.resourceLimits = limits }
}

// bootstrapResourceLimitsLocked assumes mu is already held. It is a no-op
// unless the caller set Options.ResourceLimits. Every setting is GLOBAL
// (there is no per-connection memory/thread limit in DuckDB) and, unlike
// the sandbox, stays changeable afterwards, so it runs before
// bootstrapSandboxLocked only for tidiness. Insights can't change them back
// since it only ever runs a validated SELECT.
func (p *process) bootstrapResourceLimitsLocked(ctx context.Context) error {
	if p.resourceLimits == nil {
		return nil
	}
	stmts, err := resourceLimitStmts(*p.resourceLimits)
	if err != nil {
		return err
	}
	return p.bootstrapExecLocked(ctx, "resource limits", stmts)
}

// resourceLimitStmts builds the SET GLOBAL statements for limits' non-zero
// fields.
func resourceLimitStmts(limits ResourceLimits) ([]string, error) {
	var stmts []string
	for _, s := range []struct{ name, value string }{
		{"memory_limit", limits.MemoryLimit},
		{"max_temp_directory_size", limits.MaxTempDirectorySize},
	} {
		if s.value == "" {
			continue
		}
		lit, err := encodeLiteral(s.value)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding %s: %w", s.name, err)
		}
		stmts = append(stmts, fmt.Sprintf("SET GLOBAL %s=%s;", s.name, lit))
	}
	if limits.Threads > 0 {
		stmts = append(stmts, fmt.Sprintf("SET GLOBAL threads=%d;", limits.Threads))
	}
	return stmts, nil
}
