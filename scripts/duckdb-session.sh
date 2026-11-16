#!/usr/bin/env bash
#
# Opens an interactive `duckdb` CLI session over the persisted dual-write
# lake that `inngest dev`/`inngest start --duckdb` write, so you can poke at
# lake data by hand between runs.
#
# Mirrors the layout setupDualWrite builds in pkg/devserver/duckdb.go:
#
#   <state-dir>/duckdb/main.duckdb      -- main db file (views + macros, see pkg/db/duckdb/views.sql)
#   <state-dir>/duckdb/catalog.sqlite   -- embedded Iceberg catalog
#   <state-dir>/duckdb/warehouse/       -- Iceberg data + metadata files
#
# The Iceberg catalog only runs inside the dev server (on a random port w/ a
# per-process token), so this doesn't attach it. Instead each lake table is
# a read-only `inngest.<table>` view over iceberg_scan of its current
# metadata file (the catalog's `metadata_location`, read straight out of
# catalog.sqlite), and main.duckdb is attached read-only as `local`. Stop the
# dev server first (it holds main.duckdb's lock).
#
# W/ a postgres catalog (--postgres-uri) there's no catalog.sqlite, so this
# falls back to each table's highest metadata version on disk, which can be a
# newer uncommitted file if the dev server died mid-commit.
#
# state-dir defaults to ".inngest" resolved relative to the CWD, matching
# util.ResolveStateDir's default (consts.DefaultInngestConfigDir) when
# --sqlite-dir/-INNGEST_SQLITE_DIR isn't set. Pass a directory as the first
# argument to point at a non-default --sqlite-dir.

set -euo pipefail

state_dir="${1:-.inngest}"
duckdb_dir="${state_dir%/}/duckdb"
db_file="${duckdb_dir}/main.duckdb"
catalog_db="${duckdb_dir}/catalog.sqlite"
warehouse="${duckdb_dir}/warehouse/main.db"

if ! command -v duckdb >/dev/null 2>&1; then
    echo "error: duckdb binary not found on PATH" >&2
    exit 1
fi

if [[ ! -d "${warehouse}" ]]; then
    echo "error: ${warehouse} does not exist" >&2
    echo "       expected an existing Iceberg lake (run \`inngest dev --duckdb --persist\` at least once first)" >&2
    exit 1
fi

cmds=(-cmd "INSTALL iceberg;" -cmd "LOAD iceberg;" -cmd "CREATE SCHEMA inngest;")
add_view() {
    cmds+=(-cmd "CREATE VIEW inngest.${1} AS SELECT * FROM iceberg_scan('${2}');")
}
if [[ -f "${catalog_db}" ]]; then
    # <table>|<metadata_location> for every table in the `main` namespace.
    tables="$(duckdb -noheader -list -c "INSTALL sqlite; LOAD sqlite;
        SELECT table_name, metadata_location FROM sqlite_scan('${catalog_db}', 'iceberg_tables')
        WHERE catalog_name = 'inngest' AND table_namespace = 'main' ORDER BY table_name;")"
    while IFS='|' read -r table location; do
        [[ -n "${table}" ]] && add_view "${table}" "${location}"
    done <<<"${tables}"
else
    echo "warning: ${catalog_db} not found, using each table's highest metadata version on disk" >&2
    for table_dir in "${warehouse}"/*/; do
        # Metadata files are <version>-<uuid>.metadata.json, so sort on the
        # version numerically.
        latest="$(ls -1 "${table_dir}metadata/" 2>/dev/null | { grep '\.metadata\.json$' || true; } | sort -t- -k1,1n | tail -1)"
        if [[ -n "${latest}" ]]; then
            add_view "$(basename "${table_dir}")" "${table_dir}metadata/${latest}"
        fi
    done
fi
if [[ -f "${db_file}" ]]; then
    cmds+=(-cmd "ATTACH '${db_file}' AS local (READ_ONLY);")
fi

echo "lake: ${warehouse} (read-only views under 'inngest')" >&2
exec duckdb "${cmds[@]}"
