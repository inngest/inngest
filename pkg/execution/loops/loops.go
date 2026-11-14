// Package loops implements the state of a loop instance.  a loop is a function
// with one state record per key.  each input is one run, and the run reads
// the record at start and commits it back at the end.
package loops

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// Namespace is the kv namespace that prefixes every loop record.  kv
	// namespaces allow only letters, digits, '-', and '_'.
	Namespace = "loops"

	// OpenTTL is how long a record that was created by Open and never
	// committed stays in the store.
	OpenTTL = 24 * time.Hour

	// ReturnStepID is the step id under which a run saves its LoopReturn
	// with end set, before the instance is deleted.  the instance record is
	// gone after the end, so this step is the only sign that the run already
	// ended the loop, and a retry replays it in place of an SDK request.  SDK
	// step ids are hex hashes, so this id cannot collide with one.
	ReturnStepID = "inngest.loop.return"
)

// input kinds say why a run was started.  the executor derives one from
// the run's first event on every request and the SDK receives it in the
// loop request block.  nothing about the loop is stored on the run.
const (
	InputEvent        = "event"
	InputSubscription = "subscription"
	InputWake         = "wake"
	InputTick         = "tick"
)

// internal event names that start a tick run and a wake run.  the event
// data containsd the loop key, and for a wake the wake id.
const (
	EventTick = "inngest/loop.tick"
	EventWake = "inngest/loop.wake"
)

// Input names the loop instance a run belongs to and the input that
// started the run.  Kind is one of the input constants.  ID is the wake id
// or the subscription id and empty otherwise.
type Input struct {
	Kind string
	Key  LoopKey
	ID   string
}

// NewTickEvent returns the event that starts a tick run for key, with a
// random event id.
func NewTickEvent(key LoopKey) event.Event {
	return NewTickEventWithID(key, ulid.MustNew(ulid.Now(), rand.Reader))
}

// NewTickEventWithID returns the event that starts a tick run for key with
// the event id.  the timestamp is now, the time the input is handled.
func NewTickEventWithID(key LoopKey, id ulid.ULID) event.Event {
	return newLoopEvent(EventTick, id, map[string]any{
		"key": key,
	})
}

// NewWakeEvent returns the event that starts a wake run for key with the
// wake id, with a random event id.
func NewWakeEvent(key LoopKey, wakeID string) event.Event {
	return NewWakeEventWithID(key, wakeID, ulid.MustNew(ulid.Now(), rand.Reader))
}

// NewWakeEventWithID returns the event that starts a wake run for key with
// the wake id and the event id.  the timestamp is now, the time the input
// is handled.
func NewWakeEventWithID(key LoopKey, wakeID string, id ulid.ULID) event.Event {
	return newLoopEvent(EventWake, id, map[string]any{
		"key": key,
		"id":  wakeID,
	})
}

