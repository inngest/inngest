// Package raftcoord is the Raft backed cluster.Coordinator.  Raft stores only
// the shard table.  no capacity request goes through it.
package raftcoord

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
)

// applyTimeout is the longest the leader waits for Raft on a forwarded
// request.
const applyTimeout = 10 * time.Second

// Peer is one member of the Raft membership a cluster starts with.
type Peer struct {
	// ID is the node's name in the shard table.
	ID string
	// RaftAddr is where the node's Raft transport listens.
	RaftAddr string
	// Addr is where the node's cluster transport listens.
	Addr string
	// Voter is false for a node that gets the table and has no vote.
	Voter bool
}

// Config is what New needs to run one node.
type Config struct {
	ID string
	// Addr is this node's cluster transport address.
	Addr string
	// Peers is every member the cluster starts with, this node included.
	// the first start bootstraps Raft from it.  a node with Join set lists
	// only itself.
	Peers []Peer
	// Join is the cluster transport addresses of some members of a cluster
	// that runs.  a node with an empty store then does not bootstrap.  it
	// asks these members to add it to Raft with no vote.
	Join []string
	// DataDir gets the Raft log, the stable store and the snapshots.
	DataDir string
	// Transport dials the leader to forward a proposal or a report.
	Transport cluster.Transport
	// Mux gets the handlers for forwarded requests.  the caller passes its
	// Accept to Transport.Listen at Addr.
	Mux *cluster.Mux
	// HeartbeatInterval is how often nodes report.  the controller runs
	// every half of it.
	HeartbeatInterval time.Duration
	// HeartbeatTimeout is the age of a node's last report at which the
	// controller treats the node as dead.
	HeartbeatTimeout time.Duration

	// RaftTransport replaces the TCP transport.  tests use it.
	RaftTransport raft.Transport
	// InMemory keeps the Raft log, the stable store and the snapshots in
	// process memory.  tests use it.  DataDir is not read.
	InMemory bool

	// tune changes the Raft config before the start.  tests use it to
	// shorten the Raft timeouts.
	tune func(*raft.Config)
}

// Coordinator is one node's handle to the shard table in Raft.
type Coordinator struct {
	cfg Config
	// addrs is the cluster transport address of every peer by Raft ID.  a
	// node that joined later is not in it.  the table has its address.
	addrs map[raft.ServerID]string
	raft  *raft.Raft
	// store is nil when the Raft log is in memory.
	store *raftboltdb.BoltStore
	// ctx ends at Close.
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu sync.Mutex
	// table is replaced, never changed, so a caller of Table can read its
	// copy with no lock.
	table   *cluster.Table
	changed chan struct{}
	// leader is the last value from Raft's leadership channel.
	leader bool
	// reports is emptied on every leadership change of this node.
	reports map[string]report
	// graceUntil is when the controller starts to plan after this node
	// became the Raft leader.
	graceUntil time.Time

	// holdMembership stops reconcile on this node.  tests set it to stop a
	// leader between the table command and the Raft change.
	holdMembership atomic.Bool
}

type report struct {
	at     time.Time
	bootID string
	lsn    map[int]uint64
}

