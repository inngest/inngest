package cluster

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/cluster/internal/benchload"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/stretchr/testify/require"
)

const (
	// benchLimit is the function concurrency limit of every benchmark
	// account.  no run reaches it, so no acquire is denied.
	benchLimit = 1 << 30
	// benchCallers is the number of caller goroutines per CPU.  a caller that
	// waits for a follower does no work, so a low number bounds the
	// throughput by the wait and not by the cluster.
	benchCallers = 128
	// benchTimeout is the acknowledgement timeout of every node and the
	// heartbeat timeout of the coordinator.  it is long, so a slow moment
	// under load removes no follower and moves no primary.
	benchTimeout = 30 * time.Second
	// benchLease is the lease duration of the acquire plus release runs.  it
	// is the longest the store accepts, so no lease expires before its
	// release and the sweeper reclaims nothing.
	benchLease = constraintapi.MaximumDuration
	// benchShortLease is the lease duration of the acquire only run.  it is
	// longer than a run at the default bench time and at 3 seconds, so no
	// lease expires in the timed region.  the number of live leases is at
	// most the request rate times this duration.
	benchShortLease = 5 * time.Second
	// benchCheckTTLEnv names the environment variable that sets the TTL of
	// the constraint check record of every node in seconds.  every grant
	// leaves one such record, so the TTL sets how many the primary keeps.
	benchCheckTTLEnv = "CLUSTER_BENCH_CHECK_TTL_SECONDS"
)

type benchLevel struct {
	name  string
	level constraintapi.CommitLevel
}

var benchLevels = []benchLevel{
	{
		name:  "primary",
		level: constraintapi.CommitLevelPrimary,
	},
	{
		name:  "two",
		level: constraintapi.CommitLevelTwo,
	},
	{
		name:  "all",
		level: constraintapi.CommitLevelAll,
	},
}

// benchSpec is one benchmark run.
type benchSpec struct {
	nodes    int
	shards   int
	rf       int
	accounts int
	// level is the commit level of every acquire.  a release keeps the
	// default level of a release, which does not wait for a follower.
	level constraintapi.CommitLevel
	// lease is the lease duration of every acquire.
	lease time.Duration
	// release sends a release after every acquire, so the number of live
	// leases stays flat.
	release bool
}

// BenchmarkClusterMem measures one shard on MemNetwork through the real
// Client, at RF 1, 2 and 3 and at every commit level.  one iteration is one
// acquire and one release, spread over 64 accounts.  req/s counts both calls.
// p50-µs and p99-µs are the latency of the acquire call as the caller sees
// it.  the nodes, the client and the callers share one process.
//
//	go test ./pkg/constraintapi/cluster/ -run '^$' -bench 'ClusterMem$' -benchtime=3s -cpu 8
func BenchmarkClusterMem(b *testing.B) {
	for _, rf := range []int{1, 2, 3} {
		for _, lv := range benchLevels {
			b.Run(fmt.Sprintf("rf=%d/level=%s", rf, lv.name), func(b *testing.B) {
				benchClusterRun(b, benchSpec{
					nodes:    rf,
					shards:   1,
					rf:       rf,
					accounts: 64,
					level:    lv.level,
					lease:    benchLease,
					release:  true,
				})
			})
		}
	}
}

// BenchmarkClusterMemShards is the RF 3 level all run of BenchmarkClusterMem
// with 4096 accounts spread over 1, 16 and 64 shards on 3 nodes.  every node
// is the primary of a third of the shards.
//
//	go test ./pkg/constraintapi/cluster/ -run '^$' -bench 'ClusterMemShards' -benchtime=3s -cpu 8
func BenchmarkClusterMemShards(b *testing.B) {
	for _, shards := range []int{1, 16, 64} {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			benchClusterRun(b, benchSpec{
				nodes:    3,
				shards:   shards,
				rf:       3,
				accounts: 4096,
				level:    constraintapi.CommitLevelAll,
				lease:    benchLease,
				release:  true,
			})
		})
	}
}

