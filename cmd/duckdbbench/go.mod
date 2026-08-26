// Command duckdbbench benchmarks INSERT throughput into inngest.run_trace_spans
// (pkg/db/duckdb/migrations/000001_baseline.sql) across three DuckDB access
// paths: the -jsonlines subprocess transport, the quack HTTP transport (both
// in pkg/db/duckdb), and the official github.com/duckdb/duckdb-go/v2 cgo
// bindings driving an embedded, in-process DuckDB with no subprocess at all.
//
// This is a separate Go module, not part of the root github.com/inngest/inngest
// module, specifically so duckdb-go/v2's per-platform static-library
// dependencies never enter the root module's committed vendor/ directory —
// see docs/plans/006-duckdb-poc-subprocess-dual-write.md's "Alternatives
// Considered" section on why a cgo DuckDB dependency was kept out of the main
// binary. `replace` points back at the repo root so this module can still use
// pkg/db/duckdb's exported API (Open, Migrate, Options, DuckLakeOptions,
// DuckLakeAlias) directly, without vendoring or publishing anything.
//
// Run with: cd cmd/duckdbbench && go test -bench=. -run='^$' ./...
module github.com/inngest/inngest/cmd/duckdbbench

go 1.26.4

require (
	github.com/duckdb/duckdb-go/v2 v2.10505.0
	github.com/google/uuid v1.6.0
	github.com/inngest/inngest v0.0.0-00010101000000-000000000000
)

require (
	github.com/apache/arrow-go/v18 v18.5.1 // indirect
	github.com/duckdb/duckdb-go-bindings v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-arm64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-amd64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-arm64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/windows-amd64 v0.10505.0 // indirect
	github.com/getsentry/sentry-go v0.27.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/lmittmann/tint v1.1.0 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pressly/goose/v3 v3.27.0 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20260218203240-3dfff04db8fa // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/telemetry v0.0.0-20260508192327-42602be52be6 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
)

replace github.com/inngest/inngest => ../..
