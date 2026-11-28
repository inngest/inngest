package insights

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// Query outcome statuses RecordQueryMetrics tags its counter with.
const (
	QueryStatusOK              = "ok"
	QueryStatusMeta            = "meta"
	QueryStatusMetaError       = "meta_error"
	QueryStatusValidationError = "validation_error"
	QueryStatusParseError      = "parse_error"
	QueryStatusExecutionError  = "execution_error"
	QueryStatusInternalError   = "internal_error"
)

// RecordQueryMetrics records one Insights query's outcome: a status-tagged
// count, an end-to-end duration (from start to now), and, for a successful
// query, how many rows it returned. It's shared by the REST and GQL
// endpoints so both report the same metrics; pkgName tags which one.
// primaryTable is tagged rather than the full table list to keep
// cardinality bounded to the logical tables, not their combinations; it's
// "" for a query that never got that far (a parse/early-validation
// failure).
func RecordQueryMetrics(ctx context.Context, pkgName string, start time.Time, status, primaryTable string, rowCount int) {
	tags := map[string]any{}
	if primaryTable != "" {
		tags["primary_table"] = primaryTable
	}
	metrics.IncrInsightsQueryCounter(ctx, status, metrics.CounterOpt{PkgName: pkgName, Tags: tags})
	metrics.HistogramInsightsQueryDuration(ctx, time.Since(start), metrics.HistogramOpt{PkgName: pkgName, Tags: tags})
	if status == QueryStatusOK {
		metrics.HistogramInsightsQueryRowCount(ctx, int64(rowCount), metrics.HistogramOpt{PkgName: pkgName, Tags: tags})
	}
}
