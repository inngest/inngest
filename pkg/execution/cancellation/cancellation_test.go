package cancellation

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type staticReader []cqrs.Cancellation

func (r staticReader) ReadAt(context.Context, uuid.UUID, uuid.UUID, time.Time) ([]cqrs.Cancellation, error) {
	return r, nil
}

func TestCheckerOnlyCancelsRunsQueuedInRange(t *testing.T) {
	at := time.UnixMilli(1_700_000_000_000)
	after := at.Add(-time.Hour)
	runID := ulid.MustNew(ulid.Timestamp(at), nil)
	early := cqrs.Cancellation{ID: ulid.Make(), StartedBefore: at.Add(-time.Minute)}
	late := cqrs.Cancellation{ID: ulid.Make(), StartedAfter: ptr(at.Add(time.Minute)), StartedBefore: at.Add(time.Hour)}
	match := cqrs.Cancellation{ID: ulid.Make(), StartedAfter: &after, StartedBefore: at}

	got, err := NewChecker(staticReader{early, late}).IsCancelled(t.Context(), uuid.New(), uuid.New(), runID, nil)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = NewChecker(staticReader{early, late, match}).IsCancelled(t.Context(), uuid.New(), uuid.New(), runID, nil)
	require.NoError(t, err)
	require.Equal(t, match.ID, got.ID)
}

func TestRedisReadAtExcludesCancellationsOutOfRange(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	wsID, fnID := uuid.New(), uuid.New()
	at := time.UnixMilli(1_700_000_000_000)
	store := func(c cqrs.Cancellation) {
		c.WorkspaceID, c.FunctionID = wsID, fnID
		byt, err := json.Marshal(redisWrapper{Version: 1, Cancellation: c})
		require.NoError(t, err)
		mr.HSet(redisReadWriter{prefix: DefaultPrefix}.key(wsID, fnID), c.ID.String(), string(byt))
	}
	inRange := cqrs.Cancellation{ID: ulid.Make(), StartedAfter: ptr(at.Add(-time.Hour)), StartedBefore: at.Add(time.Hour)}
	store(inRange)
	store(cqrs.Cancellation{ID: ulid.Make(), StartedBefore: at.Add(-time.Minute)})
	store(cqrs.Cancellation{ID: ulid.Make(), StartedAfter: ptr(at.Add(time.Minute)), StartedBefore: at.Add(time.Hour)})

	got, err := NewRedisReader(rc, "").ReadAt(t.Context(), wsID, fnID, at)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, inRange.ID, got[0].ID)
}

func ptr[T any](v T) *T { return &v }
