#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
report=${1:-"$root/.amp/in/artifacts/sdk-conformance-devserver-report.json"}
fixture_port=${GO_FIXTURE_PORT:-3132}
dev_port=${DEV_SERVER_PORT:-8288}
proxy_port=${PROXY_PORT:-3140}
pids=()

cleanup() {
  for pid in "${pids[@]}"; do
    kill -- "-$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

mkdir -p "$(dirname "$report")" "$root/.amp/in"

(cd "$root" && exec setsid env PORT=$fixture_port INNGEST_DEV="http://127.0.0.1:$dev_port" \
  go run ./sdk-conformance/fixtures/go) \
  >"$root/.amp/in/devserver-spike-fixture.log" 2>&1 &
pids+=("$!")

(cd "$root" && exec setsid go run ./cmd dev \
  --no-discovery --no-poll --port "$dev_port") \
  >"$root/.amp/in/devserver-spike-server.log" 2>&1 &
pids+=("$!")

for _ in {1..120}; do
  fixture_ready=false
  dev_ready=false
  if curl --silent --fail "http://127.0.0.1:$fixture_port/__conformance" >/dev/null; then
    fixture_ready=true
  fi
  if curl --silent --fail "http://127.0.0.1:$dev_port/" >/dev/null; then
    dev_ready=true
  fi
  if [[ $fixture_ready == true && $dev_ready == true ]]; then
    break
  fi
  sleep 0.5
done

curl --silent --fail "http://127.0.0.1:$fixture_port/__conformance" >/dev/null
curl --silent --fail "http://127.0.0.1:$dev_port/" >/dev/null

(cd "$root" && go run ./sdk-conformance/devserver \
  --sdk-target "http://127.0.0.1:$fixture_port" \
  --dev-server "http://127.0.0.1:$dev_port" \
  --listen "127.0.0.1:$proxy_port" \
  --output "$report")

printf 'Full report: %s\n' "$report"
