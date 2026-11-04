package db_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	"github.com/stretchr/testify/require"
)

func TestGetAppsByIDs(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	activeID := uuid.New()
	archivedID := uuid.New()
	for _, app := range []db.UpsertAppParams{
		{ID: activeID, Name: "batch-app-1", Checksum: "batch-app-1", Url: "https://example.com/batch-app-1"},
		{ID: archivedID, Name: "batch-app-2", Checksum: "batch-app-2", Url: "https://example.com/batch-app-2"},
	} {
		_, err := q.UpsertApp(ctx, app)
		require.NoError(t, err)
	}
	require.NoError(t, q.DeleteApp(ctx, archivedID))

	got, err := q.GetAppsByIDs(ctx, []uuid.UUID{archivedID, uuid.New(), activeID, activeID})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []uuid.UUID{activeID, archivedID}, []uuid.UUID{got[0].ID, got[1].ID})
	empty, err := q.GetAppsByIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestGetFunctionsByIDs(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	appID := uuid.New()
	_, err := q.UpsertApp(ctx, db.UpsertAppParams{ID: appID, Name: "batch-functions-app", Checksum: "batch-functions-app"})
	require.NoError(t, err)

	activeID := uuid.New()
	archivedID := uuid.New()
	for _, fn := range []db.UpsertFunctionParams{
		{ID: activeID, AppID: appID, Name: "Batch Function 1", Slug: "batch-function-1", Config: `{"version":1}`},
		{ID: archivedID, AppID: appID, Name: "Batch Function 2", Slug: "batch-function-2", Config: `{"version":2}`},
	} {
		_, err := q.UpsertFunction(ctx, fn)
		require.NoError(t, err)
	}
	require.NoError(t, q.DeleteFunctionsByIDs(ctx, []uuid.UUID{archivedID}))

	got, err := q.GetFunctionsByIDs(ctx, []uuid.UUID{archivedID, uuid.New(), activeID, activeID})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []uuid.UUID{activeID, archivedID}, []uuid.UUID{got[0].ID, got[1].ID})
	empty, err := q.GetFunctionsByIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}