func newLoopEvent(name string, id ulid.ULID, data map[string]any) event.Event {
	return event.Event{
		ID:        id.String(),
		Name:      name,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}

// TimerEventID returns the event id for one tick or wake item.  the id is
// derived from the item's job id and its nonce, the run id that enqueued
// it, so every run of the same item builds the same id and the schedule
// idempotency key drops all but the first.  the item's scheduled time is
// not part of it, since a retry and a scavenge move the item.  the time
// part of the id is the nonce's time, or zero when the nonce is not a
// ulid.  that is the enqueuing run's time, not the time the timer fired,
// so the id sorts by enqueue time.  the event timestamp is the firing
// time, set when the item is handled.
func TimerEventID(jobID string, nonce string) ulid.ULID {
	var ms uint64
	if id, err := ulid.Parse(nonce); err == nil {
		ms = id.Time()
	}
	sum := sha256.Sum256([]byte(jobID + ":" + nonce))
	return ulid.MustNew(ms, bytes.NewReader(sum[:]))
}

// InputFromEvent derives the input from the event map of a run's first
// event.  a tick event and a wake event carry the key and the wake id in
// their data.  any other event is a trigger, and the key comes from the
// loop key expression.
func InputFromEvent(ctx context.Context, loop inngest.Loop, evt map[string]any) (Input, error) {
	name, _ := evt["name"].(string)
	switch name {
	case EventTick, EventWake:
	default:
		key, err := EvaluateKey(ctx, loop.Key, evt)
		if err != nil {
			return Input{}, err
		}
		return Input{
			Kind: InputEvent,
			Key:  key,
		}, nil
	}

	data, _ := evt["data"].(map[string]any)
	key, _ := data["key"].(string)
	if key == "" {
		return Input{}, fmt.Errorf("loop: event %s has no key", name)
	}
	in := Input{
		Kind: InputTick,
		Key:  key,
	}
	if name == EventWake {
		in.Kind = InputWake
		in.ID, _ = data["id"].(string)
		if in.ID == "" {
			return Input{}, fmt.Errorf("loop: event %s has no wake id", name)
		}
	}
	return in, nil
}

var (
	ErrNotFound = errors.New("loop: instance not found")
	// ErrNoKey means the key expression evaluated to nil, false, or the
	// empty string.  no run is created for the event.
	ErrNoKey = errors.New("loop: key expression evaluated to no value")
	// ErrConflict means the record moved or is gone since the run loaded
	// it.  the run fails without retry and the next tick reconciles.
	ErrConflict = errors.New("loop: state conflict")
)

// LoopKey represents the key for a loop, eg. the value of
// "event.data.user_id" from an event.
type LoopKey = string

// EvaluateKey evaluates the loop key expression against the event and
// returns the key as text.  the value is printed with %v, which is the same
// formatting the keyed function semaphore uses, so the two keys agree.
func EvaluateKey(ctx context.Context, expr string, evt map[string]any) (LoopKey, error) {
	val, err := expressions.Evaluate(ctx, expr, map[string]any{"event": evt})
	if err != nil {
		return "", fmt.Errorf("loop: evaluating key %q: %w", expr, err)
	}
	// a field that is missing from the event evaluates to false, and an
	// explicit null evaluates to structpb.NullValue.
	switch v := val.(type) {
	case nil, structpb.NullValue:
		return "", ErrNoKey
	case bool:
		if !v {
			return "", ErrNoKey
		}
	}
	key := fmt.Sprintf("%v", val)
	if key == "" {
		return "", ErrNoKey
	}
	return key, nil
}

// NewID returns a new loop ID.
func NewID(t domain.Tenant, fnID domain.FunctionID, key LoopKey) ID {
	return ID{Tenant: t, FunctionID: fnID, Key: key}
}

type ID struct {
	Tenant domain.Tenant
	// FunctionID is the function that owns the loop.
	FunctionID domain.FunctionID
	// Key is the key the loop was started with.  one function has one
	// instance per key.
	Key LoopKey
}

// InstanceID returns uuid5(FunctionID, Key).  two loops in one env keyed by
// the same value get different ids.  the same id names the instance's
// pauses, so loop.end can delete them by it.
func (id ID) InstanceID() uuid.UUID {
	return uuid.NewSHA1(id.FunctionID, []byte(id.Key))
}

// kvKey returns the kv key for the instance.  the key part is the instance
// id as 32 hex characters, which fits kv.MaxPartLen.
func (id ID) kvKey() kv.Key {
	iid := id.InstanceID()
	return kv.Key{
		Tenant:    id.Tenant,
		Namespace: Namespace,
		Key:       hex.EncodeToString(iid[:]),
	}
}

type Snapshot struct {
	ID ID
	// OpenedBy is the run that created the record.  an end that is replayed
	// after a crash compares it with the value it saved, so a record that a
	// later trigger created is left alone.
	OpenedBy domain.RunID
	// State is the marshalled state for the loop.  the state is one kv
	// entry.  it is nil until the first commit.
	State     json.RawMessage
	LastRunID *domain.RunID
	// Counter is the number of commits so far.  it also acts as a lock for
	// each update.  a record that Open created and no run has committed is
	// at 0.
	Counter int64
	// ExpiresAt is when the store drops the instance.  zero means never.
	ExpiresAt time.Time
}

// Commit represents a commit to the loops state, which happens at the end of a function
// run.  each loop is an iteration over a snapshot of state by a single run.
type Commit struct {
	ID    ID
	RunID domain.RunID
	// Counter is the Counter of the snapshot the run loaded.  the commit
	// happens only when the store is still at it.
	Counter int64
	// State is the new state.  nil leaves the state unchanged.  for empty
	// state write the JSON object "{}".
	State json.RawMessage
	// ExpiresAt is when the store drops the instance.  zero means never.
	ExpiresAt time.Time
	// OpenedBy is the OpenedBy of the snapshot the run loaded.  the kv
	// entry stores the whole record, so the commit writes it back.  zero
	// makes the commit read it from the record first.
	OpenedBy domain.RunID
}
