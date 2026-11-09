package loops

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/execution/state/kv/kvredis"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func newStore(t testing.TB) (Store, *miniredis.Miniredis) {
	t.Helper()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return NewStore(kvredis.New(rc, "")), r
}

func testID() ID {
	return ID{
		Tenant:     domain.NewTenant(uuid.New(), uuid.New()),
		FunctionID: uuid.New(),
		Key:        "app-1",
	}
}

func TestInstanceID(t *testing.T) {
	id := testID()
	require.Equal(t, id.InstanceID(), id.InstanceID())

	other := id
	other.Key = "app-2"
	require.NotEqual(t, id.InstanceID(), other.InstanceID())

	fn := id
	fn.FunctionID = uuid.New()
	require.NotEqual(t, id.InstanceID(), fn.InstanceID())

	k := id.kvKey()
	require.NoError(t, k.Validate())
	require.Equal(t, Namespace, k.Namespace)
	require.Len(t, k.Key, 32)
}

func TestOpenCreatesOnce(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t)
	id := testID()

	opener := ulid.Make()
	snap, err := s.Open(ctx, id, opener)
	require.NoError(t, err)
	require.Equal(t, id, snap.ID)
	require.Equal(t, opener, snap.OpenedBy)
	require.Nil(t, snap.State)
	require.Nil(t, snap.LastRunID)
	require.Equal(t, int64(0), snap.Counter)
	require.WithinDuration(t, time.Now().Add(OpenTTL), snap.ExpiresAt, time.Minute)
	require.Equal(t, OpenTTL, r.TTL(id.kvKey().String()))

	// a second open keeps the first opener.
	again, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)
	require.Equal(t, snap, again)

	// a commit writes the opener back.
	run := ulid.Make()
	committed, err := s.Commit(ctx, Commit{
		ID:       id,
		RunID:    run,
		Counter:  snap.Counter,
		State:    json.RawMessage(`{"n":1}`),
		OpenedBy: snap.OpenedBy,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), committed.Counter)
	require.Equal(t, opener, committed.OpenedBy)

	// a commit with unchanged state reads the opener from the record.
	committed, err = s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: committed.Counter,
	})
	require.NoError(t, err)
	require.Equal(t, opener, committed.OpenedBy)

	// a commit without an opener reads it from the record as well.
	committed, err = s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: committed.Counter,
		State:   json.RawMessage(`{"n":2}`),
	})
	require.NoError(t, err)
	require.Equal(t, opener, committed.OpenedBy)
	require.JSONEq(t, `{"n":2}`, string(committed.State))

	// open on an existing instance returns it unchanged.
	again, err = s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)
	require.Equal(t, committed, again)

	// an end and a new open is a new opener.
	require.NoError(t, s.End(ctx, committed))
	reopener := ulid.Make()
	fresh, err := s.Open(ctx, id, reopener)
	require.NoError(t, err)
	require.Equal(t, reopener, fresh.OpenedBy)
	require.Equal(t, int64(0), fresh.Counter)
}

func TestLoadMissing(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	_, err := s.Load(ctx, id)
	require.ErrorIs(t, err, ErrNotFound)

	snap, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)
	loaded, err := s.Load(ctx, id)
	require.NoError(t, err)
	require.Equal(t, snap, loaded)

	require.NoError(t, s.End(ctx, snap))
	_, err = s.Load(ctx, id)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCommit(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	snap, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)

	run1 := ulid.Make()
	c1, err := s.Commit(ctx, Commit{
		ID:      id,
		RunID:   run1,
		Counter: snap.Counter,
		State:   json.RawMessage(`{"n":1}`),
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), c1.Counter)
	require.JSONEq(t, `{"n":1}`, string(c1.State))
	require.NotNil(t, c1.LastRunID)
	require.Equal(t, run1, *c1.LastRunID)
	require.True(t, c1.ExpiresAt.IsZero())

	loaded, err := s.Load(ctx, id)
	require.NoError(t, err)
	require.Equal(t, c1, loaded)

	// an unchanged commit bumps the counter and keeps the state.
	run2 := ulid.Make()
	c2, err := s.Commit(ctx, Commit{
		ID:      id,
		RunID:   run2,
		Counter: c1.Counter,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), c2.Counter)
	require.JSONEq(t, `{"n":1}`, string(c2.State))
	require.Equal(t, run2, *c2.LastRunID)

	loaded, err = s.Load(ctx, id)
	require.NoError(t, err)
	require.Equal(t, c2, loaded)
}

