package duckdbquery

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestGetEventsByInternalIDs(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID := uuid.New(), uuid.New()
	id1 := ulid.MustNew(ulid.Now(), rand.Reader)
	id2 := ulid.MustNew(ulid.Now(), rand.Reader)

	for _, e := range []struct {
		id   ulid.ULID
		name string
	}{{id1, "test/one"}, {id2, "test/two"}} {
		_, err := db.ExecContext(ctx,
			`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
			 VALUES (?, ?, ?, ?, 'test', ?, ?, '{"k":"v"}', '1', ?);`,
			accountID.String(), envID.String(), e.id.String(), time.Now().UTC(), e.id.String(), e.name, time.Now().UTC(),
		)
		require.NoError(t, err)
	}

	m := Wrap(nil, db)
	events, err := m.GetEventsByInternalIDs(ctx, []ulid.ULID{id1, id2})
	require.NoError(t, err)
	require.Len(t, events, 2)

	byID := map[ulid.ULID]*cqrs.Event{}
	for _, e := range events {
		byID[e.GetInternalID()] = e
	}
	require.Equal(t, "test/one", byID[id1].EventName)
	require.Equal(t, accountID, byID[id1].AccountID)
	require.Equal(t, envID, byID[id1].WorkspaceID)
	require.Equal(t, map[string]any{"k": "v"}, byID[id1].EventData)
	require.Nil(t, byID[id1].SourceID)
}

func TestGetEventsByInternalIDsEmptyInput(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	m := Wrap(nil, db)
	events, err := m.GetEventsByInternalIDs(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestGetEventsFiltersByNameAndExcludesInternalByDefault(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	seed := func(name string, ts time.Time) {
		_, err := db.ExecContext(ctx,
			`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
			 VALUES (?, ?, ?, ?, 'test', ?, ?, '{}', '1', ?);`,
			accountID.String(), envID.String(), ulid.MustNew(ulid.Timestamp(ts), rand.Reader).String(), ts, ulid.MustNew(ulid.Timestamp(ts), rand.Reader).String(), name, ts,
		)
		require.NoError(t, err)
	}
	// Timestamps must be in the past relative to real wall-clock time —
	// GetEvents defaults opts.Newest to time.Now() (see WorkspaceEventsOpts.Validate),
	// so a future-dated seed row would be excluded by the upper time bound.
	seed("app/one", now.Add(-2*time.Second))
	seed("app/two", now.Add(-time.Second))
	seed("inngest/function.finished", now)

	m := Wrap(nil, db)
	events, err := m.GetEvents(ctx, accountID, envID, &cqrs.WorkspaceEventsOpts{Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 2, "internal inngest/* events are excluded by default")

	events, err = m.GetEvents(ctx, accountID, envID, &cqrs.WorkspaceEventsOpts{Limit: 10, Names: []string{"app/one"}})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "app/one", events[0].EventName)
}
