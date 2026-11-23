package tcp

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
	"github.com/inngest/inngest/pkg/constraintapi/cluster/internal/benchload"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

const (
	// clusterBenchLimit is the function concurrency limit of every benchmark
	// account.  no run reaches it, so no acquire is denied.
	clusterBenchLimit = 1 << 30
	// clusterBenchCallers is the number of caller goroutines per CPU.  a
	// caller that waits for a follower does no work, so a low number bounds
	// the throughput by the wait and not by the cluster.
	clusterBenchCallers = 128
	// clusterBenchTimeout is the acknowledgement timeout of every node and
	// the heartbeat timeout of the coordinator.  it is long, so a slow moment
	// under load removes no follower and moves no primary.
	clusterBenchTimeout = 30 * time.Second
	// clusterBenchLease is the lease duration of every acquire.  it is the
	// longest the store accepts, so no lease expires before its release and
	// the sweeper reclaims nothing.
	clusterBenchLease = constraintapi.MaximumDuration
)

// clusterBenchSpec is one benchmark run.
type clusterBenchSpec struct {
	nodes    int
	shards   int
	rf       int
	accounts int
	// level is the commit level of every acquire.  a release keeps the
	// default level of a release, which does not wait for a follower.
	level constraintapi.CommitLevel
}

// BenchmarkClusterTCP measures the cluster end to end through the real
// Client over TCP on loopback, with a MemCoordinator.  one iteration is one
// acquire and one release.  req/s counts both calls.  p50-µs and p99-µs are
// the latency of the acquire call as the caller sees it.
//
// every node, the client and the callers run in this one process on one
// machine.  they share its CPUs, and loopback has no network delay.  the
// numbers show the cost of the encoding and of the socket calls.  they do
// not show a cluster on three machines.
//
//	go test ./pkg/constraintapi/cluster/tcp/ -run '^$' -bench 'ClusterTCP' -benchtime=3s -cpu 8
func BenchmarkClusterTCP(b *testing.B) {
	runs := []struct {
		name string
		spec clusterBenchSpec
	}{
		{
			name: "rf=1/level=all/shards=1",
			spec: clusterBenchSpec{
				nodes:    1,
				shards:   1,
				rf:       1,
				accounts: 64,
				level:    constraintapi.CommitLevelAll,
			},
		},
		{
			name: "rf=3/level=primary/shards=1",
			spec: clusterBenchSpec{
				nodes:    3,
				shards:   1,
				rf:       3,
				accounts: 64,
				level:    constraintapi.CommitLevelPrimary,
			},
		},
		{
			name: "rf=3/level=all/shards=1",
			spec: clusterBenchSpec{
				nodes:    3,
				shards:   1,
				rf:       3,
				accounts: 64,
				level:    constraintapi.CommitLevelAll,
			},
		},
		{
			name: "rf=3/level=all/shards=16",
			spec: clusterBenchSpec{
				nodes:    3,
				shards:   16,
				rf:       3,
				accounts: 4096,
				level:    constraintapi.CommitLevelAll,
			},
		},
	}
	for _, run := range runs {
		b.Run(run.name, func(b *testing.B) {
			clusterBenchRun(b, run.spec)
		})
	}
}

// clusterBenchNodes is a set of nodes on loopback with one MemCoordinator.
type clusterBenchNodes struct {
	co    *cluster.MemCoordinator
	tr    *Transport
	rf    int
	addrs []string

	mu        sync.Mutex
	nodes     map[string]*cluster.Node
	listeners map[string]*Listener
}

// start starts the node with this name on a new loopback port and returns
// its address.
func (c *clusterBenchNodes) start(b *testing.B, name string) string {
	b.Helper()
	// the listener starts first because the node needs the real port for
	// the address it registers.
	mux := cluster.NewMux()
	ln, err := c.tr.Listen("127.0.0.1:0", mux.Accept)
	require.NoError(b, err)
	addr := ln.(*Listener).Addr().String()
	node, err := cluster.NewNode(cluster.Config{
		Name:              name,
		Addr:              addr,
		Transport:         c.tr,
		Coordinator:       c.co.Node(name),
		Mux:               mux,
		AckTimeout:        clusterBenchTimeout,
		HeartbeatInterval: 50 * time.Millisecond,
		DialBackoff:       5 * time.Millisecond,
	})
	require.NoError(b, err)
	c.mu.Lock()
	c.nodes[name] = node
	c.listeners[name] = ln.(*Listener)
	c.mu.Unlock()
	return addr
}