// BenchmarkClusterMemAcquireOnly is the RF 3 level all run of
// BenchmarkClusterMem with no release, so it shows the grant path alone.  one
// iteration is one acquire.  the leases stay live until the run ends.  the
// limit of 1<<30 per account is far above the leases one run grants, so no
// acquire is denied.
//
//	go test ./pkg/constraintapi/cluster/ -run '^$' -bench 'ClusterMemAcquireOnly' -benchtime=3s -cpu 8
func BenchmarkClusterMemAcquireOnly(b *testing.B) {
	benchClusterRun(b, benchSpec{
		nodes:    3,
		shards:   1,
		rf:       3,
		accounts: 64,
		level:    constraintapi.CommitLevelAll,
		lease:    benchShortLease,
		release:  false,
	})
}

// newBenchCluster returns a harness cluster of real nodes with long
// timeouts.  it creates the shards, starts every node and waits until every
// shard has an open primary and the table lists every replica as in sync.
// the nodes stop when the benchmark run ends.
func newBenchCluster(b *testing.B, nodes, shards, rf int) *testCluster {
	b.Helper()
	var opts []memory.Option
	if v := os.Getenv(benchCheckTTLEnv); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			b.Fatalf("%s is %q, want a number of seconds", benchCheckTTLEnv, v)
		}
		opts = append(opts, memory.WithConstraintCheckIdempotencyTTL(time.Duration(n)*time.Second))
	}
	c := &testCluster{
		t:     b,
		net:   NewMemNetwork(),
		co:    NewMemCoordinator(20*time.Millisecond, benchTimeout),
		nodes: map[string]testNode{},
		start: func(name string, tr Transport, co Coordinator) (testNode, error) {
			return NewNode(Config{
				Name:              name,
				Transport:         tr,
				Coordinator:       co,
				AckTimeout:        benchTimeout,
				HeartbeatInterval: 50 * time.Millisecond,
				DialBackoff:       5 * time.Millisecond,
				ManagerOptions:    opts,
			})
		},
	}
	for i := range nodes {
		c.names = append(c.names, fmt.Sprintf("n%d", i))
	}
	b.Cleanup(c.Close)
	require.NoError(b, c.Init(shards, rf))
	require.NoError(b, c.StartAll())
	c.waitStable(b)
	return c
}

// benchAccount is one account with one function.
type benchAccount struct {
	account uuid.UUID
	env     uuid.UUID
	fn      uuid.UUID
}

func newBenchAccounts(n int) []benchAccount {
	out := make([]benchAccount, n)
	for i := range out {
		out[i] = benchAccount{
			account: uuid.New(),
			env:     uuid.New(),
			fn:      uuid.New(),
		}
	}
	return out
}

// acquireReq asks for one lease on the account's function concurrency.  the
// constraint list is new for every request, because MemNetwork passes the
// request to the node by reference.
func (a *benchAccount) acquireReq(key string, level constraintapi.CommitLevel, lease time.Duration) *constraintapi.CapacityAcquireRequest {
	return &constraintapi.CapacityAcquireRequest{
		IdempotencyKey: key,
		AccountID:      a.account,
		EnvID:          a.env,
		FunctionID:     a.fn,
		Configuration: constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				FunctionConcurrency: benchLimit,
			},
		},
		Constraints:          ledgerConstraints(),
		Amount:               1,
		LeaseIdempotencyKeys: []string{key},
		CurrentTime:          time.Now(),
		Duration:             lease,
		MaximumLifetime:      time.Hour,
		Source:               ledgerSource(),
		CommitLevel:          level,
	}
}

// benchOp runs one iteration.  it returns the latency of the acquire call.
func benchOp(ctx context.Context, m constraintapi.CapacityManager, a *benchAccount, key string, spec benchSpec) (time.Duration, error) {
	req := a.acquireReq(key, spec.level, spec.lease)
	start := time.Now()
	resp, err := m.Acquire(ctx, req)
	took := time.Since(start)
	if err != nil {
		return 0, fmt.Errorf("acquire %s: %w", key, err)
	}
	if len(resp.Leases) != 1 {
		return 0, fmt.Errorf("acquire %s was denied, got %d leases", key, len(resp.Leases))
	}
	if !spec.release {
		return took, nil
	}
	_, err = m.Release(ctx, &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: "r" + key,
		AccountID:      a.account,
		LeaseID:        resp.Leases[0].LeaseID,
		Source:         ledgerSource(),
	})
	if err != nil {
		return 0, fmt.Errorf("release %s: %w", key, err)
	}
	return took, nil
}

