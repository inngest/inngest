package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/logger"
)

// Config configures one Node.
type Config struct {
	// Name is the node's ID in the table.  required.
	Name string
	// Addr is where other nodes dial this node.  it defaults to Name.
	Addr string

	Transport   Transport
	Coordinator Coordinator
	// Mux routes inbound streams.  when it is set the caller listens with
	// it, and the node only adds its handlers.  when it is nil the node
	// makes one and listens at Addr.
	Mux *Mux

	// AckTimeout is how long an in-sync follower may stay silent with
	// records outstanding before the primary proposes to remove it.
	AckTimeout time.Duration
	// FenceTimeout is how long a new primary waits for the other in-sync
	// members to answer its Fence before it proposes to remove them.
	FenceTimeout time.Duration
	// HeartbeatInterval is how often the node reports to the coordinator.
	HeartbeatInterval time.Duration
	// DialBackoff is the wait between two dials and between two proposals
	// for the same thing.
	DialBackoff time.Duration
	// TailBytes is how many bytes of encoded log records a replica keeps
	// for resumes, for a follower that joins, and for a new primary to
	// pull.  every replica of every shard has its own tail.
	TailBytes int
	// TailWindow is how long a replica keeps a record in its tail.  a
	// follower that joins must take the state and reach the end of the log
	// inside both limits, or it starts again.
	TailWindow time.Duration

	// ManagerOptions are passed to every memory.Manager the node builds.
	ManagerOptions []memory.Option

	// Logger receives the log lines of the node.  nil means the standard
	// logger.
	Logger logger.Logger
}

func (c *Config) defaults() error {
	if c.Name == "" {
		return errors.New("cluster: node name is empty")
	}
	if c.Transport == nil || c.Coordinator == nil {
		return errors.New("cluster: node needs a transport and a coordinator")
	}
	if c.Addr == "" {
		c.Addr = c.Name
	}
	if c.AckTimeout <= 0 {
		c.AckTimeout = 300 * time.Millisecond
	}
	if c.FenceTimeout <= 0 {
		c.FenceTimeout = c.AckTimeout
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 200 * time.Millisecond
	}
	if c.DialBackoff <= 0 {
		c.DialBackoff = 50 * time.Millisecond
	}
	if c.TailBytes <= 0 {
		c.TailBytes = 64 << 20
	}
	if c.TailWindow <= 0 {
		c.TailWindow = 10 * time.Second
	}
	return nil
}

// proposeTimeout bounds one proposal to the coordinator.
func (c *Config) proposeTimeout() time.Duration {
	return 4 * c.AckTimeout
}

// slowCommit is how long records may wait for the acknowledgement of a
// follower before the primary logs a warning.  it is below AckTimeout, so
// the warning comes before the primary proposes to remove the follower.
func (c *Config) slowCommit() time.Duration {
	return min(slowCommitAfter, c.AckTimeout/2)
}

// retryInterval is how often a replica looks at its row again with no
// change to it, so a failed step is tried again.
func (c *Config) retryInterval() time.Duration {
	return max(c.AckTimeout/2, 10*time.Millisecond)
}

// Node is one process of the cluster.  it runs a replica of every shard the
// table gives it and follows the table.
type Node struct {
	cfg    Config
	bootID string
	logr   logger.Logger

	listener io.Closer
	table    atomic.Pointer[Table]
	draining atomic.Bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu       sync.Mutex
	replicas map[int]*replica
	closed   bool

	stats nodeStats
}

// nodeStats counts the state transfers this node started as a primary.
type nodeStats struct {
	transfers    atomic.Uint64
	stateRecords atomic.Uint64
	stateBytes   atomic.Uint64
}

// NodeStats is a copy of a node's counters.
type NodeStats struct {
	// Transfers is how many state transfers this node started as a primary.
	Transfers uint64
	// StateRecords and StateBytes are the records and the encoded bytes of
	// every state it exported for them.
	StateRecords uint64
	StateBytes   uint64
}

// Stats returns the node's counters.
func (n *Node) Stats() NodeStats {
	return NodeStats{
		Transfers:    n.stats.transfers.Load(),
		StateRecords: n.stats.stateRecords.Load(),
		StateBytes:   n.stats.stateBytes.Load(),
	}
}