// New starts Raft for one node and its controller loop.  a node with an
// empty store bootstraps the cluster from cfg.Peers.  every node can do this
// because all of them write the same membership.  a node with cfg.Join does
// not bootstrap.  it asks a member to add it.
func New(cfg Config) (*Coordinator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Coordinator{
		cfg:     cfg,
		addrs:   map[raft.ServerID]string{},
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
		table:   &cluster.Table{},
		changed: make(chan struct{}),
		reports: map[string]report{},
	}
	trans := cfg.RaftTransport
	fail := func(err error) (*Coordinator, error) {
		cancel()
		if c.store != nil {
			_ = c.store.Close()
		}
		if t, ok := trans.(raft.WithClose); ok && cfg.RaftTransport == nil {
			_ = t.Close()
		}
		return nil, fmt.Errorf("raftcoord: %w", err)
	}

	var self *Peer
	var servers []raft.Server
	for i, p := range cfg.Peers {
		if p.ID == cfg.ID {
			self = &cfg.Peers[i]
		}
		suffrage := raft.Nonvoter
		if p.Voter {
			suffrage = raft.Voter
		}
		servers = append(servers, raft.Server{
			Suffrage: suffrage,
			ID:       raft.ServerID(p.ID),
			Address:  raft.ServerAddress(p.RaftAddr),
		})
		c.addrs[raft.ServerID(p.ID)] = p.Addr
	}
	if self == nil {
		return fail(fmt.Errorf("node %q is not in the peer list", cfg.ID))
	}

	conf := raft.DefaultConfig()
	conf.LocalID = raft.ServerID(cfg.ID)
	conf.LogOutput = io.Discard
	conf.LogLevel = "OFF"
	if cfg.tune != nil {
		cfg.tune(conf)
	}

	var logs raft.LogStore
	var stable raft.StableStore
	var snaps raft.SnapshotStore
	if cfg.InMemory {
		store := raft.NewInmemStore()
		logs, stable, snaps = store, store, raft.NewInmemSnapshotStore()
	} else {
		if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
			return fail(err)
		}
		store, err := raftboltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft.db"))
		if err != nil {
			return fail(err)
		}
		c.store = store
		logs, stable = store, store
		if snaps, err = raft.NewFileSnapshotStore(cfg.DataDir, 2, io.Discard); err != nil {
			return fail(err)
		}
	}
	if trans == nil {
		advertise, err := net.ResolveTCPAddr("tcp", self.RaftAddr)
		if err != nil {
			return fail(err)
		}
		if trans, err = raft.NewTCPTransport(self.RaftAddr, advertise, 3, 10*time.Second, io.Discard); err != nil {
			return fail(err)
		}
	}

	existing, err := raft.HasExistingState(logs, stable, snaps)
	if err != nil {
		return fail(err)
	}
	join := !existing && len(cfg.Join) > 0
	if !existing && !join {
		membership := raft.Configuration{Servers: servers}
		if err := raft.BootstrapCluster(conf, logs, stable, snaps, trans, membership); err != nil {
			return fail(err)
		}
	}
	if c.raft, err = raft.NewRaft(conf, (*fsm)(c), logs, stable, snaps, trans); err != nil {
		return fail(err)
	}

	cfg.Mux.Handle(ProposeRequest{}, c.handlePropose)
	cfg.Mux.Handle(ReportRequest{}, c.handleReport)
	cfg.Mux.Handle(JoinRequest{}, c.handleJoin)
	cfg.Mux.Handle(ReplaceRequest{}, c.handleReplace)
	if join {
		go c.join(self.RaftAddr)
	}
	go c.control()
	return c, nil
}

// Close stops the controller loop and Raft.  Raft closes its transport.  the
// handlers stay on the Mux and answer that the node is unavailable.
func (c *Coordinator) Close() error {
	c.cancel()
	err := c.raft.Shutdown().Error()
	<-c.done
	if c.store != nil {
		err = errors.Join(err, c.store.Close())
	}
	return err
}

// Table returns the newest applied table and a channel that is closed when a
// newer one is applied.  the caller must not change the table.
func (c *Coordinator) Table() (*cluster.Table, <-chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.table, c.changed
}

// Propose applies one command through Raft.  a node that is not the leader
// forwards it to the leader.  after an error that wraps
// cluster.ErrUnavailable the command can still apply later.
func (c *Coordinator) Propose(ctx context.Context, cmd cluster.Command) error {
	data, err := cluster.EncodeCommand(cmd)
	if err != nil {
		return err
	}
	if c.raft.State() == raft.Leader {
		return c.apply(ctx, data)
	}
	return c.forward(ctx, ProposeRequest{Command: data})
}

