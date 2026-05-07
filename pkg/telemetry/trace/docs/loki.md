# Spans as logs (Grafana Loki integration)

Inngest can emit OpenTelemetry `LogRecord`s for run starts, step endings, and
function endings, then ship them over OTLP/HTTP to any logs-capable backend.
Pointed at an OTel Collector that fronts Grafana Loki, this gives a logs-only
path for exploring run history without standing up a separate trace store
(Tempo, Jaeger).

## Pipeline

```
Inngest userTracer
   └── BatchSpanProcessor ─► OTLP traces (existing path; unchanged)
   └── SpansAsLogsProcessor (filter ▸ JSON body ▸ severity)
            │
            ▼
       LoggerProvider ─► OTLP/HTTP logs ─► OTel Collector ─► Loki
```

The processor is purely additive. When `INNGEST_OTEL_LOGS_ENDPOINT` is unset
it is not registered and adds zero overhead. The system tracer
(`tracing-system`) does not get the side pipeline — only run-level user spans
are emitted as logs.

## Enabling

Set on the Inngest server process (`inngest start` or `inngest dev`):

| Env var | Default | Effect |
| --- | --- | --- |
| `INNGEST_OTEL_LOGS_ENDPOINT` | _(unset → off)_ | `host:port` of the OTLP/HTTP logs receiver. Presence of this var enables the side pipeline. |
| `INNGEST_OTEL_LOGS_URL_PATH` | `/v1/logs` | Path on the receiver. |
| `INNGEST_OTEL_LOGS_PAYLOAD_MAX_BYTES` | `4096` | Per-attribute byte cap for `sys.step.input`, `sys.step.output`, `sys.function.output`, `sys.step.ai.req`, `sys.step.ai.res`. |

Example:

```sh
INNGEST_OTEL_LOGS_ENDPOINT=otel-collector.monitoring:4318 \
INNGEST_OTEL_LOGS_PAYLOAD_MAX_BYTES=8192 \
inngest dev
```

The transport is OTLP/HTTP with `WithInsecure()` — run the collector on a
private network or front it with a reverse proxy that adds TLS.

## What ends up in each log record

- **Timestamp** = span end time. **ObservedTimestamp** = emit time.
- **Severity** is derived from the span attributes:
  - `sys.function.status.code` (root function spans), in priority order:
    `Completed → INFO`, `Failed`/`Overflowed → ERROR`, `Cancelled → WARN`,
    `Skipped → DEBUG`.
  - `sys.step.status` (step spans): `Completed → INFO`,
    `Failed`/`Errored → ERROR`, `Cancelled`/`TimedOut → WARN`,
    `Skipped → DEBUG`.
  - Otherwise OTel span status code (`Error → ERROR`), defaulting to `INFO`.
- **Body** is JSON, containing every kept span attribute plus
  `inngest.log.type`, `span.name`, `span.kind`, `span.scope`,
  `span.duration_ms`, `span.trace_id`, `span.span_id`, and
  `span.parent_span_id` (if any).
- **Run event data** is included only on `run.started` records as `sys.event`.
  Step and function-ending records keep event IDs but do not duplicate the full
  triggering event payloads.
- **Step/function payloads** (`sys.step.input`, `sys.step.output`,
  `sys.function.output`, `sys.step.ai.req`, `sys.step.ai.res`) are extracted
  from span events and capped by `INNGEST_OTEL_LOGS_PAYLOAD_MAX_BYTES`.
- Derived fields and span attributes are also attached as OTLP `LogRecord`
  attributes so backends that promote attributes to indexed labels can do so
  without parsing the body.

## Emitted records

Only these lifecycle records are emitted as logs:

- `run.started` — function scope with `sys.lifecycle.id=OnFunctionStarted`.
- `function.ended` — function scope with finished, cancelled, or skipped
  lifecycle IDs.
- `step.ended` — user-visible regular steps, gateway/AI gateway steps, and
  durable sleep/wait/invoke/signal completion spans.

Trigger / event / cron / batch / debounce / replay / rerun spans, planned step
spans, step-start spans, and arbitrary userland spans are dropped from the logs
pipeline (still flow through the trace pipeline normally).

## Example OTel Collector pipeline (Loki)