// kill stops the node with this name with no drain.
func (c *clusterBenchNodes) kill(name string) {
	c.mu.Lock()
	node, ln := c.nodes[name], c.listeners[name]
	delete(c.nodes, name)
	delete(c.listeners, name)
	c.mu.Unlock()
	if node != nil {
		_ = node.Close()
		_ = ln.Close()
	}
}

func (c *clusterBenchNodes) node(name string) *cluster.Node {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nodes[name]
}

// startClusterBenchNodes starts the nodes, creates the shards and waits
// until the cluster is steady.  the nodes stop when the benchmark run ends.
func startClusterBenchNodes(b *testing.B, spec clusterBenchSpec) *clusterBenchNodes {
	b.Helper()
	c := &clusterBenchNodes{
		co:        cluster.NewMemCoordinator(20*time.Millisecond, clusterBenchTimeout),
		tr:        New(),
		rf:        spec.rf,
		nodes:     map[string]*cluster.Node{},
		listeners: map[string]*Listener{},
	}
	b.Cleanup(func() {
		for _, name := range []string{"n0", "n1", "n2", "n3", "n4"} {
			c.kill(name)
		}
		_ = c.co.Close()
	})
	var names []string
	for i := range spec.nodes {
		name := fmt.Sprintf("n%d", i)
		names = append(names, name)
		c.addrs = append(c.addrs, c.start(b, name))
	}
	require.NoError(b, c.co.Node("bench").Propose(context.Background(), cluster.Init{
		Shards: spec.shards,
		RF:     spec.rf,
		Nodes:  names,
	}))
	require.Eventually(b, c.steady, 30*time.Second, 2*time.Millisecond, "the cluster did not become steady, table %s", c.tableString())
	return c
}

func (c *clusterBenchNodes) table() *cluster.Table {
	tbl, _ := c.co.Node("bench").Table()
	return tbl
}

func (c *clusterBenchNodes) tableString() string {
	out, _ := json.Marshal(c.table())
	return string(out)
}

// steady reports whether every shard has an open primary and the table
// lists every replica of it as in sync.
func (c *clusterBenchNodes) steady() bool {
	tbl := c.table()
	if len(tbl.Shards) == 0 {
		return false
	}
	for i, row := range tbl.Shards {
		node := c.node(row.Primary)
		if node == nil || len(row.InSync) != c.rf {
			return false
		}
		if _, err := node.Primary(i); err != nil {
			return false
		}
	}
	return true
}

// clusterBenchAccount is one account with one function.
type clusterBenchAccount struct {
	account uuid.UUID
	env     uuid.UUID
	fn      uuid.UUID
}

func clusterBenchSource() constraintapi.LeaseSource {
	return constraintapi.LeaseSource{
		Service:           constraintapi.ServiceExecutor,
		Location:          constraintapi.CallerLocationItemLease,
		RunProcessingMode: constraintapi.RunProcessingModeBackground,
	}
}

// clusterBenchOp runs one acquire of one lease on the account's function
// concurrency and one release.  it returns the latency of the acquire call.
func clusterBenchOp(ctx context.Context, m constraintapi.CapacityManager, a *clusterBenchAccount, key string, level constraintapi.CommitLevel) (time.Duration, error) {
	req := &constraintapi.CapacityAcquireRequest{
		IdempotencyKey: key,
		AccountID:      a.account,
		EnvID:          a.env,
		FunctionID:     a.fn,
		Configuration: constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				FunctionConcurrency: clusterBenchLimit,
			},
		},
		Constraints: []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		Amount:               1,
		LeaseIdempotencyKeys: []string{key},
		CurrentTime:          time.Now(),
		Duration:             clusterBenchLease,
		MaximumLifetime:      time.Hour,
		Source:               clusterBenchSource(),
		CommitLevel:          level,
	}
	start := time.Now()
	resp, err := m.Acquire(ctx, req)
	took := time.Since(start)
	if err != nil {
		return 0, fmt.Errorf("acquire %s: %w", key, err)
	}
	if len(resp.Leases) != 1 {
		return 0, fmt.Errorf("acquire %s was denied, got %d leases", key, len(resp.Leases))
	}
	_, err = m.Release(ctx, &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: "r" + key,
		AccountID:      a.account,
		LeaseID:        resp.Leases[0].LeaseID,
		Source:         clusterBenchSource(),
	})
	if err != nil {
		return 0, fmt.Errorf("release %s: %w", key, err)
	}
	return took, nil
}

