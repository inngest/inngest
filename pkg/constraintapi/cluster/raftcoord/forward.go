package raftcoord

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/hashicorp/raft"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
)

// ProposeRequest is the first message on a stream that forwards a proposal
// to the Raft leader.  the leader answers with one Reply.
type ProposeRequest struct {
	// Command is the output of cluster.EncodeCommand.
	Command []byte
	// Relay is true when the sender does not know the leader.  a receiver
	// that is not the leader then sends the request on to the leader.
	Relay bool `json:",omitempty"`
}

// ReportRequest is the first message on a stream that forwards a report to
// the Raft leader.  the leader answers with one Reply.
type ReportRequest struct {
	Report cluster.Report
	Relay  bool `json:",omitempty"`
}

// JoinRequest asks the Raft leader to add a node with no vote.
type JoinRequest struct {
	ID       string
	RaftAddr string
	Relay    bool `json:",omitempty"`
}

// ReplaceRequest forwards a ReplaceNode call to the Raft leader.
type ReplaceRequest struct {
	Old   string
	New   string
	Relay bool `json:",omitempty"`
}

// Reply is the leader's answer to a ProposeRequest or a ReportRequest.
type Reply struct {
	Status Status
	// Error is the error text.  it is empty when Status is StatusOK.
	Error string
}

// Status says which error a Reply carries.
type Status uint8

const (
	StatusOK Status = iota
	// StatusRejected means the command did not fit the table.  the sender
	// returns an error that wraps cluster.ErrRejected.
	StatusRejected
	// StatusUnavailable means the receiver was not the leader or Raft did
	// not apply the command in time.  the sender returns an error that
	// wraps cluster.ErrUnavailable.
	StatusUnavailable
	// StatusFailed is every other error.
	StatusFailed
	// StatusRemoved means the table replaced the sender.  the sender
	// returns an error that wraps cluster.ErrRemoved.
	StatusRemoved
)

// the offsets below are the wire IDs of the messages.  the bodies are JSON
// because a forward is rare.
func init() {
	registerJSON[ProposeRequest](cluster.MessageIDCoordinator)
	registerJSON[ReportRequest](cluster.MessageIDCoordinator + 1)
	registerJSON[Reply](cluster.MessageIDCoordinator + 2)
	registerJSON[JoinRequest](cluster.MessageIDCoordinator + 3)
	registerJSON[ReplaceRequest](cluster.MessageIDCoordinator + 4)
}

func registerJSON[M any](id uint8) {
	var zero M
	cluster.RegisterMessage(id, zero, func(dst []byte, m cluster.Message) ([]byte, error) {
		body, err := json.Marshal(m)
		return append(dst, body...), err
	}, func(b []byte) (cluster.Message, error) {
		var m M
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		return m, nil
	})
}

// remoteError is an error from the leader.  it keeps the leader's text and
// unwraps to the cluster error that the Status names.
type remoteError struct {
	text string
	base error
}

func (e *remoteError) Error() string { return e.text }
func (e *remoteError) Unwrap() error { return e.base }

func replyOf(err error) Reply {
	r := Reply{}
	switch {
	case err == nil:
		return r
	case errors.Is(err, cluster.ErrRemoved):
		r.Status = StatusRemoved
	case errors.Is(err, cluster.ErrRejected):
		r.Status = StatusRejected
	case errors.Is(err, cluster.ErrUnavailable):
		r.Status = StatusUnavailable
	default:
		r.Status = StatusFailed
	}
	r.Error = err.Error()
	return r
}

func (r Reply) err() error {
	switch r.Status {
	case StatusOK:
		return nil
	case StatusRejected:
		return &remoteError{
			text: r.Error,
			base: cluster.ErrRejected,
		}
	case StatusUnavailable:
		return &remoteError{
			text: r.Error,
			base: cluster.ErrUnavailable,
		}
	case StatusRemoved:
		return &remoteError{
			text: r.Error,
			base: cluster.ErrRemoved,
		}
	}
	return errors.New(r.Error)
}

