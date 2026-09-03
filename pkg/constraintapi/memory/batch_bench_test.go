package memory_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/stretchr/testify/require"
)

// BenchmarkBatch fires one batch of concurrent acquires at each backend, one
// goroutine per request, all released at once, and reports throughput, per
// request latency and the wall time to finish the batch.  batch sizes come
// from CONSTRAINT_BENCH_BATCH as a comma separated list, default 10000 and
// 100000.  the Redis side runs on miniredis unless CONSTRAINT_BENCH_REDIS_ADDR
// names a Valkey or Redis server.
//
//	go test ./pkg/constraintapi/memory/ -run '^$' -bench 'Batch' -benchtime=1x -cpu 8
func BenchmarkBatch(b *testing.B) {
	sizes := []int{10_000, 100_000}
	if v := os.Getenv("CONSTRAINT_BENCH_BATCH"); v != "" {
		sizes = sizes[:0]
		for _, part := range strings.Split(v, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			require.NoError(b, err)
			sizes = append(sizes, n)
		}
	}
	shape := benchShapes()[0]

	for _, n := range sizes {
		b.Run(fmt.Sprintf("%d/redis", n), func(b *testing.B) {
			cm, sm := benchRedis(b)
			runBatch(b, cm, sm, shape, n)
		})
		b.Run(fmt.Sprintf("%d/memory", n), func(b *testing.B) {
			m, err := memory.NewManager(memory.WithShardName("bench"))
			require.NoError(b, err)
			b.Cleanup(func() { _ = m.Close() })
			runBatch(b, m, m, shape, n)
		})
	}
}

func runBatch(b *testing.B, cm constraintapi.CapacityManager, sm constraintapi.SemaphoreManager, shape benchShape, n int) {
	id := newIDs()
	shape.setup(b, sm, id)
	ctx := context.Background()

	var wallTotal time.Duration
	var all []time.Duration

	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		b.StopTimer()
		reqs := make([]*constraintapi.CapacityAcquireRequest, n)
		for j := range reqs {
			reqs[j] = shape.request(id, fmt.Sprintf("b%d-%d", iter, j))
		}
		lat := make([]time.Duration, n)
		var failed atomic.Int64
		start := make(chan struct{})
		var ready, done sync.WaitGroup
		ready.Add(n)
		done.Add(n)
		for j := range reqs {
			go func(j int) {
				defer done.Done()
				ready.Done()
				<-start
				t0 := time.Now()
				resp, err := cm.Acquire(ctx, reqs[j])
				lat[j] = time.Since(t0)
				if err != nil || len(resp.Leases) != 1 {
					failed.Add(1)
				}
			}(j)
		}
		ready.Wait()

		b.StartTimer()
		batchStart := time.Now()
		close(start)
		done.Wait()
		wall := time.Since(batchStart)
		b.StopTimer()

		require.Zero(b, failed.Load(), "acquires failed")
		wallTotal += wall
		all = append(all, lat...)
	}

	wall := wallTotal / time.Duration(b.N)
	b.ReportMetric(float64(n)/wall.Seconds(), "req/s")
	b.ReportMetric(float64(wall.Microseconds())/1e3, "batch-ms")
	b.ReportMetric(float64(percentile(all, 0.5).Microseconds()), "p50-µs")
	b.ReportMetric(float64(percentile(all, 0.99).Microseconds()), "p99-µs")
	b.ReportMetric(float64(percentile(all, 1).Microseconds()), "max-µs")
}
