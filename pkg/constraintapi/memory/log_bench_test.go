package memory_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/stretchr/testify/require"
)

// BenchmarkAcquireLog measures what an attached log costs an acquire, beside
// the plain store on the same shapes.
//
// plain has no log.  logged appends every record and never waits, which is
// the cost on the hot path.  logged+wait also waits for every replica, with a
// goroutine that drains and acknowledges at once, so it adds one park and
// wake per acquire and no network.  logged+wait runs 64 goroutines per CPU,
// because a goroutine that waits does no work and a real primary has many
// requests in flight.
//
//	go test ./pkg/constraintapi/memory/ -run '^$' -bench 'AcquireLog' -benchmem -cpu 8
func BenchmarkAcquireLog(b *testing.B) {
	for _, name := range []string{"semaphore", "throttle", "mixed"} {
		shape := benchShapeNamed(b, name)
		b.Run(name+"/plain", func(b *testing.B) {
			benchAcquireLog(b, shape, false, constraintapi.CommitLevelPrimary, 1, false)
		})
		b.Run(name+"/logged", func(b *testing.B) {
			benchAcquireLog(b, shape, true, constraintapi.CommitLevelPrimary, 1, false)
		})
		b.Run(name+"/logged+wait", func(b *testing.B) {
			benchAcquireLog(b, shape, true, constraintapi.CommitLevelAll, 64, false)
		})
	}
}

// BenchmarkAcquireReleaseLog is BenchmarkAcquireLog with a release after
// every acquire, so the number of live leases stays flat.
func BenchmarkAcquireReleaseLog(b *testing.B) {
	for _, name := range []string{"semaphore", "throttle", "mixed"} {
		shape := benchShapeNamed(b, name)
		b.Run(name+"/plain", func(b *testing.B) {
			benchAcquireLog(b, shape, false, constraintapi.CommitLevelPrimary, 1, true)
		})
		b.Run(name+"/logged", func(b *testing.B) {
			benchAcquireLog(b, shape, true, constraintapi.CommitLevelPrimary, 1, true)
		})
		b.Run(name+"/logged+wait", func(b *testing.B) {
			benchAcquireLog(b, shape, true, constraintapi.CommitLevelAll, 64, true)
		})
	}
}

// drainAndAck is the smallest thing that can stand in for a replicator.  it
// drains the log whenever an operation waits, and at least every 200
// microseconds, and acknowledges each handle at once.
func drainAndAck(l *memory.Log, stop <-chan struct{}, done chan<- struct{}, records *atomic.Int64) {
	defer close(done)
	tick := time.NewTicker(200 * time.Microsecond)
	defer tick.Stop()
	var buf []memory.Record
	for {
		select {
		case <-l.Wake():
		case <-tick.C:
		case <-stop:
			return
		}
		recs, batch := l.Drain(buf[:0])
		records.Add(int64(len(recs)))
		buf = recs
		batch.AckAll()
	}
}

func benchAcquireLog(b *testing.B, shape benchShape, logged bool, level constraintapi.CommitLevel, parallelism int, release bool) {
	opts := []memory.Option{memory.WithShardName("bench")}
	var records atomic.Int64
	if logged {
		l := memory.NewLog()
		opts = append(opts, memory.WithLog(l))
		stop, done := make(chan struct{}), make(chan struct{})
		go drainAndAck(l, stop, done, &records)
		b.Cleanup(func() {
			close(stop)
			<-done
		})
	}
	m, err := memory.NewManager(opts...)
	require.NoError(b, err)
	b.Cleanup(func() { _ = m.Close() })

	id := newIDs()
	shape.setup(b, m, id)
	lat := &latencies{}
	var n atomic.Int64

	b.SetParallelism(parallelism)
	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		var local []time.Duration
		for pb.Next() {
			key := "k" + strconv.FormatInt(n.Add(1), 36)
			req := shape.request(id, key)
			req.CommitLevel = level
			t0 := time.Now()
			resp, err := m.Acquire(ctx, req)
			local = append(local, time.Since(t0))
			if err != nil {
				b.Fatalf("acquire failed: %v", err)
			}
			if len(resp.Leases) != 1 {
				b.Fatalf("expected one lease, got %d", len(resp.Leases))
			}
			if !release {
				continue
			}
			_, err = m.Release(ctx, &constraintapi.CapacityReleaseRequest{
				IdempotencyKey: "r" + key,
				AccountID:      id.account,
				LeaseID:        resp.Leases[0].LeaseID,
				Source: constraintapi.LeaseSource{
					Service:  constraintapi.ServiceExecutor,
					Location: constraintapi.CallerLocationItemLease,
				},
			})
			if err != nil {
				b.Fatalf("release failed: %v", err)
			}
		}
		lat.add(local)
	})
	wall := time.Since(start)
	b.StopTimer()
	b.ReportMetric(float64(b.N)/wall.Seconds(), "req/s")
	if logged {
		b.ReportMetric(float64(records.Load())/float64(b.N), "records/op")
	}
	lat.report(b)
}
