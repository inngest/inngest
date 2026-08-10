#!/usr/bin/env bash
#
# Refreshes the embedded LLM model pricing snapshot used by
# pkg/tracing/metadata/extractors for AI cost estimation.
#
# Source: https://github.com/BerriAI/litellm (MIT licensed outside enterprise/)

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest="${repo_root}/pkg/tracing/metadata/extractors/model_prices.json"
src_url="https://raw.githubusercontent.com/BerriAI/litellm/refs/heads/litellm_internal_staging/model_prices_and_context_window.json"

echo "Fetching ${src_url}"
curl -sSfL "${src_url}" -o "${dest}"

echo "Slimming ${dest} to only the fields we use"
(cd "${repo_root}" && go run ./tools/model-prices -in "${dest}")

echo "Run 'go test ./pkg/tracing/metadata/extractors/...' to verify the update."