// Report records the report in the leader's memory.  a node that is not the
// leader forwards it to the leader.  Raft does not get it.  an error that
// wraps cluster.ErrRemoved means the table replaced the node.
func (c *Coordinator) Report(ctx context.Context, r cluster.Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.raft.State() == raft.Leader {
		return c.record(r)
	}
	return c.forward(ctx, ReportRequest{Report: r})
}

// apply appends one encoded command to the Raft log and waits until the
// state machine has run it or ctx ends.  it works only on the leader.
func (c *Coordinator) apply(ctx context.Context, data []byte) error {
	done := make(chan error, 1)
	go func() {
		f := c.raft.Apply(data, applyTimeout)
		if err := f.Error(); err != nil {
			done <- unavailable(err)
			return
		}
		err, _ := f.Response().(error)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return unavailable(ctx.Err())
	}
}

func unavailable(err error) error {
	return fmt.Errorf("%w: %w", cluster.ErrUnavailable, err)
}

func (c *Coordinator) record(r cluster.Report) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if by, ok := c.table.Removed[r.Node]; ok {
		return fmt.Errorf("%w by node %q", cluster.ErrRemoved, by)
	}
	// a report from a boot ID the table does not have comes from a process
	// that has not registered or that was replaced.  it must not keep the
	// node live.
	if info, ok := c.table.Nodes[r.Node]; !ok || info.BootID != r.BootID {
		return nil
	}
	c.reports[r.Node] = report{
		at:     time.Now(),
		bootID: r.BootID,
		lsn:    maps.Clone(r.LSN),
	}
	return nil
}

// control runs the controller every half heartbeat interval while this node
// is the Raft leader.
func (c *Coordinator) control() {
	defer close(c.done)
	tick := time.NewTicker(max(c.cfg.HeartbeatInterval/2, time.Microsecond))
	defer tick.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case leader := <-c.raft.LeaderCh():
			// a new leader has no reports.  it waits one heartbeat
			// timeout so that every live node has time to report.  without
			// the wait a Raft leader change moves primaries off live nodes.
			c.mu.Lock()
			c.leader = leader
			c.reports = map[string]report{}
			c.graceUntil = time.Now().Add(c.cfg.HeartbeatTimeout)
			c.mu.Unlock()
		case <-tick.C:
			c.plan(time.Now())
			c.reconcile()
		}
	}
}

// plan runs the controller once.
func (c *Coordinator) plan(now time.Time) {
	c.mu.Lock()
	if !c.leader || now.Before(c.graceUntil) {
		c.mu.Unlock()
		return
	}
	table := c.table
	live := func(node string) bool {
		r, ok := c.reports[node]
		return ok && r.bootID == table.Nodes[node].BootID && now.Sub(r.at) <= c.cfg.HeartbeatTimeout
	}
	lastLSN := func(node string, shard int) uint64 {
		return c.reports[node].lsn[shard]
	}
	cmds := cluster.Plan(table, live, lastLSN)
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.HeartbeatTimeout)
	defer cancel()
	for _, cmd := range cmds {
		if data, err := cluster.EncodeCommand(cmd); err == nil {
			_ = c.apply(ctx, data)
		}
	}
}

// Member is one server of the Raft membership.
type Member struct {
	ID    string
	Voter bool
}

// Members returns the Raft membership this node has.
func (c *Coordinator) Members() ([]Member, error) {
	f := c.raft.GetConfiguration()
	if err := f.Error(); err != nil {
		return nil, unavailable(err)
	}
	var out []Member
	for _, s := range f.Configuration().Servers {
		out = append(out, Member{
			ID:    string(s.ID),
			Voter: s.Suffrage == raft.Voter,
		})
	}
	return out, nil
}