func TestCommitSameRunRetry(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	snap, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)

	run := ulid.Make()
	c := Commit{
		ID:      id,
		RunID:   run,
		Counter: snap.Counter,
		State:   json.RawMessage(`{"n":1}`),
	}
	first, err := s.Commit(ctx, c)
	require.NoError(t, err)

	second, err := s.Commit(ctx, c)
	require.NoError(t, err)
	require.Equal(t, first, second)

	// the retry of an unchanged commit is also a no-op.
	c.State = nil
	third, err := s.Commit(ctx, c)
	require.NoError(t, err)
	require.Equal(t, first, third)

	// a retry after another run committed is a conflict.
	other := ulid.Make()
	_, err = s.Commit(ctx, Commit{
		ID:      id,
		RunID:   other,
		Counter: first.Counter,
		State:   json.RawMessage(`{"n":2}`),
	})
	require.NoError(t, err)
	_, err = s.Commit(ctx, c)
	require.ErrorIs(t, err, ErrConflict)
}

func TestCommitStaleCounter(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	snap, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)

	_, err = s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: snap.Counter + 1,
		State:   json.RawMessage(`{"n":1}`),
	})
	require.ErrorIs(t, err, ErrConflict)

	_, err = s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: snap.Counter + 1,
	})
	require.ErrorIs(t, err, ErrConflict)

	loaded, err := s.Load(ctx, id)
	require.NoError(t, err)
	require.Equal(t, snap, loaded)

	// a commit to an ended instance is a conflict, not a create.
	require.NoError(t, s.End(ctx, snap))
	_, err = s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: snap.Counter,
		State:   json.RawMessage(`{"n":1}`),
	})
	require.ErrorIs(t, err, ErrConflict)
	_, err = s.Load(ctx, id)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestEndStaleCounter(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	snap, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)

	committed, err := s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: snap.Counter,
		State:   json.RawMessage(`{"n":1}`),
	})
	require.NoError(t, err)

	require.ErrorIs(t, s.End(ctx, snap), ErrConflict)
	_, err = s.Load(ctx, id)
	require.NoError(t, err)

	require.NoError(t, s.End(ctx, committed))
	_, err = s.Load(ctx, id)
	require.ErrorIs(t, err, ErrNotFound)

	// ending twice reports the record as gone.
	require.ErrorIs(t, s.End(ctx, committed), ErrNotFound)
}

func TestEndMissing(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	require.ErrorIs(t, s.End(ctx, Snapshot{
		ID:      id,
		Counter: 0,
	}), ErrNotFound)
	require.ErrorIs(t, s.End(ctx, Snapshot{
		ID:      id,
		Counter: 3,
	}), ErrNotFound)
}

func TestCommitTTL(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t)
	id := testID()
	key := id.kvKey().String()

	snap, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)

	expires := time.Now().Add(time.Hour)
	c1, err := s.Commit(ctx, Commit{
		ID:        id,
		RunID:     ulid.Make(),
		Counter:   snap.Counter,
		State:     json.RawMessage(`{"n":1}`),
		ExpiresAt: expires,
	})
	require.NoError(t, err)
	require.WithinDuration(t, expires, c1.ExpiresAt, time.Second)
	require.WithinDuration(t, expires, time.Now().Add(r.TTL(key)), time.Second)

	r.FastForward(30 * time.Minute)
	_, err = s.Load(ctx, id)
	require.NoError(t, err)

	// a commit without an expiry makes the instance live forever.
	c2, err := s.Commit(ctx, Commit{
		ID:      id,
		RunID:   ulid.Make(),
		Counter: c1.Counter,
	})
	require.NoError(t, err)
	require.True(t, c2.ExpiresAt.IsZero())
	require.Zero(t, r.TTL(key))

	c3, err := s.Commit(ctx, Commit{
		ID:        id,
		RunID:     ulid.Make(),
		Counter:   c2.Counter,
		ExpiresAt: time.Now().Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), c3.Counter)
	r.FastForward(2 * time.Minute)
	_, err = s.Load(ctx, id)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestKVIsolation(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	id := testID()

	_, err := s.Open(ctx, id, ulid.Make())
	require.NoError(t, err)

	other := id
	other.Key = "app-2"
	_, err = s.Load(ctx, other)
	require.ErrorIs(t, err, ErrNotFound)

	require.NotEqual(t, id.kvKey(), other.kvKey())
}

