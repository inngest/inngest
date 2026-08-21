package batch

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const (
	batchMetricsPkgName = "batch"
	batchBackendDefault = "default"

	// account_tier intentionally uses the enterprise|other|unknown vocabulary.
	// It is distinct from executor's account_plan tag, whose domain is
	// free|self_serve|enterprise|unknown.
	batchTierEnterprise = "enterprise"
	batchTierOther      = "other"
	batchTierUnknown    = "unknown"
)

// AccountPlanMetricTagResolver returns an account's plan name for metrics only.
// Values are normalized to enterprise, other, or unknown before being emitted.
// The resolver is called synchronously while flushing buffered writes, so it
// must use local data only and must not block or perform network I/O.
type AccountPlanMetricTagResolver func(ctx context.Context, accountID uuid.UUID) string

// WithAccountPlanMetricTagResolver adds a low-cardinality account tier tag to
// batch metrics. Resolver failures should return an empty string.
func WithAccountPlanMetricTagResolver(resolver AccountPlanMetricTagResolver) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.accountPlanMetricTagResolver = resolver
	}
}

// WithBatchMetricBackend identifies the logical storage backend in batch metrics.
func WithBatchMetricBackend(backend string) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		if backend != "" {
			m.metricBackend = backend
		}
	}
}

func (b *redisBatchManager) accountTierMetricTag(ctx context.Context, accountID uuid.UUID) string {
	if b.accountPlanMetricTagResolver == nil || accountID == uuid.Nil {
		return batchTierUnknown
	}
	plan := b.accountPlanMetricTagResolver(ctx, accountID)
	if plan == "" {
		return batchTierUnknown
	}
	if plan == batchTierEnterprise {
		return batchTierEnterprise
	}
	return batchTierOther
}

func batchStorageMetricTags(accountTier, backend string) map[string]any {
	return map[string]any{
		"account_tier": accountTier,
		"backend":      backend,
	}
}

func withBatchMetricTags(base, extra map[string]any) map[string]any {
	tags := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		tags[key] = value
	}
	for key, value := range extra {
		tags[key] = value
	}
	return tags
}

func recordBatchCommitMetrics(ctx context.Context, accountTier, backend string, committedBytes int64) {
	tags := batchStorageMetricTags(accountTier, backend)
	metrics.HistogramBatchPayloadSizeBytes(ctx, committedBytes, metrics.HistogramOpt{PkgName: batchMetricsPkgName, Tags: tags})
}

func recordBatchListObservation(ctx context.Context, accountTier, backend string, listResidentBytes, listItemCount int64) {
	tags := batchStorageMetricTags(accountTier, backend)
	metrics.HistogramBatchListResidentSizeBytes(ctx, listResidentBytes, metrics.HistogramOpt{PkgName: batchMetricsPkgName, Tags: tags})
	metrics.HistogramBatchListItemCount(ctx, listItemCount, metrics.HistogramOpt{PkgName: batchMetricsPkgName, Tags: tags})
}

func shouldRecordBatchListObservation(status string) bool {
	return status != "overflow"
}

func recordBatchRetrieveMetrics(ctx context.Context, backend, status string, itemCount int64, started time.Time) {
	tags := map[string]any{"backend": backend, "status": status}
	metrics.HistogramBatchRetrieveItemsDuration(ctx, time.Since(started), metrics.HistogramOpt{PkgName: batchMetricsPkgName, Tags: tags})
	if status == "success" {
		metrics.HistogramBatchRetrievedItemCount(ctx, itemCount, metrics.HistogramOpt{PkgName: batchMetricsPkgName, Tags: tags})
	}
}

func recordBatchDeleteMetrics(ctx context.Context, backend, status string, started time.Time) {
	tags := map[string]any{"backend": backend, "status": status}
	metrics.HistogramBatchDeleteKeysDuration(ctx, time.Since(started), metrics.HistogramOpt{PkgName: batchMetricsPkgName, Tags: tags})
}
