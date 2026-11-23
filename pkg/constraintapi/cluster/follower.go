package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi/memory"
)

// followerHousekeepEvery is how often a follower frees what it no longer
// needs.  the work is split in steps of equal spacing, so the goroutine that
// acknowledges stops for one step at a time.
const (
	followerHousekeepEvery = 5 * time.Second
	followerHousekeepStep  = followerHousekeepEvery / memory.FollowerHousekeepSteps
)

var (
	// errStaleTerm means the stream comes from a primary that was replaced.
	errStaleTerm = errors.New("cluster: stream from a lower term")
	// errRebuild means this replica no longer matches the log and must take
	// the whole state again.
	errRebuild = errors.New("cluster: replica does not match the log")
)

// follower keeps one stream to the primary of a shard and applies what it
// receives.  one goroutine receives, acknowledges and applies, so a stopped
// follower has applied everything it acknowledged.
type follower struct {
	r       *replica
	primary string
	term    uint64

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu sync.Mutex
	s  Stream

	// buf is where the follower goroutine decodes records.  Apply does not
	// keep the slice, so every message uses it again.
	buf []memory.Record
}

func newFollower(r *replica, primary string, term uint64) *follower {
	ctx, cancel := context.WithCancel(context.Background())
	return &follower{
		r:       r,
		primary: primary,
		term:    term,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

func (f *follower) start() {
	go f.run()
}

// stop ends the stream and returns when the goroutine has applied the last
// message it received.
func (f *follower) stop() {
	f.cancel()
	f.mu.Lock()
	if f.s != nil {
		_ = f.s.Close()
	}
	f.mu.Unlock()
	<-f.done
}

// setStream records the open stream so stop can close it.  it returns false
// when the follower was stopped.
func (f *follower) setStream(s Stream) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.s = s
	return f.ctx.Err() == nil
}

func (f *follower) sleep() {
	select {
	case <-time.After(f.r.n.cfg.DialBackoff):
	case <-f.ctx.Done():
	}
}

func (f *follower) run() {
	defer close(f.done)
	n := f.r.n
	for f.ctx.Err() == nil {
		addr := n.addrOf(f.primary)
		if addr == "" {
			f.sleep()
			continue
		}
		s, err := n.cfg.Transport.Dial(f.ctx, addr)
		if err != nil {
			f.sleep()
			continue
		}
		if !f.setStream(s) {
			_ = s.Close()
			return
		}
		err = f.serve(s)
		_ = s.Close()
		f.setStream(nil)
		if errors.Is(err, errRebuild) {
			f.r.markBroken(f)
			return
		}
		f.sleep()
	}
}

// serve runs one stream until it fails.
func (f *follower) serve(s Stream) error {
	r := f.r
	if r.inTransfer() {
		// an earlier stream broke in the middle of the state.  what arrived
		// is no use.  start again from an empty manager at LSN 0.
		if _, err := r.beginTransfer(); err != nil {
			return err
		}
		r.endTransfer(0)
	}
	err := s.Send(Hello{
		Shard:   r.shard,
		Node:    r.n.cfg.Name,
		Term:    f.term,
		LastLSN: r.lastLSN.Load(),
	})
	if err != nil {
		return err
	}

	mgr := r.mgr.Load()
	housekept := time.Now()
	for {
		m, err := s.Recv()
		if err != nil {
			return err
		}
		switch v := m.(type) {
		case Resume:
			if v.Term < r.highestTerm() {
				return errStaleTerm
			}

		case StateBegin:
			if v.Term < r.highestTerm() {
				return errStaleTerm
			}
			// the primary sends the state only after the table removed this
			// node from the in-sync set, so nobody elects it meanwhile.
			mgr, err = r.beginTransfer()
			if err != nil {
				return err
			}

		case StateRecords:
			recs, err := f.decode(v.Count, v.Data)
			if err == nil {
				err = mgr.Apply(recs)
			}
			if err != nil {
				r.n.log().Error("applying the state failed", "shard", r.shard, "err", err)
				return errRebuild
			}

		case StateEnd:
			r.endTransfer(v.LSN)
			if err := s.Send(Ack{LSN: v.LSN}); err != nil {
				return err
			}

		case Batch:
			if v.Term < r.highestTerm() {
				return errStaleTerm
			}
			last := r.lastLSN.Load()
			if r.inTransfer() || v.FirstLSN > last+1 {
				r.n.log().Error("batch does not follow the log", "shard", r.shard, "first", v.FirstLSN, "last", last)
				return errRebuild
			}
			if v.FirstLSN+uint64(v.Count) <= last+1 {
				continue
			}
			recs, err := f.decode(v.Count, v.Data)
			if err != nil {
				r.n.log().Error("decoding a batch failed", "shard", r.shard, "err", err)
				return errRebuild
			}
			recs = r.keep(recs, v.FirstLSN, v.Data)
			last += uint64(len(recs))
			r.lastLSN.Store(last)
			// acknowledge on receipt.  the records are applied before this
			// goroutine does anything else, also when the send fails.
			sendErr := s.Send(Ack{LSN: last})
			if err := mgr.Apply(recs); err != nil {
				r.n.log().Error("applying a batch failed", "shard", r.shard, "err", err)
				return errRebuild
			}
			r.applied.Store(last)
			if sendErr != nil {
				return sendErr
			}
		}

		if now := time.Now(); now.Sub(housekept) > followerHousekeepStep {
			housekept = now
			mgr.FollowerHousekeepStep()
		}
	}
}

// decode decodes count records from data into the follower's buffer.  the
// result is valid until the next call.
func (f *follower) decode(count int, data []byte) ([]memory.Record, error) {
	if cap(f.buf) < count || cap(f.buf) > maxDrainBuffer {
		f.buf = make([]memory.Record, 0, min(count, maxPrealloc))
	}
	recs, err := DecodeRecords(f.buf[:0], data)
	if err != nil {
		return nil, err
	}
	if len(recs) != count {
		return nil, fmt.Errorf("cluster: message says %d records and has %d", count, len(recs))
	}
	f.buf = recs
	return recs, nil
}
