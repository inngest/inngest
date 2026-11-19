package manager

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

func TestCQRSGetAppsByIDs(t *testing.T) {
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()

	activeID := uuid.New()
	archivedID := uuid.New()
	for _, app := range []cqrs.UpsertAppParams{
		{ID: activeID, Name: "Batch App 1", Checksum: "batch-app-1", Url: "https://example.com/batch-app-1"},
		{ID: archivedID, Name: "Batch App 2", Checksum: "batch-app-2", Url: "https://example.com/batch-app-2"},
	} {
		_, err := cm.UpsertApp(ctx, app)
		require.NoError(t, err)
	}
	require.NoError(t, cm.DeleteApp(ctx, archivedID))

	got, err := cm.GetAppsByIDs(ctx, []uuid.UUID{archivedID, uuid.New(), activeID})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []uuid.UUID{activeID, archivedID}, []uuid.UUID{got[0].ID, got[1].ID})
}

func TestCQRSGetFunctionsByInternalUUIDs(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	activeID := uuid.New()
	archivedID := uuid.New()
	for _, fn := range []cqrs.UpsertFunctionParams{
		{ID: activeID, AppID: appID, Name: "Batch Function 1", Slug: "batch-function-1", Config: `{"version":1}`, CreatedAt: time.Now()},
		{ID: archivedID, AppID: appID, Name: "Batch Function 2", Slug: "batch-function-2", Config: `{"version":2}`, CreatedAt: time.Now()},
	} {
		_, err := cm.UpsertFunction(ctx, fn)
		require.NoError(t, err)
	}
	require.NoError(t, cm.DeleteFunctionsByIDs(ctx, []uuid.UUID{archivedID}))

	got, err := cm.GetFunctionsByInternalUUIDs(ctx, []uuid.UUID{archivedID, uuid.New(), activeID})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []uuid.UUID{activeID, archivedID}, []uuid.UUID{got[0].ID, got[1].ID})
}