// benchWarm runs one acquire at level all and one release for every account.
// afterwards the client has a stream to every primary, every primary has the
// constraint set of every account, and every in-sync follower has
// acknowledged a record.
func benchWarm(tb testing.TB, m constraintapi.CapacityManager, accounts []benchAccount) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	spec := benchSpec{
		level:   constraintapi.CommitLevelAll,
		lease:   benchLease,
		release: true,
	}
	var next atomic.Int64
	var wg sync.WaitGroup
	errc := make(chan error, 64)
	for range cap(errc) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= len(accounts) {
					return
				}
				if _, err := benchOp(ctx, m, &accounts[i], "warm"+strconv.Itoa(i), spec); err != nil {
					errc <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		require.NoError(tb, err)
	}
}

// benchSetup starts the cluster of spec and a client, and warms every
// account.
func benchSetup(b *testing.B, spec benchSpec) (*testCluster, *Client, []benchAccount) {
	b.Helper()
	c := newBenchCluster(b, spec.nodes, spec.shards, spec.rf)
	cl, err := NewClient(ClientConfig{
		Transport: c.net.Transport("bench-client"),
		Seeds:     c.names,
	})
	require.NoError(b, err)
	b.Cleanup(func() { _ = cl.Close() })

	accounts := newBenchAccounts(spec.accounts)
	tbl := c.Table()
	covered := map[int]bool{}
	for i := range accounts {
		covered[tbl.ShardOf(accounts[i].account)] = true
	}
	require.Len(b, covered, spec.shards, "the accounts do not reach every shard")
	benchWarm(b, cl, accounts)
	require.True(b, c.stable(), "the cluster is not steady after the warm up, table %s", tableString(c))
	return c, cl, accounts
}

// benchRates are the request rates of the paced runs.  zero is no pacing.
var benchRates = []struct {
	name string
	rate int
}{
	{
		name: "50k",
		rate: 50_000,
	},
	{
		name: "100k",
		rate: 100_000,
	},
	{
		name: "250k",
		rate: 250_000,
	},
	{
		name: "500k",
		rate: 500_000,
	},
	{
		name: "1M",
		rate: 1_000_000,
	},
	{
		name: "max",
		rate: 0,
	},
}

// BenchmarkPaced measures one shard on MemNetwork at a fixed request rate,
// on one node and on three nodes with RF 3 at every commit level.  the load
// is acquire plus release from 128 callers per CPU through the real Client,
// spread over 64 accounts.  one run is a new cluster, 3 seconds of load that
// is not measured, and then 15 seconds that are.  CLUSTER_BENCH_SECONDS
// changes the 15.
//
// req/s is the rate the load reached and counts both calls.  a run whose
// req/s is below its rate could not keep that rate.  p50-µs, p99-µs, p999-µs
// and max-µs are the latency of the acquire call as the caller sees it.
// sched-p99-µs and sched-p999-µs count from the time the schedule wanted the
// acquire sent.  they add the time a caller was behind its schedule, so they
// also show the requests that a stall delayed and that a caller that waits
// for its answer never sent.
//
//	go test ./pkg/constraintapi/cluster/ -run '^$' -bench 'Paced' -benchtime=1x -cpu 8
func BenchmarkPaced(b *testing.B) {
	setups := []struct {
		name  string
		nodes int
		level constraintapi.CommitLevel
	}{
		{
			name:  "nodes=1",
			nodes: 1,
			level: constraintapi.CommitLevelAll,
		},
		{
			name:  "nodes=3/level=primary",
			nodes: 3,
			level: constraintapi.CommitLevelPrimary,
		},
		{
			name:  "nodes=3/level=two",
			nodes: 3,
			level: constraintapi.CommitLevelTwo,
		},
		{
			name:  "nodes=3/level=all",
			nodes: 3,
			level: constraintapi.CommitLevelAll,
		},
	}
	for _, s := range setups {
		for _, r := range benchRates {
			b.Run(s.name+"/rate="+r.name, func(b *testing.B) {
				spec := benchSpec{
					nodes:    s.nodes,
					shards:   1,
					rf:       s.nodes,
					accounts: 64,
					level:    s.level,
					lease:    benchLease,
					release:  true,
				}
				c, cl, accounts := benchSetup(b, spec)
				version := c.Table().Version
				benchload.Run(b, benchload.Config{
					Rate:    r.rate,
					Callers: benchCallers * runtime.GOMAXPROCS(0),
					Op: func(ctx context.Context, k int64) (time.Duration, error) {
						return benchOp(ctx, cl, &accounts[int(k%int64(len(accounts)))], "p"+strconv.FormatInt(k, 36), spec)
					},
				})
				if got := c.Table().Version; got != version {
					b.Fatalf("the table went from version %d to %d during the run, table %s", version, got, tableString(c))
				}
			})
		}
	}
}