// NewNode starts a node.  it registers with a new boot ID, so the table
// removes any seat an earlier process with this name had.
func NewNode(cfg Config) (*Node, error) {
	if err := cfg.defaults(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	logr := cfg.Logger
	if logr == nil {
		logr = logger.StdlibLogger(ctx)
	}
	n := &Node{
		cfg:      cfg,
		bootID:   uuid.NewString(),
		logr:     logr.With("node", cfg.Name),
		ctx:      ctx,
		cancel:   cancel,
		replicas: map[int]*replica{},
	}
	n.table.Store(&Table{})

	mux := cfg.Mux
	if mux == nil {
		mux = NewMux()
	}
	mux.Handle(Hello{}, n.onHello)
	mux.Handle(Fence{}, n.onFence)
	mux.Handle(TailRequest{}, n.onTail)
	mux.Handle(RequestFrame{}, n.onRequests)
	mux.Handle(TableRequest{}, n.onTable)
	if cfg.Mux == nil {
		l, err := cfg.Transport.Listen(cfg.Addr, mux.Accept)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("cluster: listen at %s: %w", cfg.Addr, err)
		}
		n.listener = l
	}

	n.wg.Add(2)
	go n.run()
	go n.heartbeat()
	return n, nil
}

func (n *Node) log() logger.Logger {
	return n.logr
}

// run registers the node and then follows the table.
func (n *Node) run() {
	defer n.wg.Done()
	for {
		err := n.propose(n.ctx, RegisterNode{
			ID:     n.cfg.Name,
			Addr:   n.cfg.Addr,
			BootID: n.bootID,
		})
		if err == nil {
			break
		}
		select {
		case <-time.After(n.cfg.DialBackoff):
		case <-n.ctx.Done():
			return
		}
	}
	for {
		tbl, ch := n.cfg.Coordinator.Table()
		n.observe(tbl)
		select {
		case <-ch:
		case <-n.ctx.Done():
			return
		}
	}
}

// observe gives every replica its row.  a table that does not have this
// process's boot ID is older than the registration, or belongs to a newer
// process with the same name.  the node does not act on it.
func (n *Node) observe(tbl *Table) {
	if tbl.Nodes[n.cfg.Name].BootID != n.bootID {
		return
	}
	n.table.Store(tbl)
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return
	}
	for i := range tbl.Shards {
		row := tbl.Shards[i]
		if !slices.Contains(row.Replicas, n.cfg.Name) {
			continue
		}
		if r := n.replicas[i]; r != nil {
			r.setRow(tbl.Version, row)
			continue
		}
		r, err := newReplica(n, i, tbl.Version, row)
		if err != nil {
			n.logr.Error("building a replica failed", "shard", i, "err", err)
			continue
		}
		n.replicas[i] = r
		n.wg.Add(1)
		go func() {
			defer n.wg.Done()
			r.run(n.ctx)
		}()
	}
}

// heartbeat reports this node as alive, with the last LSN of every replica.
func (n *Node) heartbeat() {
	defer n.wg.Done()
	tick := time.NewTicker(n.cfg.HeartbeatInterval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
		case <-n.ctx.Done():
			return
		}
		lsn := map[int]uint64{}
		now := time.Now()
		n.mu.Lock()
		for i, r := range n.replicas {
			lsn[i] = r.lastLSN.Load()
			// an idle shard appends nothing, so its tail ages out here.
			r.tail.evict(now)
		}
		n.mu.Unlock()
		ctx, cancel := context.WithTimeout(n.ctx, n.cfg.proposeTimeout())
		_ = n.cfg.Coordinator.Report(ctx, Report{
			Node:   n.cfg.Name,
			BootID: n.bootID,
			LSN:    lsn,
		})
		cancel()
	}
}

func (n *Node) propose(ctx context.Context, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, n.cfg.proposeTimeout())
	defer cancel()
	return n.cfg.Coordinator.Propose(ctx, cmd)
}

// addrOf returns the address the table has for a node, or "".
func (n *Node) addrOf(name string) string {
	return n.table.Load().Nodes[name].Addr
}

