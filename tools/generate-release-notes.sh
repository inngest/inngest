#!/usr/bin/env bash
#
# generate-release-notes.sh
#
# Generates user-facing release notes for the Inngest repo by:
# 1. Fetching commit delta between two tags
# 2. Classifying each commit as user-facing or internal based on files touched
# 3. Outputting categorized markdown
#
# Usage:
#   ./generate-release-notes.sh <prev_tag> <new_tag> [repo_path]
#
# Example:
#   ./generate-release-notes.sh v1.17.0 v1.17.1 ../inngest

set -euo pipefail

PREV_TAG="${1:?Usage: $0 <prev_tag> <new_tag> [repo_path]}"
NEW_TAG="${2:?Usage: $0 <prev_tag> <new_tag> [repo_path]}"
REPO_PATH="${3:-../inngest}"

cd "$REPO_PATH"

# ─── Directory classification rules ──────────────────────────────────────────
#
# Exclude patterns are checked FIRST and take priority over includes.
# This allows broad include patterns (e.g. pkg/execution/) while carving out
# specific subdirectories that are cloud/internal only.

EXCLUDE_PATTERNS=(
  # Cloud dashboard & support apps
  "ui/apps/dashboard/"
  "ui/apps/support/"
  # Cloud infrastructure within execution
  "pkg/execution/queue/shard_lease"
  # Capacity management (cloud)
  "pkg/constraintapi/"
  # Debug/ops tooling
  "pkg/debugapi/"
  "cmd/debug/"
  "proto/debug/"
  "proto/gen/debug/"
  # Shard lease tests
  "tests/execution/queue/queue_shard_lease"
  # Dashboard-only shared components
  "ui/packages/components/src/SQLEditor/"
)

INCLUDE_PATTERNS=(
  "ui/apps/dev-server-ui/"
  "pkg/devserver/"
  "pkg/execution/"
  "pkg/coreapi/"
  "cmd/devserver/"
  "cmd/start/"
  "cmd/main.go"
)

# Shared UI components — cross-reference with dev-server-ui to decide
SHARED_UI_PATTERN="ui/packages/components/"

# Commit message prefixes that indicate internal/cloud-only work
# SYS-* = internal cloud engineering tickets
MESSAGE_EXCLUDE_PATTERNS=(
  "SYS-"
)

# ─── Helpers ─────────────────────────────────────────────────────────────────

classify_commit() {
  local hash="$1"
  local message="$2"

  # Check message-level excludes first (e.g., SYS-* tickets)
  for pat in "${MESSAGE_EXCLUDE_PATTERNS[@]}"; do
    if [[ "$message" == $pat* ]]; then
      echo "internal"
      return
    fi
  done

  local files
  # Use diff-tree for clean file-only output (no stat noise)
  files=$(git diff-tree --no-commit-id --name-only -r "$hash")

  local include_count=0
  local exclude_count=0
  local total_count=0

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    total_count=$((total_count + 1))

    # Check explicit excludes first (higher priority)
    local is_excluded=false
    for pat in "${EXCLUDE_PATTERNS[@]}"; do
      if [[ "$file" == $pat* ]]; then
        is_excluded=true
        exclude_count=$((exclude_count + 1))
        break
      fi
    done
    [[ "$is_excluded" == true ]] && continue

    # Check includes
    for pat in "${INCLUDE_PATTERNS[@]}"; do
      if [[ "$file" == $pat* ]]; then
        include_count=$((include_count + 1))
        break
      fi
    done

    # Check shared UI components — cross-reference with dev-server-ui imports
    if [[ "$file" == $SHARED_UI_PATTERN* ]]; then
      # Get the specific component directory name (e.g., "CodeBlock" from "ui/packages/components/src/CodeBlock/...")
      local component_path
      component_path=$(echo "$file" | sed "s|${SHARED_UI_PATTERN}src/||" | cut -d'/' -f1)
      # Only include if dev-server-ui actually imports this component
      if grep -rq "components/src/${component_path}\|@inngest/components.*${component_path}" "ui/apps/dev-server-ui/src/" 2>/dev/null; then
        include_count=$((include_count + 1))
      fi
    fi
  done <<< "$files"

  # A commit is user-facing if it touches at least one include path
  # AND the majority of its changes aren't in excluded paths
  # (This catches commits that primarily add debug APIs but incidentally touch shared code)
  if [[ $include_count -gt 0 && $include_count -ge $exclude_count ]]; then
    echo "user-facing"
  else
    echo "internal"
  fi
}

categorize_commit() {
  local message="$1"
  local lower
  lower=$(echo "$message" | tr '[:upper:]' '[:lower:]')

  if [[ "$lower" == fix* ]] || [[ "$lower" == *": fix "* ]] || [[ "$lower" == *"fix "* && "$lower" != *"prefix"* ]]; then
    echo "fix"
  elif [[ "$lower" == *"always "* ]] || [[ "$lower" == *"check for"* ]] || [[ "$lower" == *"cleanup"* ]] || [[ "$lower" == *"clean up"* ]] || [[ "$lower" == *"correct"* ]] || [[ "$lower" == *"resolve"* ]]; then
    # Defensive/cleanup changes are typically bug fixes
    echo "fix"
  else
    echo "improvement"
  fi
}

# ─── Main ────────────────────────────────────────────────────────────────────

echo "# Analyzing commits between $PREV_TAG and $NEW_TAG"
echo ""

commits=$(git log "$PREV_TAG..$NEW_TAG" --oneline --no-merges)

improvements=()
fixes=()
excluded=()

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  hash=$(echo "$line" | awk '{print $1}')
  message=$(echo "$line" | cut -d' ' -f2-)

  classification=$(classify_commit "$hash" "$message")

  if [[ "$classification" == "user-facing" ]]; then
    category=$(categorize_commit "$message")

    if [[ "$category" == "fix" ]]; then
      fixes+=("- $message")
    else
      improvements+=("- $message")
    fi
  else
    excluded+=("- [$hash] $message")
  fi
done <<< "$commits"

# ─── Output ──────────────────────────────────────────────────────────────────

echo "## $NEW_TAG"
echo ""

if [[ ${#improvements[@]} -gt 0 ]]; then
  echo "### Improvements"
  echo ""
  for item in "${improvements[@]}"; do
    echo "$item"
  done
  echo ""
fi

if [[ ${#fixes[@]} -gt 0 ]]; then
  echo "### Bug Fixes"
  echo ""
  for item in "${fixes[@]}"; do
    echo "$item"
  done
  echo ""
fi

echo "---"
echo ""
echo "### Excluded (internal/cloud-only) — for review"
echo ""
if [[ ${#excluded[@]} -gt 0 ]]; then
  for item in "${excluded[@]}"; do
    echo "$item"
  done
else
  echo "_None_"
fi