func benchClusterRun(b *testing.B, spec benchSpec) {
	c, cl, accounts := benchSetup(b, spec)
	version := c.Table().Version

	lat := &benchLatencies{}
	var n atomic.Int64
	var failed atomic.Bool
	b.SetParallelism(benchCallers)
	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		var local []time.Duration
		for pb.Next() {
			i := n.Add(1)
			took, err := benchOp(ctx, cl, &accounts[int(i%int64(len(accounts)))], "k"+strconv.FormatInt(i, 36), spec)
			if err != nil {
				// one message is enough.  every caller fails the same way.
				if failed.CompareAndSwap(false, true) {
					b.Error(err)
				}
				continue
			}
			local = append(local, took)
		}
		lat.add(local)
	})
	wall := time.Since(start)
	b.StopTimer()

	// a table change in the timed region means a follower left the in-sync
	// set or a primary moved.  the numbers are then not those of the RF and
	// the level in the name.
	if got := c.Table().Version; got != version {
		b.Fatalf("the table went from version %d to %d during the run, table %s", version, got, tableString(c))
	}
	calls := b.N
	if spec.release {
		calls *= 2
	}
	b.ReportMetric(float64(calls)/wall.Seconds(), "req/s")
	lat.report(b)
}

// benchLatencies collects the acquire latency of every caller.
type benchLatencies struct {
	mu  sync.Mutex
	all []time.Duration
}

func (l *benchLatencies) add(d []time.Duration) {
	l.mu.Lock()
	l.all = append(l.all, d...)
	l.mu.Unlock()
}

func (l *benchLatencies) report(b *testing.B) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.all) == 0 {
		return
	}
	sort.Slice(l.all, func(i, j int) bool { return l.all[i] < l.all[j] })
	at := func(p float64) float64 {
		return float64(l.all[int(float64(len(l.all)-1)*p)].Nanoseconds()) / 1e3
	}
	b.ReportMetric(at(0.5), "p50-µs")
	b.ReportMetric(at(0.99), "p99-µs")
}

// benchJoinLimit is how long one join may take before the run gives up.
const benchJoinLimit = 30 * time.Second

// BenchmarkJoinUnderLoad measures a follower that joins one shard from a
// fresh start while the shard is under load.  the shard has RF 3 and every
// acquire waits at level all.  one iteration kills a follower, starts it
// again with no state, and waits until the table lists it as in sync.
//
// the load is acquire plus release from 128 callers per CPU through the
// real Client, paced at 200k and at 800k requests per second and not paced
// at all.  steady-req/s is what the load reached before the first join and
// req/s is what it reached while the follower joined.
//
// join-ms is the time from the start of the new process until it is in
// sync.  that is at least the idempotency TTL of 5 seconds, because the
// primary waits until the acquire records the export left out have expired.
// transfers is how many state transfers the primary started for it, and 1
// means the follower reached the end of the log from the tail after one
// transfer.  state-records and state-MB are the size of one exported state.
// max-acquire-ms is the slowest acquire a caller saw during the join.  a
// join is as long as the 5 seconds between two housekeeping passes of the
// primary, so it has one such pass in it.  joined is 1 when every join ended
// inside the limit.
//
//	go test ./pkg/constraintapi/cluster/ -run '^$' -bench 'JoinUnderLoad' -benchtime=3x -cpu 8
func BenchmarkJoinUnderLoad(b *testing.B) {
	for _, v := range []struct {
		name string
		rate int
	}{
		{
			name: "rate=200k",
			rate: 200_000,
		},
		{
			name: "rate=800k",
			rate: 800_000,
		},
		{
			name: "rate=max",
			rate: 0,
		},
	} {
		b.Run(v.name, func(b *testing.B) {
			c := newBenchCluster(b, 3, 1, 3)
			benchJoinRun(b, c, c.net.Transport("bench-client"), c.names, v.rate)
		})
	}
}

