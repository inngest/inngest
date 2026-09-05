#!/usr/bin/env bash
#
# Opens an interactive `duckdb` CLI session against the same database file
# and DuckLake catalog that `inngest dev`/`inngest start --duckdb` use, so
# you can poke at staged/lake data by hand between runs.
#
# Mirrors the layout setupDualWrite builds in pkg/devserver/dualwrite.go and
# the ATTACH sequence bootstrapDuckLakeLocked runs in pkg/db/duckdb/process.go:
#
#   <state-dir>/duckdb/main.duckdb     -- main db file (staging tables)
#   <state-dir>/duckdb/catalog.duckdb  -- DuckLake catalog
#   <state-dir>/duckdb/data/           -- DuckLake data files
#
# state-dir defaults to ".inngest" resolved relative to the CWD, matching
# util.ResolveStateDir's default (consts.DefaultInngestConfigDir) when
# --sqlite-dir/-INNGEST_SQLITE_DIR isn't set. Pass a directory as the first
# argument to point at a non-default --sqlite-dir.

set -euo pipefail

state_dir="${1:-.inngest}"
duckdb_dir="${state_dir%/}/duckdb"
db_file="${duckdb_dir}/main.duckdb"
catalog_file="${duckdb_dir}/catalog.duckdb"
data_dir="${duckdb_dir}/data"

if ! command -v duckdb >/dev/null 2>&1; then
    echo "error: duckdb binary not found on PATH" >&2
    exit 1
fi

if [[ ! -f "${db_file}" ]]; then
    echo "error: ${db_file} does not exist" >&2
    echo "       expected an existing dual-write database (run \`inngest dev --duckdb\` at least once first)" >&2
    exit 1
fi

if [[ ! -f "${catalog_file}" ]]; then
    echo "error: ${catalog_file} does not exist" >&2
    echo "       expected an existing DuckLake catalog (run \`inngest dev --duckdb\` at least once first)" >&2
    exit 1
fi

echo "main db: ${db_file}" >&2
echo "lake catalog: ${catalog_file} (attached as 'inngest')" >&2

exec duckdb "${db_file}" \
    -cmd "INSTALL ducklake;" \
    -cmd "LOAD ducklake;" \
    -cmd "ATTACH IF NOT EXISTS 'ducklake:${catalog_file}' AS inngest (DATA_PATH '${data_dir}/', DATA_INLINING_ROW_LIMIT 1000);"
