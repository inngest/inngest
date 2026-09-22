#!/usr/bin/env bash
#
# Runs the DuckDB/quack smoke set against a real duckdb binary and fails if
# any selected test skips, or if a pattern matches no tests at all.
#
# The unit test job runs these same tests, but most of them skip when no
# duckdb binary (or quack extension) is available, so that job can pass
# without the real stack ever executing. This script is the guard against
# that: it expects `duckdb` on PATH (see `inngest duckdb download`).
#
# Usage: scripts/duckdb-smoke.sh [extra go test flags...]

set -euo pipefail

command -v duckdb >/dev/null || {
	echo "duckdb not found on PATH; run 'go run ./cmd duckdb download' and add the binary's directory to PATH" >&2
	exit 1
}
command -v jq >/dev/null || {
	echo "jq is required" >&2
	exit 1
}

# package => -run pattern. Covers process close/restart, pooled-connection
# restart recovery, quack append, migrations (incl. persistent reopen),
# listener shutdown flush, dual-write -> query round trips, and the
# duckdbseed appender against the real schema (catches column drift).
declare -A SMOKE=(
	[./pkg/duckdb/driver]='^(TestConnectorCloseTerminatesSubprocess|TestExecTriggersRestartAfterCrash|TestQuackTransportSurvivesRestart|TestPooledQuackConn.*|TestQuackAppenderWritesRowsIntoRealTable|TestQuackBootstrapErrorDoesNotLeakToken)$'
	[./pkg/duckdb/driver/internal/quack]='^TestQuackSession(HandshakeAndExec|FetchesAllRowsWhenResultExceedsOneInlineResponse|StatementErrorMapsToErrStatementFailed)$'
	[./pkg/db/duckdb]='^TestMigrate'
	[./pkg/duckdb/tracing]='^(TestNewListenerEndToEnd.*|TestBatcherDrainsChannelOnStopBeforeExiting)$'
	[./pkg/duckdb/query]='^TestDualWriteThenDuckDBQueryRoundTrip$'
	[./cmd/duckdbseed]='^(TestInsertGeneratedRunsWritesRunsSpansAndEvents|TestSampleTemplatesReadsRealRowsAsTemplates)$'
	[./pkg/devserver]='^(TestSetupDualWriteReturnsListenerWhenBinaryPresent|TestDualWriteEndToEnd.*|TestStopDualWriteStopsBatcherGoroutines|TestSetupDualWriteTwoInstancesDoNotCollideOnQuackPort)$'
)

out="$(mktemp)"
trap 'rm -f "$out"' EXIT

status=0
for pkg in "${!SMOKE[@]}"; do
	pattern="${SMOKE[$pkg]}"
	echo "==> $pkg -run '$pattern'"
	if ! go test -json -count=1 -run "$pattern" "$@" "$pkg" >"$out"; then
		status=1
	fi

	jq -r 'select(.Test != null and (.Action == "pass" or .Action == "fail" or .Action == "skip")) | "    \(.Action)\t\(.Test)"' "$out"

	ran="$(jq -s '[.[] | select(.Test != null and (.Action == "pass" or .Action == "fail"))] | length' "$out")"
	skipped="$(jq -r 'select(.Test != null and .Action == "skip") | .Test' "$out")"
	if [[ "$ran" -eq 0 ]]; then
		echo "    no tests matched in $pkg; the smoke pattern is stale" >&2
		status=1
	fi
	if [[ -n "$skipped" ]]; then
		echo "    required smoke tests skipped in $pkg:" >&2
		# Show why: the skip message is in the test's output lines.
		jq -r 'select(.Test != null and .Action == "output" and (.Output | test("SKIP|skipping"))) | "      \(.Output)"' "$out" >&2
		status=1
	fi
	if jq -e 'select(.Action == "fail")' "$out" >/dev/null; then
		jq -r 'select(.Action == "output") | .Output' "$out" | tail -60 >&2
	fi
done

exit "$status"
