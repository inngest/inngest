# Loki end-to-end tests

End-to-end tests for the spans-as-logs side pipeline
(`pkg/telemetry/exporters/spans_as_logs.go`). Boots the
[grafana-lgtm](https://golang.testcontainers.org/modules/grafana-lgtm/)
testcontainer (Loki + Tempo + Grafana in one container), wires the production
`SpansAsLogsProcessor` to its OTLP/HTTP endpoint, emits real spans, and
asserts via Loki's HTTP query API.

```sh
go test -tags=e2e_loki -v -timeout=120s ./tests/loki/...
```

Requirements: Docker. The first run pulls `grafana/otel-lgtm:0.8.1` (~500 MB).

The `e2e_loki` build tag keeps the suite out of the default `go test ./...`
run.
