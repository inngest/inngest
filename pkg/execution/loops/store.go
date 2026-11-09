package loops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/execution/state/kv"
)

type Store interface {
	// Open returns the instance for a declared trigger, and creates it at
	// counter 0 when it is missing, with the run id as OpenedBy.  a record
	// that never commits expires after OpenTTL.
	Open(context.Context, ID, domain.RunID) (Snapshot, error)
	// Load returns the instance for a tick, a wake, a subscription, or the
	// API.  after loop.end it returns ErrNotFound.  the executor then
	// discards the input without starting a run, so a late timer or a late
	// subscription match cannot recreate an ended instance.
	Load(context.Context, ID) (Snapshot, error)
	// Commit writes the run's state, as long as the Commit's counter matches
	// the existing counter in the state store.
	//
	// a nil State means unchanged and keeps the stored state.  a zero
	// OpenedBy is read from the record first, which costs one read.  a second
	// commit by the same run returns the current snapshot and no error.
	// ErrConflict means the record moved or is gone, and the run fails
	// without retry.
	Commit(context.Context, Commit) (Snapshot, error)
	// End deletes the instance as long as the snapshot's counter matches
	// the stored counter.  ErrConflict means the record moved.  ErrNotFound
	// means the record is already gone.
	End(context.Context, Snapshot) error
}

// openAttempts bounds the create then read loop in Open.  each extra pass
// covers an End that ran between the two calls.
const openAttempts = 3

// record is the value stored in the kv entry.
type record struct {
	OpenedBy  domain.RunID    `json:"o"`
	LastRunID *domain.RunID   `json:"r,omitempty"`
	ExpiresAt time.Time       `json:"e,omitzero"`
	State     json.RawMessage `json:"s"`
}

// store keeps one kv entry per instance.  the kv version is the lock.  Open
// writes the entry at kv version 1, which is Counter 0, and every Commit
// adds one to both, so Counter is always the kv version minus one.
type store struct {
	kv kv.Store
}

func NewStore(kv kv.Store) Store {
	return &store{kv: kv}
}

func (s *store) Open(ctx context.Context, id ID, runID domain.RunID) (Snapshot, error) {
	key := id.kvKey()
	var lastErr error
	for range openAttempts {
		rec := record{
			OpenedBy:  runID,
			ExpiresAt: time.Now().Add(OpenTTL),
			State:     json.RawMessage("null"),
		}
		e, err := s.write(ctx, key, rec, kv.WithExpected(0), kv.WithTTL(OpenTTL))
		if err == nil {
			return toSnapshot(id, e, rec)
		}
		if !errors.Is(err, kv.ErrConflict) {
			return Snapshot{}, err
		}
		snap, err := s.Load(ctx, id)
		if err == nil {
			return snap, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Snapshot{}, err
		}
		lastErr = err
	}
	return Snapshot{}, fmt.Errorf("loop: open gave up after %d attempts: %w", openAttempts, lastErr)
}

func (s *store) Load(ctx context.Context, id ID) (Snapshot, error) {
	e, err := s.kv.Get(ctx, id.kvKey())
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, err
	}
	var rec record
	if err := json.Unmarshal(e.Value, &rec); err != nil {
		return Snapshot{}, fmt.Errorf("loop: decoding record: %w", err)
	}
	return toSnapshot(id, e, rec)
}

func (s *store) Commit(ctx context.Context, c Commit) (Snapshot, error) {
	key := c.ID.kvKey()
	runID := c.RunID
	rec := record{
		OpenedBy:  c.OpenedBy,
		LastRunID: &runID,
		ExpiresAt: c.ExpiresAt,
		State:     c.State,
	}

	if rec.State == nil || rec.OpenedBy == (domain.RunID{}) {
		// unchanged state and an unknown opener still have to be written
		// back, since the kv entry holds the whole record.  read the
		// current bytes first.
		cur, err := s.Load(ctx, c.ID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return Snapshot{}, ErrConflict
			}
			return Snapshot{}, err
		}
		if cur.Counter != c.Counter {
			return s.resolveConflict(cur, c)
		}
		if rec.State == nil {
			rec.State = cur.State
		}
		if rec.State == nil {
			rec.State = json.RawMessage("null")
		}
		rec.OpenedBy = cur.OpenedBy
	}

	opts := []kv.Option{kv.WithExpected(c.Counter + 1)}
	if !c.ExpiresAt.IsZero() {
		opts = append(opts, kv.WithTTL(ttlUntil(c.ExpiresAt)))
	}
	e, err := s.write(ctx, key, rec, opts...)
	if err == nil {
		return toSnapshot(c.ID, e, rec)
	}
	if !errors.Is(err, kv.ErrConflict) {
		return Snapshot{}, err
	}

	cur, err := s.Load(ctx, c.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Snapshot{}, ErrConflict
		}
		return Snapshot{}, err
	}
	return s.resolveConflict(cur, c)
}

// resolveConflict returns the current snapshot when the same run already
// committed it, and ErrConflict otherwise.
func (s *store) resolveConflict(cur Snapshot, c Commit) (Snapshot, error) {
	if cur.LastRunID != nil && *cur.LastRunID == c.RunID && cur.Counter == c.Counter+1 {
		return cur, nil
	}
	return Snapshot{}, ErrConflict
}

func (s *store) End(ctx context.Context, snap Snapshot) error {
	err := s.kv.Delete(ctx, snap.ID.kvKey(), kv.WithExpected(snap.Counter+1))
	if !errors.Is(err, kv.ErrConflict) {
		return err
	}
	// a missing key is at kv version 0, so the delete reports a conflict
	// for it as well.  a read tells the two apart.
	_, err = s.Load(ctx, snap.ID)
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case err != nil:
		return err
	}
	return ErrConflict
}

// write encodes rec and stores it under key.  times are stored in utc
// with no monotonic reading, so a snapshot read back equals the one the
// writer returned.
func (s *store) write(ctx context.Context, key kv.Key, rec record, opts ...kv.Option) (kv.Entry, error) {
	rec.ExpiresAt = rec.ExpiresAt.UTC()
	val, err := json.Marshal(rec)
	if err != nil {
		return kv.Entry{}, fmt.Errorf("loop: encoding record: %w", err)
	}
	e := kv.Entry{
		Key:   key,
		Value: val,
	}
	return s.kv.Set(ctx, e, opts...)
}

// toSnapshot builds the snapshot for an entry.  a stored "null" state is
// returned as nil so the executor sends the SDK no state.
func toSnapshot(id ID, e kv.Entry, rec record) (Snapshot, error) {
	if e.Version < 1 {
		return Snapshot{}, fmt.Errorf("loop: kv version %d below 1", e.Version)
	}
	state := rec.State
	if string(state) == "null" {
		state = nil
	}
	return Snapshot{
		ID:        id,
		OpenedBy:  rec.OpenedBy,
		State:     state,
		LastRunID: rec.LastRunID,
		Counter:   e.Version - 1,
		ExpiresAt: rec.ExpiresAt.UTC(),
	}, nil
}

// ttlUntil returns the time left until t.  a time in the past expires the
// record as soon as the store allows.
func ttlUntil(t time.Time) time.Duration {
	d := time.Until(t)
	if d < time.Millisecond {
		return time.Millisecond
	}
	return d
}