// clusterBenchWarm runs one acquire at level all and one release for every
// account.  afterwards the client has a connection to every primary, every
// primary has the constraint set of every account, and every in-sync
// follower has acknowledged a record.
func clusterBenchWarm(tb testing.TB, m constraintapi.CapacityManager, accounts []clusterBenchAccount) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
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
				if _, err := clusterBenchOp(ctx, m, &accounts[i], "warm"+strconv.Itoa(i), constraintapi.CommitLevelAll); err != nil {
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

// clusterBenchSetup starts the nodes of spec and a client, and warms every
// account.
func clusterBenchSetup(b *testing.B, spec clusterBenchSpec) (*clusterBenchNodes, *cluster.Client, []clusterBenchAccount) {
	b.Helper()
	c := startClusterBenchNodes(b, spec)
	cl, err := cluster.NewClient(cluster.ClientConfig{
		Transport: New(),
		Seeds:     c.addrs,
	})
	require.NoError(b, err)
	b.Cleanup(func() { _ = cl.Close() })

	accounts := make([]clusterBenchAccount, spec.accounts)
	tbl := c.table()
	covered := map[int]bool{}
	for i := range accounts {
		accounts[i] = clusterBenchAccount{
			account: uuid.New(),
			env:     uuid.New(),
			fn:      uuid.New(),
		}
		covered[tbl.ShardOf(accounts[i].account)] = true
	}
	require.Len(b, covered, spec.shards, "the accounts do not reach every shard")
	clusterBenchWarm(b, cl, accounts)
	require.True(b, c.steady(), "the cluster is not steady after the warm up, table %s", c.tableString())
	return c, cl, accounts
}

// BenchmarkPacedTCP is BenchmarkPaced of the cluster package over TCP on
// loopback.  the setups, the rates, the run and the metrics are the same.
// every node, the client and the callers run in this one process and share
// its CPUs.
//
//	go test ./pkg/constraintapi/cluster/tcp/ -run '^$' -bench 'PacedTCP' -benchtime=1x -cpu 8
func BenchmarkPacedTCP(b *testing.B) {
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
	rates := []struct {
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
	for _, s := range setups {
		for _, r := range rates {
			b.Run(s.name+"/rate="+r.name, func(b *testing.B) {
				spec := clusterBenchSpec{
					nodes:    s.nodes,
					shards:   1,
					rf:       s.nodes,
					accounts: 64,
					level:    s.level,
				}
				c, cl, accounts := clusterBenchSetup(b, spec)
				version := c.table().Version
				benchload.Run(b, benchload.Config{
					Rate:    r.rate,
					Callers: clusterBenchCallers * runtime.GOMAXPROCS(0),
					Op: func(ctx context.Context, k int64) (time.Duration, error) {
						return clusterBenchOp(ctx, cl, &accounts[int(k%int64(len(accounts)))], "p"+strconv.FormatInt(k, 36), spec.level)
					},
				})
				if got := c.table().Version; got != version {
					b.Fatalf("the table went from version %d to %d during the run, table %s", version, got, c.tableString())
				}
			})
		}
	}
}

func clusterBenchRun(b *testing.B, spec clusterBenchSpec) {
	c, cl, accounts := clusterBenchSetup(b, spec)
	version := c.table().Version

	var all []time.Duration
	var mu sync.Mutex
	var n atomic.Int64
	var failed atomic.Bool
	b.SetParallelism(clusterBenchCallers)
	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		var local []time.Duration
		for pb.Next() {
			i := n.Add(1)
			took, err := clusterBenchOp(ctx, cl, &accounts[int(i%int64(len(accounts)))], "k"+strconv.FormatInt(i, 36), spec.level)
			if err != nil {
				// one message is enough.  every caller fails the same way.
				if failed.CompareAndSwap(false, true) {
					b.Error(err)
				}
				continue
			}
			local = append(local, took)
		}
		mu.Lock()
		all = append(all, local...)
		mu.Unlock()
	})
	wall := time.Since(start)
	b.StopTimer()

	// a table change in the timed region means a follower left the in-sync
	// set or a primary moved.  the numbers are then not those of the RF and
	// the level in the name.
	if got := c.table().Version; got != version {
		b.Fatalf("the table went from version %d to %d during the run, table %s", version, got, c.tableString())
	}
	b.ReportMetric(float64(2*b.N)/wall.Seconds(), "req/s")
	if len(all) == 0 {
		return
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	at := func(p float64) float64 {
		return float64(all[int(float64(len(all)-1)*p)].Nanoseconds()) / 1e3
	}
	b.ReportMetric(at(0.5), "p50-µs")
	b.ReportMetric(at(0.99), "p99-µs")
}

// clusterBenchJoinLimit is how long one join may take before the run gives
// up.
const clusterBenchJoinLimit = 30 * time.Second

// BenchmarkJoinUnderLoadTCP is BenchmarkJoinUnderLoad of the cluster
// package over TCP on loopback.  one shard has RF 3 and is under acquire
// plus release load at level all.  one iteration kills a follower, starts
// it again with no state on a new port, and waits until the table lists it
// as in sync.  the metrics have the same names and meaning.  the socket
// buffer is small, so here the follower reads the log it lacks from the
// primary's tail and not from a queue in the transport.
//
//	go test ./pkg/constraintapi/cluster/tcp/ -run '^$' -bench 'JoinUnderLoadTCP' -benchtime=3x -cpu 8
func BenchmarkJoinUnderLoadTCP(b *testing.B) {
	for _, v := range []struct {
		name string
		rate int
	}{
		{
			name: "rate=200k",
			rate: 200_000,
		},
		{
			name: "rate=max",
			rate: 0,
		},
	} {
		b.Run(v.name, func(b *testing.B) {
			clusterBenchJoin(b, v.rate)
		})
	}
}

func clusterBenchJoin(b *testing.B, rate int) {
	c := startClusterBenchNodes(b, clusterBenchSpec{
		nodes:  3,
		shards: 1,
		rf:     3,
	})
	cl, err := cluster.NewClient(cluster.ClientConfig{
		Transport: New(),
		Seeds:     c.addrs,
	})
	require.NoError(b, err)
	b.Cleanup(func() { _ = cl.Close() })
	accounts := make([]clusterBenchAccount, 64)
	for i := range accounts {
		accounts[i] = clusterBenchAccount{
			account: uuid.New(),
			env:     uuid.New(),
			fn:      uuid.New(),
		}
	}
	clusterBenchWarm(b, cl, accounts)
	require.True(b, c.steady(), "the cluster is not steady after the warm up, table %s", c.tableString())

	callers := clusterBenchCallers * runtime.GOMAXPROCS(0)
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
				took, err := clusterBenchOp(ctx, cl, &accounts[int(k%int64(len(accounts)))], "j"+strconv.FormatInt(k, 36), constraintapi.CommitLevelAll)
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

	row := c.table().Shards[0]
	var follower string
	for _, name := range row.Replicas {
		if name != row.Primary {
			follower = name
			break
		}
	}
	primary := c.node(row.Primary)
	inSync := func() bool { return slices.Contains(c.table().Shards[0].InSync, follower) }

	var joinTime time.Duration
	var transfers, records, size uint64
	var loadCalls int64
	joined := 0
	b.ResetTimer()
	for range b.N {
		c.kill(follower)
		require.Eventually(b, func() bool { return !inSync() }, clusterBenchJoinLimit, time.Millisecond, "%s did not leave the in-sync set", follower)
		before := primary.Stats()
		callsBefore := calls.Load()
		joining.Store(true)
		start := time.Now()
		c.start(b, follower)
		ok := true
		for !inSync() {
			if time.Since(start) > clusterBenchJoinLimit {
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
			b.Logf("%s was not in sync after %v, the primary started %d transfers, table %s", follower, took, after.Transfers-before.Transfers, c.tableString())
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