// call opens a stream, sends req and returns the first answer.
func (n *Node) call(ctx context.Context, addr string, req Message) (Message, error) {
	if addr == "" {
		return nil, ErrDialRefused
	}
	s, err := n.cfg.Transport.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	// the close also ends the receive when ctx ends first.
	defer s.Close()
	if err := s.Send(req); err != nil {
		return nil, err
	}
	type answer struct {
		m   Message
		err error
	}
	ch := make(chan answer, 1)
	go func() {
		m, err := s.Recv()
		ch <- answer{
			m:   m,
			err: err,
		}
	}()
	select {
	case a := <-ch:
		return a.m, a.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// newManager builds an empty follower manager for a shard.
func (n *Node) newManager(shard int, nonce uint32) (*memory.Manager, error) {
	opts := []memory.Option{memory.WithShardName(fmt.Sprintf("%s-%d", n.cfg.Name, shard))}
	opts = append(opts, n.cfg.ManagerOptions...)
	opts = append(opts, memory.WithNonce(nonce), memory.WithFollower())
	return memory.NewManager(opts...)
}

func (n *Node) replica(shard int) *replica {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil
	}
	return n.replicas[shard]
}

func (n *Node) onHello(first Message, s Stream) {
	h := first.(Hello)
	r := n.replica(h.Shard)
	if r == nil {
		_ = s.Close()
		return
	}
	r.acceptHello(h, s)
}

func (n *Node) onFence(first Message, s Stream) {
	defer s.Close()
	f := first.(Fence)
	r := n.replica(f.Shard)
	if r == nil {
		return
	}
	lsn, ok := r.fence(f.Term)
	_ = s.Send(FenceReply{
		OK:      ok,
		LastLSN: lsn,
	})
}

func (n *Node) onTail(first Message, s Stream) {
	defer s.Close()
	req := first.(TailRequest)
	r := n.replica(req.Shard)
	if r == nil {
		return
	}
	from, count, data, ok := r.tail.read(req.FromLSN, maxTailReplyBytes)
	_ = s.Send(TailReply{
		OK:       ok,
		FirstLSN: from,
		Count:    count,
		Data:     data,
	})
}

// Primary returns the shard's manager while this node is its open primary.
// otherwise the error is a *NotPrimaryError.
func (n *Node) Primary(shard int) (*memory.Manager, error) {
	if r := n.replica(shard); r != nil && r.open.Load() {
		return r.mgr.Load(), nil
	}
	tbl := n.table.Load()
	e := &NotPrimaryError{
		Shard:   shard,
		Version: tbl.Version,
	}
	if shard >= 0 && shard < len(tbl.Shards) {
		e.Primary = tbl.Shards[shard].Primary
		e.Addr = tbl.Nodes[e.Primary].Addr
	}
	return nil, e
}

// Replica returns whatever manager this node has for the shard, or nil.  a
// follower's manager serves reads at most one replication lag behind.
func (n *Node) Replica(shard int) *memory.Manager {
	if r := n.replica(shard); r != nil {
		return r.mgr.Load()
	}
	return nil
}

// Table returns the newest table the node has acted on.
func (n *Node) Table() *Table {
	return n.table.Load()
}

// Drain is the planned stop.  it moves every primary seat this node has to
// an in-sync follower and takes the node out of every in-sync set, so a
// Close after it loses nothing and stalls nobody.  the node starts no new
// follower stream afterwards.
func (n *Node) Drain(ctx context.Context) error {
	n.draining.Store(true)
	n.mu.Lock()
	replicas := make([]*replica, 0, len(n.replicas))
	for _, r := range n.replicas {
		replicas = append(replicas, r)
	}
	n.mu.Unlock()
	sort.Slice(replicas, func(i, j int) bool { return replicas[i].shard < replicas[j].shard })

	var errs []error
	for _, r := range replicas {
		if err := r.movePrimary(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	for _, r := range replicas {
		if err := r.leaveInSync(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Close stops the node at once.  it proposes nothing, so the rest of the
// cluster sees the same as after a crash.
func (n *Node) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true
	replicas := n.replicas
	n.mu.Unlock()

	n.cancel()
	if n.listener != nil {
		_ = n.listener.Close()
	}
	// the replica goroutines end first, so none of them swaps a manager in
	// after its replica was closed.
	n.wg.Wait()
	for _, r := range replicas {
		r.close()
	}
	return nil
}
