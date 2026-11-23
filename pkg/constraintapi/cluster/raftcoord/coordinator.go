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
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
)

// applyTimeout is the longest the leader waits for Raft on a forwarded
// request.
const applyTimeout = 10 * time.Second

// Peer is one member of the static Raft membership.
type Peer struct {
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
	// Peers is every member, this node included.  the first start
	// bootstraps Raft from it.  a follower finds the leader's Addr in it.
	Peers []Peer
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
	// addrs is the cluster transport address of every peer by Raft ID.
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
}

type report struct {
	at     time.Time
	bootID string
	lsn    map[int]uint64
}

// New starts Raft for one node and its controller loop.  a node with an
// empty store bootstraps the cluster from cfg.Peers.  every node can do this
// because all of them write the same membership.
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
	if !existing {
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
// leader forwards it to the leader.  Raft does not get it.
func (c *Coordinator) Report(ctx context.Context, r cluster.Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.raft.State() == raft.Leader {
		c.record(r)
		return nil
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

func (c *Coordinator) record(r cluster.Report) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// a report from a boot ID the table does not have comes from a process
	// that has not registered or that was replaced.  it must not keep the
	// node live.
	if info, ok := c.table.Nodes[r.Node]; !ok || info.BootID != r.BootID {
		return
	}
	c.reports[r.Node] = report{
		at:     time.Now(),
		bootID: r.BootID,
		lsn:    maps.Clone(r.LSN),
	}
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