// leaderAddr returns the cluster transport address of the Raft leader, or
// "" when this node knows no leader or is the leader.
func (c *Coordinator) leaderAddr() string {
	_, id := c.raft.LeaderWithID()
	if id == "" || string(id) == c.cfg.ID {
		return ""
	}
	if addr := c.addrs[id]; addr != "" {
		return addr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.table.Nodes[string(id)].Addr
}

// members returns every cluster transport address this node knows, without
// its own.
func (c *Coordinator) members() []string {
	out := slices.Clone(c.cfg.Join)
	add := func(addr string) {
		if addr != "" && addr != c.cfg.Addr && !slices.Contains(out, addr) {
			out = append(out, addr)
		}
	}
	for _, p := range c.cfg.Peers {
		add(p.Addr)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, info := range c.table.Nodes {
		add(info.Addr)
	}
	return out
}

// relayed returns req with Relay set.
func relayed(req cluster.Message, relay bool) cluster.Message {
	switch r := req.(type) {
	case ProposeRequest:
		r.Relay = relay
		return r
	case ReportRequest:
		r.Relay = relay
		return r
	case JoinRequest:
		r.Relay = relay
		return r
	case ReplaceRequest:
		r.Relay = relay
		return r
	}
	return req
}

// forward sends one request to the Raft leader and returns the error in its
// Reply.  a node that knows no leader sends it to the other members, and
// one of them sends it on.  a node that joins and a node that Raft removed
// know no leader.
func (c *Coordinator) forward(ctx context.Context, req cluster.Message) error {
	if addr := c.leaderAddr(); addr != "" {
		return c.send(ctx, addr, relayed(req, false))
	}
	err := unavailable(errors.New("no raft leader"))
	for _, addr := range c.members() {
		if err = c.send(ctx, addr, relayed(req, true)); !errors.Is(err, cluster.ErrUnavailable) {
			return err
		}
	}
	return err
}

// send sends one request to addr and returns the error in its Reply.  the
// stream is closed when ctx ends, which stops a blocked Recv.
func (c *Coordinator) send(ctx context.Context, addr string, req cluster.Message) error {
	s, err := c.cfg.Transport.Dial(ctx, addr)
	if err != nil {
		return unavailable(err)
	}
	defer s.Close()
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()

	if err := s.Send(req); err != nil {
		return unavailable(cmp.Or(ctx.Err(), err))
	}
	m, err := s.Recv()
	if err != nil {
		return unavailable(cmp.Or(ctx.Err(), err))
	}
	reply, ok := m.(Reply)
	if !ok {
		return unavailable(fmt.Errorf("reply has type %T", m))
	}
	return reply.err()
}

// serve answers one forwarded request.  the leader runs lead.  another node
// sends the request on to the leader when relay is set, and only then, so a
// request cannot go around between two nodes.
func (c *Coordinator) serve(s cluster.Stream, req cluster.Message, relay bool, lead func(ctx context.Context) error) {
	defer s.Close()
	ctx, cancel := context.WithTimeout(c.ctx, applyTimeout)
	defer cancel()
	if c.raft.State() == raft.Leader {
		_ = s.Send(replyOf(lead(ctx)))
		return
	}
	addr := c.leaderAddr()
	if !relay || addr == "" {
		_ = s.Send(replyOf(unavailable(raft.ErrNotLeader)))
		return
	}
	_ = s.Send(replyOf(c.send(ctx, addr, relayed(req, false))))
}

func (c *Coordinator) handlePropose(first cluster.Message, s cluster.Stream) {
	req := first.(ProposeRequest)
	c.serve(s, req, req.Relay, func(ctx context.Context) error {
		return c.apply(ctx, req.Command)
	})
}

func (c *Coordinator) handleReport(first cluster.Message, s cluster.Stream) {
	req := first.(ReportRequest)
	c.serve(s, req, req.Relay, func(context.Context) error {
		return c.record(req.Report)
	})
}

func (c *Coordinator) handleJoin(first cluster.Message, s cluster.Stream) {
	req := first.(JoinRequest)
	c.serve(s, req, req.Relay, func(context.Context) error {
		return c.addMember(req.ID, req.RaftAddr)
	})
}

func (c *Coordinator) handleReplace(first cluster.Message, s cluster.Stream) {
	req := first.(ReplaceRequest)
	c.serve(s, req, req.Relay, func(ctx context.Context) error {
		return c.ReplaceNode(ctx, req.Old, req.New)
	})
}
