package cluster

import (
	"errors"
	"fmt"
)

// Hello is the first message of a replication stream.  a follower sends it to
// the primary of a shard.  LastLSN is the last record the follower has.
type Hello struct {
	Shard   int
	Node    string
	Term    uint64
	LastLSN uint64
}

// Resume answers a Hello when the primary sends batches from LastLSN + 1.
type Resume struct {
	Term uint64
}

// StateBegin answers a Hello when the primary sends its whole state first.
// the follower drops the state it has.
type StateBegin struct {
	Term uint64
}

// StateRecords is one part of the state.  the follower applies it at once.
// Data is the AppendRecords encoding of Count records.  nobody changes Data
// after the message is sent.
type StateRecords struct {
	Count int
	Data  []byte
}

// StateEnd ends the state.  LSN is the log position the state belongs to.
// the batches that follow start at LSN + 1.
type StateEnd struct {
	LSN uint64
}

// Batch is a run of log records.  Data is the AppendRecords encoding of
// Count records and FirstLSN is the position of the first one.  FirstLSN can
// be below the record the follower asked for, because the primary sends
// whole chunks of its tail.  the follower skips what it has.  nobody changes
// Data after the message is sent, so the tail and the message share it.
type Batch struct {
	Term     uint64
	FirstLSN uint64
	Count    int
	Data     []byte
}

// Ack says the follower has every record up to LSN.
type Ack struct {
	LSN uint64
}

// Fence is sent by a node that was elected primary at Term to every other
// in-sync member.  the receiver stops its stream and accepts nothing from a
// lower term again.
type Fence struct {
	Shard int
	Term  uint64
}

// FenceReply answers a Fence.  OK is false when the receiver knows a higher
// term or has no complete state.  LastLSN is the last record it has.
type FenceReply struct {
	OK      bool
	LastLSN uint64
}

// TailRequest asks a fenced replica for every record from FromLSN on.
type TailRequest struct {
	Shard   int
	FromLSN uint64
}

// TailReply answers a TailRequest.  OK is false when the replica no longer
// has FromLSN.  Data is the AppendRecords encoding of Count records and
// FirstLSN is the position of the first one, at or below FromLSN.  one reply
// carries a bounded part of the tail, so the caller asks again until it has
// everything.
type TailReply struct {
	OK       bool
	FirstLSN uint64
	Count    int
	Data     []byte
}

// ErrNotPrimary means this node is not the open primary of the shard.  the
// caller reads the table again and retries.
var ErrNotPrimary = errors.New("cluster: not the primary")

// NotPrimaryError is ErrNotPrimary with what this node knows of the shard.
// Primary and Addr are empty when the table names no primary.
type NotPrimaryError struct {
	Shard   int
	Primary string
	Addr    string
	// Version is the table version the answer comes from.
	Version uint64
}

func (e *NotPrimaryError) Error() string {
	return fmt.Sprintf("cluster: not the primary of shard %d, table version %d names %q", e.Shard, e.Version, e.Primary)
}

// Is makes errors.Is(err, ErrNotPrimary) true.
func (e *NotPrimaryError) Is(target error) bool {
	return target == ErrNotPrimary
}
