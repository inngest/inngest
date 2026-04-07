#!/bin/bash
# check.sh — Orchestrate upstream compatibility checks for CI.
#
# Runs classify, ifacecheck, and compile checks against monorepo/ using
# the current inngest/ PR branch. All raw tool output is captured to temp
# files and never printed directly — only sanitized output reaches the
# markdown report.
#
# Required env vars:
#   INNGEST_DIR   — path to inngest/ checkout
#   MONOREPO_DIR  — path to monorepo/ checkout
#
# Always exits 0 — findings are informational only.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPORT="$SCRIPT_DIR/upstream-report.md"
TMPDIR_CHECK=$(mktemp -d)
REDACT_PREFIX="$MONOREPO_DIR"

cleanup() {
    rm -rf "$TMPDIR_CHECK"
}
trap cleanup EXIT

# ── Helpers ──────────────────────────────────────────────────────────

# sanitize_line strips monorepo paths and replaces them with [redacted].
sanitize_line() {
    local line="$1"
    # Strip absolute monorepo path prefix
    line="${line//$REDACT_PREFIX/[redacted]}"
    # Strip relative monorepo references
    line=$(echo "$line" | sed 's|monorepo/[^ ]*|[redacted]|g')
    echo "$line"
}

# sanitize_build_errors processes go build stderr and extracts only
# error descriptions that reference inngest/ types.
sanitize_build_errors() {
    local errfile="$1"
    local count=0
    local inngest_errors=0
    local output=""

    while IFS= read -r line; do
        count=$((count + 1))
        # go build errors look like: /path/to/file.go:42:15: error description
        # Extract the error description (everything after the last column number colon)
        local desc
        desc=$(echo "$line" | sed -E 's|^[^:]+:[0-9]+:[0-9]+: ||')
        if [ "$desc" = "$line" ]; then
            # Didn't match the pattern — try simpler pattern
            desc=$(echo "$line" | sed -E 's|^[^:]+: ||')
        fi

        # Only show errors referencing inngest/ types
        if echo "$desc" | grep -q 'github.com/inngest/inngest'; then
            inngest_errors=$((inngest_errors + 1))
            # Clean up the description — redact any remaining monorepo paths
            desc=$(sanitize_line "$desc")
            output="${output}Compile error: ${desc}\n"
        fi
    done < "$errfile"

    if [ "$count" -eq 0 ]; then
        echo "No compile errors."
    else
        echo "$count compile errors total, $inngest_errors referencing inngest/ types."
        if [ -n "$output" ]; then
            echo ""
            echo -e "$output"
        fi
    fi
}

# ── Step 1: Classify ────────────────────────────────────────────────

echo "Running classify..."
CLASSIFY_EXIT=0
go run -C "$SCRIPT_DIR" ./classify \
    --ci \
    --inngest="$INNGEST_DIR" \
    --monorepo="$MONOREPO_DIR" \
    > "$TMPDIR_CHECK/classify-raw.txt" 2>&1 || CLASSIFY_EXIT=$?

CLASSIFY_LABEL="SAFE"
CLASSIFY_EMOJI="✅"
if [ "$CLASSIFY_EXIT" -eq 1 ]; then
    CLASSIFY_LABEL="ADDITIVE"
    CLASSIFY_EMOJI="➕"
elif [ "$CLASSIFY_EXIT" -ge 2 ]; then
    CLASSIFY_LABEL="BREAKING"
    CLASSIFY_EMOJI="🔴"
fi

# ── Step 2: Interface check ─────────────────────────────────────────

echo "Running ifacecheck..."
IFACE_EXIT=0
go run -C "$SCRIPT_DIR" ./ifacecheck \
    --ci \
    --new="$INNGEST_DIR" \
    --monorepo="$MONOREPO_DIR" \
    > "$TMPDIR_CHECK/iface-raw.txt" 2>&1 || IFACE_EXIT=$?

# ── Step 3: Compile check ───────────────────────────────────────────

echo "Running compile check..."
WORK_FILE="$TMPDIR_CHECK/go.work"
INNGEST_REL=$(python3 -c "import os.path; print(os.path.relpath('$INNGEST_DIR', '$MONOREPO_DIR'))")
GO_VERSION=$(grep '^go ' "$MONOREPO_DIR/go.mod" | head -1 | awk '{print $2}')

cat > "$WORK_FILE" <<EOF
go ${GO_VERSION}

use (
	.
	${INNGEST_REL}
)
EOF

# Get package list (exclude test/integration)
PKGS=$(cd "$MONOREPO_DIR" && GOWORK="$WORK_FILE" go list ./... 2>/dev/null | grep -v '/test/integration' || true)

BUILD_OK=true
if [ -n "$PKGS" ]; then
    GOWORK="$WORK_FILE" go build -C "$MONOREPO_DIR" $PKGS 2> "$TMPDIR_CHECK/build-errors.txt" || BUILD_OK=false
else
    echo "No packages found to build." > "$TMPDIR_CHECK/build-errors.txt"
    BUILD_OK=false
fi

# ── Step 4: Determine overall classification ─────────────────────────

OVERALL_LABEL="$CLASSIFY_LABEL"
OVERALL_EMOJI="$CLASSIFY_EMOJI"

# Escalate if interface check found breaking changes
if [ "$IFACE_EXIT" -ge 2 ] && [ "$OVERALL_LABEL" != "BREAKING" ]; then
    OVERALL_LABEL="BREAKING"
    OVERALL_EMOJI="🔴"
fi

# Escalate if compile check failed
if [ "$BUILD_OK" = false ] && [ "$OVERALL_LABEL" = "SAFE" ]; then
    OVERALL_LABEL="BREAKING"
    OVERALL_EMOJI="🔴"
elif [ "$BUILD_OK" = false ] && [ "$OVERALL_LABEL" = "ADDITIVE" ]; then
    OVERALL_LABEL="BREAKING"
    OVERALL_EMOJI="🔴"
fi

# ── Step 5: Write report ────────────────────────────────────────────

echo "Writing report..."

cat > "$REPORT" <<EOF
## Upstream Compatibility: ${OVERALL_LABEL} ${OVERALL_EMOJI}

### Symbol Changes
\`\`\`
$(cat "$TMPDIR_CHECK/classify-raw.txt")
\`\`\`

### Interface Changes
\`\`\`
$(cat "$TMPDIR_CHECK/iface-raw.txt")
\`\`\`

### Compile Check
EOF

if [ "$BUILD_OK" = true ]; then
    echo '```' >> "$REPORT"
    echo "PASS: downstream compiles cleanly against this branch." >> "$REPORT"
    echo '```' >> "$REPORT"
else
    echo '```' >> "$REPORT"
    sanitize_build_errors "$TMPDIR_CHECK/build-errors.txt" >> "$REPORT"
    echo '```' >> "$REPORT"
fi

cat >> "$REPORT" <<'EOF'

---
> Checks whether this PR's changes are compatible with downstream consumers.
> Breaking changes will need corresponding downstream updates before the next vendor cycle.
EOF

# ── Step 6: Final redaction guard ────────────────────────────────────

# Strip any remaining monorepo path references that slipped through
sed -i.bak "s|${REDACT_PREFIX}|[redacted]|g" "$REPORT" 2>/dev/null || true
sed -i.bak 's|monorepo/[^ ]*|[redacted]|g' "$REPORT" 2>/dev/null || true
rm -f "$REPORT.bak"

echo "Report written to $REPORT"
echo "Overall: $OVERALL_LABEL"

# Always exit 0 — this check is informational only
exit 0
