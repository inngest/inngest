#!/usr/bin/env bash
# Poll an HTTP endpoint until it returns success, failing loudly if it never does.
# Replaces blind `sleep N` waits in CI so a server that never booted fails fast
# and attributably instead of letting tests run against a dead server.
#
# Usage: wait-for-healthy.sh <name> <url> [max_seconds]
set -euo pipefail

name="$1"
url="$2"
tries="${3:-10}"

for i in $(seq 1 "$tries"); do
  if curl -fsS "$url" >/dev/null 2>&1; then
    echo "$name is ready (after ${i}s): $url"
    exit 0
  fi
  sleep 1
done

echo "ERROR: $name failed to become healthy after ${tries}s: $url" >&2
exit 1