```yaml
receivers:
  otlp:
    protocols:
      http: { endpoint: 0.0.0.0:4318 }

processors:
  batch: {}
  attributes/loki_labels:
    actions:
      - key: loki.attribute.labels
        action: insert
        value: inngest.log.type,sys.account.id,sys.workspace.id,sys.app.id,sys.function.id,sys.function.slug,sys.function.status.code,sys.step.status,sys.step.opcode,sys.step.run.type,service.name
      - key: loki.resource.labels
        action: insert
        value: service.name

exporters:
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

service:
  pipelines:
    logs:
      receivers:  [otlp]
      processors: [attributes/loki_labels, batch]
      exporters:  [loki]
```

## Loki label cardinality guidance

Promote only **bounded** attributes to labels. The processor emits all kept
attributes; the operator chooses which ones become indexed.

**Safe to promote** (low/bounded cardinality):
`sys.account.id`, `sys.workspace.id`, `sys.app.id`, `sys.function.id`,
`sys.function.slug`, `sys.function.status.code`, `sys.step.status`,
`sys.step.opcode`, `sys.step.run.type`, `inngest.log.type`, `service.name`.

**Borderline** (depends on your scale):
`sys.function.version`, `sys.step.attempt`, `sys.step.attempt.max`.

**Never promote** (high or unbounded cardinality — keep as log-line fields):
`sdk.run.id`, `sys.lifecycle.id`, `sys.idempotency.key`,
`sys.event.internal.id`, `sys.event.request.id`, `sys.event.ids`,
`sys.batch.id`, `sys.debounce.id`, `sys.step.id`, `sys.step.group.id`,
`sys.step.stack`, `sys.step.input`, `sys.step.output`,
`sys.function.output`, `sys.step.ai.req`, `sys.step.ai.res`, all
`sys.*.time.*` timestamps.

## LogQL examples (Grafana Explore)

Loki replaces `.` with `_` in promoted label names.

```logql
# All logs from the user tracer
{service_name="tracing"}

# Failed runs for a specific function (label-only filter, fast)
{service_name="tracing", inngest_log_type="function.ended", sys_function_slug="my-app/my-fn"} | json | severity="ERROR"

# Step errors across the fleet
{service_name="tracing", inngest_log_type="step.ended"} | json | severity="ERROR"

# Runs triggered by event payload data
{service_name="tracing", inngest_log_type="run.started"} | json | sys_event =~ ".*a@example.com.*"

# 5xx-equivalent rate by function (logs-as-metrics)
sum by (sys_function_slug) (
  rate({service_name="tracing"} | json | severity="ERROR" [5m])
)
```

## Troubleshooting

- **No log lines arriving** — check the collector is reachable on the path
  configured via `INNGEST_OTEL_LOGS_URL_PATH` (default `/v1/logs`); the
  underlying exporter uses `WithInsecure()`.
- **Bodies too large in Loki** — lower
  `INNGEST_OTEL_LOGS_PAYLOAD_MAX_BYTES`. The truncation suffix records the
  original size so you can tell when it's hitting the cap.
- **Cardinality explosion** — narrow the `loki.attribute.labels` allowlist;
  in particular do not promote `sdk.run.id` or any `sys.event.*`/`sys.step.id`.
- **Severity always INFO** — confirm spans carry `sys.function.status.code`,
  `sys.step.status`, or an OTel error status.

## Testing locally

An automated end-to-end test under [`tests/loki/`](../../../../tests/loki/)
boots a Grafana LGTM container (Loki + Tempo + Grafana, all in one),
wires the production `SpansAsLogsProcessor` to its OTLP/HTTP endpoint,
emits synthetic spans for each scenario, and asserts the results via
Loki's HTTP query API. Run it with:

```sh
go test -tags=e2e_loki -v -timeout=120s ./tests/loki/...
```

Requires Docker. The build tag keeps the suite out of the default
`go test ./...` run.

## Non-goals

- Direct Loki push (no `lokiexporter` in Inngest itself — go via collector).
- Trace waterfalls: this is a logs-only path. Use Tempo/Jaeger via the
  existing OTLP traces pipeline if you need waterfall visualization.
- Run lifecycle counters (run started / finished as metrics) — out of scope
  for this MVP.
