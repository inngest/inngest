package memory_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// BenchmarkAcquire compares the Redis manager and the memory manager on the
// same request shapes.  the Redis side runs on miniredis unless
// CONSTRAINT_BENCH_REDIS_ADDR names a Valkey or Redis server.  every acquire
// uses a fresh idempotency key and limits are high enough that nothing is
// exhausted, so each iteration is one full grant.
//
//	go test ./pkg/constraintapi/memory/ -run '^$' -bench 'Acquire' -benchmem -cpu 1,16
func BenchmarkAcquire(b *testing.B) {
	for _, shape := range benchShapes() {
		b.Run(shape.name+"/redis", func(b *testing.B) {
			cm, sm := benchRedis(b)
			benchAcquire(b, cm, sm, shape)
		})
		b.Run(shape.name+"/memory", func(b *testing.B) {
			m, err := memory.NewManager(memory.WithShardName("bench"))
			require.NoError(b, err)
			b.Cleanup(func() { _ = m.Close() })
			benchAcquire(b, m, m, shape)
		})
	}
}

// BenchmarkAcquireRelease pairs every acquire with a release so the number of
// live leases stays flat.
func BenchmarkAcquireRelease(b *testing.B) {
	for _, shape := range benchShapes() {
		b.Run(shape.name+"/redis", func(b *testing.B) {
			cm, sm := benchRedis(b)
			benchAcquireRelease(b, cm, sm, shape)
		})
		b.Run(shape.name+"/memory", func(b *testing.B) {
			m, err := memory.NewManager(memory.WithShardName("bench"))
			require.NoError(b, err)
			b.Cleanup(func() { _ = m.Close() })
			benchAcquireRelease(b, m, m, shape)
		})
	}
}

// TestAcquireP99Guard asserts the memory manager's acquire p99 on one hot key
// under 16 goroutines.  it runs only when CONSTRAINT_BENCH_GUARD is set, so a
// loaded CI machine does not fail the build.
func TestAcquireP99Guard(t *testing.T) {
	if os.Getenv("CONSTRAINT_BENCH_GUARD") == "" {
		t.Skip("set CONSTRAINT_BENCH_GUARD=1 to run the latency guard")
	}
	m, err := memory.NewManager(memory.WithShardName("guard"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })

	shape := benchShapes()[2]
	id := newIDs()
	shape.setup(t, m, id)

	const goroutines, perG = 16, 2_000
	lat := make([][]time.Duration, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			lat[g] = make([]time.Duration, 0, perG)
			for i := 0; i < perG; i++ {
				req := shape.request(id, fmt.Sprintf("g%d-%d", g, i))
				start := time.Now()
				_, err := m.Acquire(context.Background(), req)
				lat[g] = append(lat[g], time.Since(start))
				require.NoError(t, err)
			}
		}(g)
	}
	wg.Wait()

	var all []time.Duration
	for _, l := range lat {
		all = append(all, l...)
	}
	p99 := percentile(all, 0.99)
	t.Logf("acquire p50 %s p99 %s max %s over %d calls", percentile(all, 0.5), p99, percentile(all, 1), len(all))
	require.Less(t, p99, 50*time.Microsecond)
}

type benchShape struct {
	name string
	// setup creates capacity so every acquire is granted.
	setup func(tb testing.TB, sm constraintapi.SemaphoreManager, id ids)
	// request builds a fresh request with the given idempotency key.
	request func(id ids, key string) *constraintapi.CapacityAcquireRequest
}

