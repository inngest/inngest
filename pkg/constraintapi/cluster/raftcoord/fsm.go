package raftcoord

import (
	"encoding/json"
	"io"

	"github.com/hashicorp/raft"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
)

// fsm is the Coordinator as Raft's state machine.  the separate type keeps
// Apply, Snapshot and Restore out of the Coordinator's exported methods.
type fsm Coordinator

// Apply runs one committed command on a copy of the table and publishes the
// copy.  the leader's Propose returns the error.
func (f *fsm) Apply(l *raft.Log) any {
	cmd, err := cluster.DecodeCommand(l.Data)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	next := f.table.Clone()
	if err := next.Apply(cmd); err != nil {
		return err
	}
	f.publish(next)
	return nil
}

// Snapshot returns the current table.  no copy is made because a published
// table never changes.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return snapshot{table: f.table}, nil
}

// Restore replaces the table with the one in a snapshot.
func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	table := &cluster.Table{}
	if err := json.NewDecoder(rc).Decode(table); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.publish(table)
	return nil
}

// publish replaces the table and wakes every caller that waits on the
// channel from Table.  the caller has the mutex.
func (f *fsm) publish(table *cluster.Table) {
	f.table = table
	close(f.changed)
	f.changed = make(chan struct{})
}

// snapshot writes one table as JSON.
type snapshot struct {
	table *cluster.Table
}

func (s snapshot) Persist(sink raft.SnapshotSink) error {
	if err := json.NewEncoder(sink).Encode(s.table); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (snapshot) Release() {}