// benchJoinRun is the body of BenchmarkJoinUnderLoad on a cluster that is
// already steady.  the client reaches the cluster through tr and seeds.
func benchJoinRun(b *testing.B, c *testCluster, tr Transport, seeds []string, rate int) {
	cl, err := NewClient(ClientConfig{
		Transport: tr,
		Seeds:     seeds,
	})
	require.NoError(b, err)
	b.Cleanup(func() { _ = cl.Close() })
	accounts := newBenchAccounts(64)
	benchWarm(b, cl, accounts)
	require.True(b, c.stable(), "the cluster is not steady after the warm up, table %s", tableString(c))

	spec := benchSpec{
		level:   constraintapi.CommitLevelAll,
		lease:   benchLease,
		release: true,
	}
	callers := benchCallers * runtime.GOMAXPROCS(0)
	// one caller makes two requests per iteration.
	var period time.Duration
	if rate > 0 {
		period = time.Duration(float64(time.Second) * 2 * float64(callers) / float64(rate))
	}

	var (
		n       atomic.Int64
		calls   atomic.Int64
		joining atomic.Bool
		worst   atomic.Int64
		failed  atomic.Bool
		wg      sync.WaitGroup
	)
	ctx, stop := context.WithCancel(context.Background())
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			next := time.Now().Add(period * time.Duration(i) / time.Duration(callers))
			for ctx.Err() == nil {
				if period > 0 {
					if d := time.Until(next); d > 0 {
						time.Sleep(d)
					} else if d < -10*period {
						// do not make up for a long stall with a burst.
						next = time.Now()
					}
					next = next.Add(period)
				}
				k := n.Add(1)
				took, err := benchOp(ctx, cl, &accounts[int(k%int64(len(accounts)))], "j"+strconv.FormatInt(k, 36), spec)
				if err != nil {
					if ctx.Err() == nil && failed.CompareAndSwap(false, true) {
						b.Error(err)
					}
					continue
				}
				calls.Add(2)
				if joining.Load() {
					for {
						cur := worst.Load()
						if int64(took) <= cur || worst.CompareAndSwap(cur, int64(took)) {
							break
						}
					}
				}
			}
		}()
	}
	defer func() {
		stop()
		wg.Wait()
	}()
	// let the load and the idempotency records it leaves reach a steady
	// size.  an acquire record lives 5 seconds.
	time.Sleep(3 * time.Second)
	steadyCalls := calls.Load()
	time.Sleep(3 * time.Second)
	steady := float64(calls.Load()-steadyCalls) / 3

	row := c.Table().Shards[0]
	follower := followersOf(row)[0]
	primary := c.node(row.Primary)
	inSync := func() bool { return slices.Contains(c.Table().Shards[0].InSync, follower) }

	var joinTime time.Duration
	var transfers, records, size uint64
	var loadCalls int64
	joined := 0
	b.ResetTimer()
	for range b.N {
		c.Kill(follower)
		eventually(b, benchJoinLimit, func() bool { return !inSync() }, "%s did not leave the in-sync set", follower)
		before := primary.Stats()
		callsBefore := calls.Load()
		joining.Store(true)
		start := time.Now()
		require.NoError(b, c.Restart(follower))
		ok := true
		for !inSync() {
			if time.Since(start) > benchJoinLimit {
				ok = false
				break
			}
			time.Sleep(time.Millisecond)
		}
		took := time.Since(start)
		joining.Store(false)
		after := primary.Stats()

		joinTime += took
		loadCalls += calls.Load() - callsBefore
		transfers += after.Transfers - before.Transfers
		records += after.StateRecords - before.StateRecords
		size += after.StateBytes - before.StateBytes
		if !ok {
			b.Logf("%s was not in sync after %v, the primary started %d transfers, table %s", follower, took, after.Transfers-before.Transfers, tableString(c))
			break
		}
		joined++
	}
	b.StopTimer()

	runs := float64(b.N)
	b.ReportMetric(0, "ns/op")
	b.ReportMetric(float64(joinTime.Milliseconds())/runs, "join-ms")
	b.ReportMetric(float64(transfers)/runs, "transfers")
	if transfers > 0 {
		b.ReportMetric(float64(records)/float64(transfers), "state-records")
		b.ReportMetric(float64(size)/float64(transfers)/(1<<20), "state-MB")
	}
	b.ReportMetric(float64(worst.Load())/1e6, "max-acquire-ms")
	b.ReportMetric(steady, "steady-req/s")
	b.ReportMetric(float64(loadCalls)/joinTime.Seconds(), "req/s")
	b.ReportMetric(float64(joined)/runs, "joined")
}
