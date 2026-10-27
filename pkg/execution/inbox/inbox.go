// Package inbox is a durable per-key message queue used by step.subscribe.
//
// Messages are appended with increasing sequence numbers. A reader (a
// WaitForSignal pause created by step.subscribe) is delivered one message at a
// time; the acked cursor records the last sequence handed to a reader, so a
// message is delivered at most once per cursor position and survives the gaps
// between runs of the same thread.
package inbox

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// SignalPrefix prefixes the signal ID of a pause created by step.subscribe.
const SignalPrefix = "inbox:"

// Message is one queued message.
type Message struct {
	Seq  int64           `json:"seq"`
	Data json.RawMessage `json:"data"`
}

// Store holds inbox messages and cursors.
type Store interface {
	// Append adds a message and returns its sequence number (starting at 1).
	Append(ws uuid.UUID, key string, data json.RawMessage) int64
	// Get returns the message with the given sequence.
	Get(ws uuid.UUID, key string, seq int64) (Message, bool)
	// Head is the latest sequence appended, or 0.
	Head(ws uuid.UUID, key string) int64
	// Acked is the latest sequence delivered to a reader, or 0.
	Acked(ws uuid.UUID, key string) int64
	// Claim moves the acked cursor from `from` to `from+1` if it is still
	// `from`. It reports whether the caller now owns delivery of that message.
	Claim(ws uuid.UUID, key string, from int64) bool
	// Unclaim moves the cursor back from `from+1` to `from` after a failed
	// delivery.
	Unclaim(ws uuid.UUID, key string, from int64)
	// MarkHandoff records that a next run was started for the inbox head, and
	// reports whether this is the first time (idempotent on key + head).
	MarkHandoff(ws uuid.UUID, key string, head int64) bool
}

type slot struct {
	msgs    []Message
	acked   int64
	handoff int64
}

type memStore struct {
	mu    sync.Mutex
	slots map[string]*slot
}

// NewMemoryStore returns an in-process Store. It is the development
// implementation; a shared store is required for multiple executors.
func NewMemoryStore() Store {
	return &memStore{slots: map[string]*slot{}}
}

func (m *memStore) slot(ws uuid.UUID, key string) *slot {
	id := ws.String() + "/" + key
	s, ok := m.slots[id]
	if !ok {
		s = &slot{}
		m.slots[id] = s
	}
	return s
}

func (m *memStore) Append(ws uuid.UUID, key string, data json.RawMessage) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.slot(ws, key)
	seq := int64(len(s.msgs)) + 1
	s.msgs = append(s.msgs, Message{Seq: seq, Data: data})
	return seq
}

func (m *memStore) Get(ws uuid.UUID, key string, seq int64) (Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.slot(ws, key)
	if seq < 1 || seq > int64(len(s.msgs)) {
		return Message{}, false
	}
	return s.msgs[seq-1], true
}

func (m *memStore) Head(ws uuid.UUID, key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.slot(ws, key).msgs))
}

func (m *memStore) Acked(ws uuid.UUID, key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.slot(ws, key).acked
}

func (m *memStore) Claim(ws uuid.UUID, key string, from int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.slot(ws, key)
	if s.acked != from || from >= int64(len(s.msgs)) {
		return false
	}
	s.acked = from + 1
	return true
}

func (m *memStore) Unclaim(ws uuid.UUID, key string, from int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.slot(ws, key)
	if s.acked == from+1 {
		s.acked = from
	}
}

var def = NewMemoryStore()

// Default is the process-wide store used by the API and the executor.
func Default() Store {
	return def
}

// Resumer resumes the pause waiting on a signal and reports whether one matched.
type Resumer func(signalID string, data json.RawMessage) (bool, error)

// Deliver hands the next undelivered message to a waiting reader, if both
// exist. It is safe to call concurrently and repeatedly.
func Deliver(s Store, ws uuid.UUID, key string, resume Resumer) (bool, error) {
	from := s.Acked(ws, key)
	msg, ok := s.Get(ws, key, from+1)
	if !ok {
		return false, nil
	}
	if !s.Claim(ws, key, from) {
		return false, nil
	}
	data, err := json.Marshal(msg)
	if err != nil {
		s.Unclaim(ws, key, from)
		return false, err
	}
	matched, err := resume(SignalPrefix+key, data)
	if err != nil || !matched {
		s.Unclaim(ws, key, from)
		return false, err
	}
	return true, nil
}

func (m *memStore) MarkHandoff(ws uuid.UUID, key string, head int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.slot(ws, key)
	if s.handoff >= head {
		return false
	}
	s.handoff = head
	return true
}