func benchShapes() []benchShape {
	const huge = 1 << 30
	sem := constraintapi.SemaphoreConstraint{ID: "app:bench", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	semConfig := semaphoreConfig(sem)
	concConfig := constraintapi.ConstraintConfig{FunctionVersion: 1, Concurrency: constraintapi.ConcurrencyConfig{AccountConcurrency: huge, FunctionConcurrency: huge}}
	mixedConfig := semaphoreConfig(sem)
	mixedConfig.Concurrency = concConfig.Concurrency

	base := func(id ids, key string, config constraintapi.ConstraintConfig, constraints []constraintapi.ConstraintItem) *constraintapi.CapacityAcquireRequest {
		return &constraintapi.CapacityAcquireRequest{
			AccountID: id.account, EnvID: id.env, FunctionID: id.fn, AppID: id.app,
			IdempotencyKey: key, LeaseIdempotencyKeys: []string{key}, Amount: 1,
			Configuration: config, Constraints: constraints,
			CurrentTime: time.Now(), Duration: 3 * time.Second, MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease},
		}
	}
	setSem := func(tb testing.TB, sm constraintapi.SemaphoreManager, id ids) {
		_, err := sm.SetCapacity(context.Background(), id.account, sem.ID, "bench-"+uuid.NewString(), huge)
		require.NoError(tb, err)
	}

	return []benchShape{
		{
			name:  "semaphore",
			setup: setSem,
			request: func(id ids, key string) *constraintapi.CapacityAcquireRequest {
				s := sem
				return base(id, key, semConfig, []constraintapi.ConstraintItem{semaphoreItem(&s)})
			},
		},
		{
			name:  "concurrency",
			setup: func(testing.TB, constraintapi.SemaphoreManager, ids) {},
			request: func(id ids, key string) *constraintapi.CapacityAcquireRequest {
				return base(id, key, concConfig, []constraintapi.ConstraintItem{
					{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount}},
					{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn}},
				})
			},
		},
		{
			name:  "mixed",
			setup: setSem,
			request: func(id ids, key string) *constraintapi.CapacityAcquireRequest {
				s := sem
				return base(id, key, mixedConfig, []constraintapi.ConstraintItem{
					{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount}},
					{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn}},
					semaphoreItem(&s),
				})
			},
		},
	}
}

func benchRedis(b *testing.B) (constraintapi.CapacityManager, constraintapi.SemaphoreManager) {
	var rc rueidis.Client
	if addr := os.Getenv("CONSTRAINT_BENCH_REDIS_ADDR"); addr != "" {
		var err error
		rc, err = rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
		require.NoError(b, err)
		b.Cleanup(rc.Close)
	} else {
		_, rc = newMiniredisClient(b)
	}
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithShardName("bench"),
		constraintapi.WithClient(rc),
		constraintapi.WithClock(clockwork.NewRealClock()),
	)
	require.NoError(b, err)
	return cm, constraintapi.NewRedisSemaphoreManager(rc)
}

func benchAcquire(b *testing.B, cm constraintapi.CapacityManager, sm constraintapi.SemaphoreManager, shape benchShape) {
	id := newIDs()
	shape.setup(b, sm, id)
	lat := &latencies{}
	var n atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var local []time.Duration
		for pb.Next() {
			key := "k" + strconv.FormatInt(n.Add(1), 36)
			req := shape.request(id, key)
			start := time.Now()
			resp, err := cm.Acquire(context.Background(), req)
			local = append(local, time.Since(start))
			if err != nil {
				b.Fatalf("acquire failed: %v", err)
			}
			if len(resp.Leases) != 1 {
				b.Fatalf("expected one lease, got %d", len(resp.Leases))
			}
		}
		lat.add(local)
	})
	b.StopTimer()
	lat.report(b)
}

func benchAcquireRelease(b *testing.B, cm constraintapi.CapacityManager, sm constraintapi.SemaphoreManager, shape benchShape) {
	id := newIDs()
	shape.setup(b, sm, id)
	var n atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := "k" + strconv.FormatInt(n.Add(1), 36)
			resp, err := cm.Acquire(context.Background(), shape.request(id, key))
			if err != nil {
				b.Fatalf("acquire failed: %v", err)
			}
			if len(resp.Leases) != 1 {
				b.Fatalf("expected one lease, got %d", len(resp.Leases))
			}
			_, err = cm.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
				IdempotencyKey: "r" + key, AccountID: id.account, LeaseID: resp.Leases[0].LeaseID,
				Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease},
			})
			if err != nil {
				b.Fatalf("release failed: %v", err)
			}
		}
	})
}

type latencies struct {
	mu  sync.Mutex
	all []time.Duration
}

func (l *latencies) add(d []time.Duration) {
	l.mu.Lock()
	l.all = append(l.all, d...)
	l.mu.Unlock()
}

func (l *latencies) report(b *testing.B) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.all) == 0 {
		return
	}
	b.ReportMetric(float64(percentile(l.all, 0.5).Nanoseconds())/1e3, "p50-µs")
	b.ReportMetric(float64(percentile(l.all, 0.99).Nanoseconds())/1e3, "p99-µs")
}

func percentile(d []time.Duration, p float64) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := make([]time.Duration, len(d))
	copy(s, d)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := int(float64(len(s)-1) * p)
	return s[i]
}
