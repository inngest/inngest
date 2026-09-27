#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
report=${1:-"$root/.amp/in/artifacts/sdk-conformance-report.json"}
typescript_port=${TYPESCRIPT_PORT:-3101}
go_port=${GO_PORT:-3102}
python_port=${PYTHON_PORT:-3103}
signing_key=${INNGEST_SIGNING_KEY:-7468697320697320612074657374206b6579}
signing_key_fallback=${INNGEST_SIGNING_KEY_FALLBACK:-66616c6c6261636b20636f6e666f726d616e6365}
pids=()

cleanup() {
  for pid in "${pids[@]}"; do
    kill -- "-$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

mkdir -p "$(dirname "$report")" "$root/.amp/in"

if [[ ! -x "$root/sdk-conformance/fixtures/typescript/node_modules/.bin/tsx" ]]; then
  npm --prefix "$root/sdk-conformance/fixtures/typescript" ci
fi
uv sync --frozen --project "$root/sdk-conformance/fixtures/python"

setsid env PORT=$typescript_port INNGEST_SIGNING_KEY=$signing_key \
  INNGEST_SIGNING_KEY_FALLBACK=$signing_key_fallback \
  "$root/sdk-conformance/fixtures/typescript/node_modules/.bin/tsx" \
  "$root/sdk-conformance/fixtures/typescript/src/server.ts" \
  >"$root/.amp/in/typescript-fixture.log" 2>&1 &
pids+=("$!")

(cd "$root" && exec setsid env PORT=$go_port INNGEST_SIGNING_KEY=$signing_key \
  INNGEST_SIGNING_KEY_FALLBACK=$signing_key_fallback go run ./sdk-conformance/fixtures/go) \
  >"$root/.amp/in/go-fixture.log" 2>&1 &
pids+=("$!")

setsid env PORT=$python_port INNGEST_SIGNING_KEY=$signing_key \
  INNGEST_SIGNING_KEY_FALLBACK=$signing_key_fallback \
  uv run --project "$root/sdk-conformance/fixtures/python" \
  uvicorn app:app --app-dir "$root/sdk-conformance/fixtures/python" \
  --host 127.0.0.1 --port "$python_port" \
  >"$root/.amp/in/python-fixture.log" 2>&1 &
pids+=("$!")

for port in "$typescript_port" "$go_port" "$python_port"; do
  for _ in {1..60}; do
    if curl --silent --fail "http://127.0.0.1:$port/__conformance" >/dev/null; then
      break
    fi
    sleep 0.25
  done
  curl --silent --fail "http://127.0.0.1:$port/__conformance" >/dev/null
done

(cd "$root" && go run ./sdk-conformance/matrix \
  --target "typescript=http://127.0.0.1:$typescript_port" \
  --target "go=http://127.0.0.1:$go_port" \
  --target "python=http://127.0.0.1:$python_port" \
  --reference typescript \
  --probes "$root/sdk-conformance/probes" \
  --output "$report")

printf 'Full report: %s\n' "$report"
