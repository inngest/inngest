package raftcoord

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/raft"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
)

// ProposeRequest is the first message on a stream that forwards a proposal
// to the Raft leader.  the leader answers with one Reply.
type ProposeRequest struct {
	// Command is the output of cluster.EncodeCommand.
	Command []byte
}

// ReportRequest is the first message on a stream that forwards a report to
// the Raft leader.  the leader answers with one Reply.
type ReportRequest struct {
	Report cluster.Report
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
)

// the offsets below are the wire IDs of the three messages.  the bodies are
// JSON because a forward is rare.
func init() {
	registerJSON[ProposeRequest](cluster.MessageIDCoordinator)
	registerJSON[ReportRequest](cluster.MessageIDCoordinator + 1)
	registerJSON[Reply](cluster.MessageIDCoordinator + 2)
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
	}
	return errors.New(r.Error)
}

// forward sends one request to the Raft leader and returns the error in its
// Reply.  the stream is closed when ctx ends, which stops a blocked Recv.
func (c *Coordinator) forward(ctx context.Context, req cluster.Message) error {
	_, id := c.raft.LeaderWithID()
	addr := c.addrs[id]
	if addr == "" || addr == c.cfg.Addr {
		return unavailable(errors.New("no raft leader"))
	}
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

// handlePropose applies a forwarded command on this node.  it never forwards
// again, so a request cannot go around between two nodes.
func (c *Coordinator) handlePropose(first cluster.Message, s cluster.Stream) {
	defer s.Close()
	ctx, cancel := context.WithTimeout(c.ctx, applyTimeout)
	defer cancel()
	_ = s.Send(replyOf(c.apply(ctx, first.(ProposeRequest).Command)))
}

func (c *Coordinator) handleReport(first cluster.Message, s cluster.Stream) {
	defer s.Close()
	if c.raft.State() != raft.Leader {
		_ = s.Send(replyOf(unavailable(raft.ErrNotLeader)))
		return
	}
	c.record(first.(ReportRequest).Report)
	_ = s.Send(Reply{})
}
