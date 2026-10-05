package federate

import (
	"context"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const metricsPkg = "federate"

// deltaRowBoundaries spans an empty delta up through the row caps a
// federated query sets.
var deltaRowBoundaries = []float64{0, 1, 10, 100, 1_000, 10_000, 100_000, 1_000_000}

// deltaKind is what a delta stream carries: the buffer's rows, or its
// partial aggregates (one row per group).
type deltaKind string

const (
	deltaRows     deltaKind = "rows"
	deltaPartials deltaKind = "partials"
)

// deltaCount counts one delta stream's rows against its cap and records the
// total once the stream ends.
type deltaCount struct {
	table    Table
	kind     deltaKind
	rowCap   int
	total    int
	recorded bool
}

// add counts n more rows, failing with ErrRowCapExceeded past the cap.
func (c *deltaCount) add(ctx context.Context, n int) error {
	c.total += n
	if c.rowCap > 0 && c.total > c.rowCap {
		recordRowCapExceeded(ctx, c.table, c.kind)
		return ErrRowCapExceeded
	}
	return nil
}

// recordRowCapExceeded counts a federated read failed by a delta over its
// row cap.
func recordRowCapExceeded(ctx context.Context, t Table, kind deltaKind) {
	metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
		PkgName:     metricsPkg,
		MetricName:  "delta_row_cap_exceeded_total",
		Description: "Federated reads failed because a buffer delta exceeded its row cap",
		Tags:        map[string]any{"table": string(t), "kind": string(kind)},
	})
}

// done records the stream's total; later calls do nothing.
func (c *deltaCount) done(ctx context.Context) {
	if c.recorded {
		return
	}
	c.recorded = true
	metrics.RecordIntHistogramMetric(ctx, int64(c.total), metrics.HistogramOpt{
		PkgName:     metricsPkg,
		MetricName:  "delta_rows",
		Description: "Rows a federated read streamed from the buffer for one table",
		Tags:        map[string]any{"table": string(c.table), "kind": string(c.kind)},
		Unit:        "rows",
		Boundaries:  deltaRowBoundaries,
	})
}

// recordAggregateFallback counts an aggregate the buffer couldn't compute
// exactly (ErrNotExact), so its delta's rows were aggregated in DuckDB.
func recordAggregateFallback(ctx context.Context, t Table) {
	metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
		PkgName:     metricsPkg,
		MetricName:  "aggregate_fallback_total",
		Description: "Federated aggregates whose filter the buffer couldn't apply exactly, aggregated over the delta's rows instead",
		Tags:        map[string]any{"table": string(t)},
	})
}
