# Inngest load-testing harness

A standalone tool that measures how an Inngest server (dev or otherwise)
behaves under load. It does not know anything about Inngest internals — it
targets a running server over HTTP, fires events, and collects per-step
timings via subprocess workers that run the real SDK.

## Layout

```
loadtest/
  cmd/harness/          # main binary: UI, SQLite, orchestration
  cmd/worker-go/        # Go SDK worker (one subprocess per app under test)
  internal/config/      # shared config schema
  internal/telemetry/   # unix-socket wire protocol + ring buffer
  internal/shapes/      # function-shape templates
  internal/firer/       # batched event POSTs with rate limiting
  internal/runner/      # run orchestration: spawn, fire, tear down, aggregate
  internal/storage/     # SQLite persistence
  internal/metrics/     # percentile math
  internal/api/         # JSON REST handlers
  internal/uiembed/     # embed.FS wrapper for the built SPA
  ui/                   # Vite + TanStack Router SPA; builds into uiembed/dist
```

Isolated Go module (`loadtest/go.mod`) so the harness never imports Inngest
internal packages.

## Build & run

```bash
# Build everything (UI + Go binaries)
cd loadtest
make all

# Assume `inngest dev` is already running at http://127.0.0.1:8288
make run
# open http://127.0.0.1:9010
```

Under the hood `make run` does:

```bash
./bin/harness \
  --worker ./bin/worker-go \
  --db ./loadtest.db \
  --port 9010
```

## Metrics

| Metric | Definition |
|--------|------------|
| event → run | first `fn_start` ts − event `sent_at` ts |
| inter-step | next `step_start` ts − previous `step_end` ts (includes checkpoint + queue + dispatch) |
| step duration | `step_end` − `step_start` |
| SDK overhead | `sdk_request_recv` − previous `sdk_response_sent` (v1 placeholder) |

Checkpoint latency is not separately measurable externally; inter-step is the
headline proxy and the UI labels it accordingly.

## Subprocess contract (language-agnostic)

A TS worker can slot in later by honoring the same contract:

- **stdin**: one JSON object of shape `config.WorkerConfig`
- **telemetry**: connect to the `telemetrySocket` unix socket, send
  length-prefixed JSON frames (4-byte big-endian length header, then the JSON
  body of a `telemetry.Frame`)
- **lifecycle**: emit `phase: "ready"` once registration has succeeded
- **shutdown**: graceful on SIGTERM/SIGINT

## Tests

```bash
go test ./...
```

Covers telemetry frame encode/decode, ring-buffer drop-oldest semantics,
percentile math, and SQLite round-trip.

## Limitations (v1)

- Dev-server miniredis backend is not representative of production latencies.
  The UI shows a notice on the configure page.
- Durable Endpoints not exercised.
- TS SDK workers not yet implemented (contract is reserved).
- Single host only (no cross-host clock-skew handling).