// ReplaceNode gives the seats of the node old to the node by, which must
// have joined and registered.  old must be out of every in-sync set.  the
// table records the change in one command.  the Raft leader then removes old
// from Raft, and when old had a vote it gives by a vote.  a call that is
// repeated after any error is safe.
func (c *Coordinator) ReplaceNode(ctx context.Context, old, by string) error {
	if c.raft.State() != raft.Leader {
		return c.forward(ctx, ReplaceRequest{
			Old: old,
			New: by,
		})
	}
	f := c.raft.GetConfiguration()
	if err := f.Error(); err != nil {
		return unavailable(err)
	}
	voter, joined := false, false
	for _, s := range f.Configuration().Servers {
		switch string(s.ID) {
		case old:
			voter = s.Suffrage == raft.Voter
		case by:
			joined = true
		}
	}
	if !joined {
		return fmt.Errorf("%w: node %q has not joined Raft", cluster.ErrRejected, by)
	}
	data, err := cluster.EncodeCommand(cluster.ReplaceNode{
		Old:   old,
		New:   by,
		Voter: voter,
	})
	if err != nil {
		return err
	}
	return c.apply(ctx, data)
}

// reconcile makes the Raft membership follow the table, one change per
// call.  it removes a server the table has replaced.  after that it gives a
// vote to a server that the table says must have one.  the order matters.
// a vote for the new server while the dead one still has its vote makes
// four voters with two of them not proven, and the loss of one more stops
// Raft.  every Raft leader runs this, so a change that one leader started
// is finished by the next.
func (c *Coordinator) reconcile() {
	c.mu.Lock()
	leader, table := c.leader, c.table
	c.mu.Unlock()
	if !leader || len(table.Removed) == 0 || c.holdMembership.Load() {
		return
	}
	f := c.raft.GetConfiguration()
	if f.Error() != nil {
		return
	}
	servers := f.Configuration().Servers
	for _, s := range servers {
		if _, ok := table.Removed[string(s.ID)]; ok {
			_ = c.raft.RemoveServer(s.ID, 0, applyTimeout).Error()
			return
		}
	}
	for _, s := range servers {
		if s.Suffrage != raft.Voter && table.Nodes[string(s.ID)].Voter {
			_ = c.raft.AddVoter(s.ID, s.Address, 0, applyTimeout).Error()
			return
		}
	}
}

// join asks the members in cfg.Join to add this node to Raft until one of
// them does or the table has replaced this node.
func (c *Coordinator) join(raftAddr string) {
	req := JoinRequest{
		ID:       c.cfg.ID,
		RaftAddr: raftAddr,
		Relay:    true,
	}
	for {
		for _, addr := range c.cfg.Join {
			ctx, cancel := context.WithTimeout(c.ctx, applyTimeout)
			err := c.send(ctx, addr, req)
			cancel()
			if err == nil || errors.Is(err, cluster.ErrRemoved) {
				return
			}
		}
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(max(c.cfg.HeartbeatInterval, time.Millisecond)):
		}
	}
}

// addMember adds a node to Raft with no vote.  it works only on the leader.
func (c *Coordinator) addMember(id, raftAddr string) error {
	c.mu.Lock()
	by, removed := c.table.Removed[id]
	c.mu.Unlock()
	if removed {
		return fmt.Errorf("%w by node %q", cluster.ErrRemoved, by)
	}
	f := c.raft.GetConfiguration()
	if err := f.Error(); err != nil {
		return unavailable(err)
	}
	for _, s := range f.Configuration().Servers {
		// a voter with an empty store has lost the record of its votes, and
		// it can vote two times in one term.  it must come back under a new
		// name and replace the old one.
		if string(s.ID) == id && s.Suffrage == raft.Voter {
			return fmt.Errorf("%w: node %q is a voter and cannot join again", cluster.ErrRejected, id)
		}
	}
	if err := c.raft.AddNonvoter(raft.ServerID(id), raft.ServerAddress(raftAddr), 0, applyTimeout).Error(); err != nil {
		return unavailable(err)
	}
	return nil
}
