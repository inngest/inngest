package queue

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var hintMetricReader = sdkmetric.NewManualReader()

func TestMain(m *testing.M) {
	// Instruments are cached globally; install the reader before any test emits.
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(hintMetricReader)))
	os.Exit(m.Run())
}

func hintMetrics(t *testing.T) (uint64, int64, map[string]int64) {
	t.Helper()
	var data metricdata.ResourceMetrics
	require.NoError(t, hintMetricReader.Collect(t.Context(), &data))
	var count uint64
	var sum int64
	outcomes := map[string]int64{}
	for _, scope := range data.ScopeMetrics {
		for _, metric := range scope.Metrics {
			switch metric.Name {
			case "inngest_fast_path_execution_latency":
				require.Equal(t, "ms", metric.Unit)
				for _, point := range metric.Data.(metricdata.Histogram[int64]).DataPoints {
					require.Equal(t, 2, point.Attributes.Len())
					_, ok := point.Attributes.Value("queue_shard")
					require.True(t, ok)
					backend, ok := point.Attributes.Value("queue_backend")
					require.True(t, ok)
					require.Contains(t, []string{"redis", "fdb"}, backend.AsString())
					require.Contains(t, point.Bounds, float64(20))
					require.Contains(t, point.Bounds, float64(15000))
					count += point.Count
					sum += point.Sum
				}
			case "inngest_queue_item_hint_total":
				for _, point := range metric.Data.(metricdata.Sum[int64]).DataPoints {
					require.Equal(t, 3, point.Attributes.Len())
					outcome, ok := point.Attributes.Value("outcome")
					require.True(t, ok)
					outcomes[outcome.AsString()] += point.Value
				}
			}
		}
	}
	return count, sum, outcomes
}

func TestFastPathExecutionLatencyScope(t *testing.T) {
	for _, backend := range []enums.QueueShardKind{enums.QueueShardKindRedis, "fdb"} {
		for _, tc := range []struct {
			name             string
			fromHint         bool
			invalidTimestamp string
			wantCount        uint64
		}{
			{name: "hint", fromHint: true, wantCount: 1},
			{name: "scan"},
			{name: "missing enqueue timestamp", fromHint: true, invalidTimestamp: "missing"},
			{name: "clock skew", fromHint: true, invalidTimestamp: "future"},
		} {
			t.Run(string(backend)+"/"+tc.name, func(t *testing.T) {
				q, shard, clock := hintTestQueue(t, nil)
				shard.kind = backend
				item := hintItem(shard.item, "latency")
				item.EnqueuedAt = clock.Now().Add(-125 * time.Millisecond).UnixMilli()
				item.WallTimeMS = item.AtMS
				if tc.invalidTimestamp == "missing" {
					item.EnqueuedAt = 0
				}
				if tc.invalidTimestamp == "future" {
					item.EnqueuedAt = clock.Now().Add(time.Second).UnixMilli()
				}
				lease := ulid.MustNew(ulid.Timestamp(clock.Now().Add(QueueLeaseDuration)), nil)
				item.LeaseID = &lease
				beforeCount, beforeSum, _ := hintMetrics(t)
				called := false
				_, err := q.ProcessItem(t.Context(), ProcessItem{I: item, fromHint: tc.fromHint}, func(context.Context, RunInfo, Item) (RunResult, error) {
					called = true
					return RunResult{}, nil
				})
				require.NoError(t, err)
				require.True(t, called, "invalid metric timestamps must not affect execution")
				count, sum, _ := hintMetrics(t)
				require.Equal(t, tc.wantCount, count-beforeCount)
				require.Equal(t, int64(tc.wantCount)*125, sum-beforeSum)
			})
		}
	}
}

func TestItemHintLeaseMetrics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		leaseErr   error
		noCapacity bool
		want       string
	}{
		{name: "missing or stale", leaseErr: ErrQueueItemNotFound, want: "not_found"},
		{name: "ordinary worker leased", leaseErr: ErrQueueItemAlreadyLeased, want: "lease_contention"},
		{name: "throttled", leaseErr: ErrQueueItemThrottled, want: "throttled"},
		{name: "concurrency", leaseErr: ErrPartitionConcurrencyLimit, want: "concurrency_limited"},
		{name: "backend failure", leaseErr: errors.New("backend unavailable"), want: "lease_error"},
		{name: "no workers", noCapacity: true, want: "no_worker_capacity"},
		{name: "dispatched", want: "dispatched"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, shard, _ := hintTestQueue(t, nil)
			shard.leaseErr = tc.leaseErr
			if tc.noCapacity {
				require.True(t, q.Semaphore().TryAcquire(2))
				defer q.Semaphore().Release(2)
			}
			beforeCount, _, before := hintMetrics(t)
			q.processItemHint(t.Context(), hintItem(shard.item, "outcome"), func(_ context.Context, work ProcessItem) (DispatchedItem, error) {
				require.True(t, work.fromHint)
				q.Semaphore().Release(1)
				return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
			})
			count, _, after := hintMetrics(t)
			require.Equal(t, beforeCount, count, "lease/dispatch alone is not a work start")
			delta := map[string]int64{}
			for outcome, n := range after {
				if n != before[outcome] {
					delta[outcome] = n - before[outcome]
				}
			}
			require.Equal(t, map[string]int64{tc.want: 1}, delta)
		})
	}
}