func TestEvaluateKey(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		expr string
		evt  map[string]any
		want string
		err  error
	}{
		{
			name: "string",
			expr: "event.data.id",
			evt:  map[string]any{"data": map[string]any{"id": "user-1"}},
			want: "user-1",
		},
		{
			name: "number prints with %v",
			expr: "event.data.id",
			evt:  map[string]any{"data": map[string]any{"id": 42}},
			want: "42",
		},
		{
			name: "missing field",
			expr: "event.data.id",
			evt:  map[string]any{"data": map[string]any{}},
			err:  ErrNoKey,
		},
		{
			name: "null field",
			expr: "event.data.id",
			evt:  map[string]any{"data": map[string]any{"id": nil}},
			err:  ErrNoKey,
		},
		{
			name: "empty string",
			expr: "event.data.id",
			evt:  map[string]any{"data": map[string]any{"id": ""}},
			err:  ErrNoKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateKey(ctx, tt.expr, tt.evt)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewLoopEvents(t *testing.T) {
	tick := NewTickEvent("user-1")
	require.Equal(t, EventTick, tick.Name)
	require.Equal(t, map[string]any{"key": "user-1"}, tick.Data)
	_, err := ulid.Parse(tick.ID)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), time.UnixMilli(tick.Timestamp), 5*time.Second)
	require.True(t, tick.IsInternal())

	wake := NewWakeEvent("user-1", "wake-1")
	require.Equal(t, EventWake, wake.Name)
	require.Equal(t, map[string]any{
		"key": "user-1",
		"id":  "wake-1",
	}, wake.Data)
	_, err = ulid.Parse(wake.ID)
	require.NoError(t, err)
	require.NotEqual(t, tick.ID, wake.ID)
	require.WithinDuration(t, time.Now(), time.UnixMilli(wake.Timestamp), 5*time.Second)
	require.True(t, wake.IsInternal())

	// a given id names the event, and the timestamp is still now, not the
	// time part of the id.
	old := ulid.MustNew(ulid.Timestamp(time.Now().Add(-time.Hour)), nil)
	tick = NewTickEventWithID("user-1", old)
	require.Equal(t, old.String(), tick.ID)
	require.WithinDuration(t, time.Now(), time.UnixMilli(tick.Timestamp), 5*time.Second)
	wake = NewWakeEventWithID("user-1", "wake-1", old)
	require.Equal(t, old.String(), wake.ID)
	require.WithinDuration(t, time.Now(), time.UnixMilli(wake.Timestamp), 5*time.Second)

	// the timer id is deterministic and its time part is the nonce's.
	nonce := ulid.MustNew(ulid.Timestamp(time.Now().Add(-time.Hour)), nil)
	a := TimerEventID("tick:x", nonce.String())
	require.Equal(t, a, TimerEventID("tick:x", nonce.String()))
	require.NotEqual(t, a, TimerEventID("tick:x", ulid.Make().String()))
	require.NotEqual(t, a, TimerEventID("tick:y", nonce.String()))
	require.Equal(t, nonce.Time(), a.Time())
}

func TestInputFromEvent(t *testing.T) {
	ctx := context.Background()
	loop := inngest.Loop{
		Kind: "loop",
		Key:  "event.data.id",
	}
	tests := []struct {
		name    string
		evt     map[string]any
		want    Input
		wantErr error
		wantMsg string
	}{
		{
			name: "trigger uses the key expression",
			evt: map[string]any{
				"name": "app/user.created",
				"data": map[string]any{"id": "user-1"},
			},
			want: Input{
				Kind: InputEvent,
				Key:  "user-1",
			},
		},
		{
			name: "trigger without the key field",
			evt: map[string]any{
				"name": "app/user.created",
				"data": map[string]any{},
			},
			wantErr: ErrNoKey,
		},
		{
			name: "tick reads the key from data",
			evt:  NewTickEvent("user-1").Map(),
			want: Input{
				Kind: InputTick,
				Key:  "user-1",
			},
		},
		{
			name: "wake reads the key and the id from data",
			evt:  NewWakeEvent("user-1", "wake-1").Map(),
			want: Input{
				Kind: InputWake,
				Key:  "user-1",
				ID:   "wake-1",
			},
		},
		{
			name: "tick without a key",
			evt: map[string]any{
				"name": EventTick,
				"data": map[string]any{"id": "user-1"},
			},
			wantMsg: "has no key",
		},
		{
			name: "wake without an id",
			evt: map[string]any{
				"name": EventWake,
				"data": map[string]any{"key": "user-1"},
			},
			wantMsg: "has no wake id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InputFromEvent(ctx, loop, tt.evt)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.wantMsg != "" {
				require.ErrorContains(t, err, tt.wantMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
