package loader

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

type fakeAppFunctionReader struct {
	apps             map[uuid.UUID]*cqrs.App
	functions        map[uuid.UUID]*cqrs.Function
	appsErr          error
	functionsErr     error
	appRequests      [][]uuid.UUID
	functionRequests [][]uuid.UUID
}

func (f *fakeAppFunctionReader) GetAppsByIDs(_ context.Context, ids []uuid.UUID) ([]*cqrs.App, error) {
	f.appRequests = append(f.appRequests, append([]uuid.UUID(nil), ids...))
	if f.appsErr != nil {
		return nil, f.appsErr
	}
	out := make([]*cqrs.App, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		if app, ok := f.apps[id]; ok {
			out = append(out, app)
		}
	}
	return out, nil
}

func (f *fakeAppFunctionReader) GetFunctionsByInternalUUIDs(_ context.Context, ids []uuid.UUID) ([]*cqrs.Function, error) {
	f.functionRequests = append(f.functionRequests, append([]uuid.UUID(nil), ids...))
	if f.functionsErr != nil {
		return nil, f.functionsErr
	}
	out := make([]*cqrs.Function, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		if fn, ok := f.functions[id]; ok {
			out = append(out, fn)
		}
	}
	return out, nil
}

func TestAppLoaderPreservesOrderDuplicatesMissingAndInvalidKeys(t *testing.T) {
	ctx := context.Background()
	app1ID := uuid.New()
	app2ID := uuid.New()
	missingID := uuid.New()
	reader := &fakeAppFunctionReader{apps: map[uuid.UUID]*cqrs.App{
		app1ID: {ID: app1ID, Name: "first"},
		app2ID: {ID: app2ID, Name: "second"},
	}}

	results := (appFunctionReader{reader: reader}).GetApps(ctx, dataloader.NewKeysFromStrings([]string{
		app2ID.String(),
		"invalid",
		app1ID.String(),
		app2ID.String(),
		missingID.String(),
	}))

	require.Len(t, results, 5)
	require.Equal(t, []uuid.UUID{app2ID, app1ID, app2ID, missingID}, reader.appRequests[0])
	require.Equal(t, app2ID, results[0].Data.(*cqrs.App).ID)
	require.Error(t, results[1].Error)
	require.Equal(t, app1ID, results[2].Data.(*cqrs.App).ID)
	require.Equal(t, app2ID, results[3].Data.(*cqrs.App).ID)
	require.ErrorIs(t, results[4].Error, sql.ErrNoRows)
	for _, result := range results {
		require.NotNil(t, result)
	}
}

func TestFunctionLoaderPreservesOrderAndReportsMissingRows(t *testing.T) {
	ctx := context.Background()
	fn1ID := uuid.New()
	fn2ID := uuid.New()
	missingID := uuid.New()
	reader := &fakeAppFunctionReader{functions: map[uuid.UUID]*cqrs.Function{
		fn1ID: {ID: fn1ID, Name: "first"},
		fn2ID: {ID: fn2ID, Name: "second"},
	}}

	results := (appFunctionReader{reader: reader}).GetFunctions(ctx, dataloader.NewKeysFromStrings([]string{
		fn2ID.String(),
		missingID.String(),
		fn1ID.String(),
	}))

	require.Len(t, results, 3)
	require.Equal(t, []uuid.UUID{fn2ID, missingID, fn1ID}, reader.functionRequests[0])
	require.Equal(t, fn2ID, results[0].Data.(*cqrs.Function).ID)
	require.ErrorIs(t, results[1].Error, sql.ErrNoRows)
	require.Equal(t, fn1ID, results[2].Data.(*cqrs.Function).ID)
}

func TestAppAndFunctionLoadersPropagateDatabaseErrorsWithoutPoisoningInvalidKeys(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("database unavailable")
	appID := uuid.New()
	fnID := uuid.New()
	reader := &fakeAppFunctionReader{appsErr: wantErr, functionsErr: wantErr}

	appResults := (appFunctionReader{reader: reader}).GetApps(ctx, dataloader.NewKeysFromStrings([]string{appID.String(), "invalid"}))
	require.ErrorIs(t, appResults[0].Error, wantErr)
	require.Error(t, appResults[1].Error)
	require.NotErrorIs(t, appResults[1].Error, wantErr)

	functionResults := (appFunctionReader{reader: reader}).GetFunctions(ctx, dataloader.NewKeysFromStrings([]string{fnID.String(), "invalid"}))
	require.ErrorIs(t, functionResults[0].Error, wantErr)
	require.Error(t, functionResults[1].Error)
	require.NotErrorIs(t, functionResults[1].Error, wantErr)
}

func TestAppFunctionLoadersReturnResultsWithoutQueryingForEmptyOrInvalidBatches(t *testing.T) {
	ctx := context.Background()
	reader := &fakeAppFunctionReader{}
	appReader := appFunctionReader{reader: reader}

	for _, keys := range []dataloader.Keys{
		dataloader.NewKeysFromStrings(nil),
		dataloader.NewKeysFromStrings([]string{"invalid"}),
	} {
		results := appReader.GetApps(ctx, keys)
		for _, result := range results {
			require.NotNil(t, result)
		}
	}
	require.Empty(t, reader.appRequests)

	results := appReader.GetFunctions(ctx, dataloader.NewKeysFromStrings(nil))
	require.Empty(t, results)
	require.Empty(t, reader.functionRequests)
}

func TestAppLoaderCacheIsScopedToLoaderInstance(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	reader := &fakeAppFunctionReader{apps: map[uuid.UUID]*cqrs.App{
		appID: {ID: appID, Name: "cached"},
	}}

	firstRequestLoader := dataloader.NewBatchedLoader((appFunctionReader{reader: reader}).GetApps)
	first := firstRequestLoader.Load(ctx, dataloader.StringKey(appID.String()))
	second := firstRequestLoader.Load(ctx, dataloader.StringKey(appID.String()))
	firstResult, firstErr := first()
	secondResult, secondErr := second()
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	require.Equal(t, appID, firstResult.(*cqrs.App).ID)
	require.Equal(t, appID, secondResult.(*cqrs.App).ID)
	require.Len(t, reader.appRequests, 1)

	secondRequestLoader := dataloader.NewBatchedLoader((appFunctionReader{reader: reader}).GetApps)
	third := secondRequestLoader.Load(ctx, dataloader.StringKey(appID.String()))
	thirdResult, thirdErr := third()
	require.NoError(t, thirdErr)
	require.Equal(t, appID, thirdResult.(*cqrs.App).ID)
	require.Len(t, reader.appRequests, 2)
}

func TestAppAndFunctionLoadersBatchDistinctKeys(t *testing.T) {
	const count = 100
	ctx := context.Background()
	reader := &fakeAppFunctionReader{
		apps:      make(map[uuid.UUID]*cqrs.App, count),
		functions: make(map[uuid.UUID]*cqrs.Function, count),
	}
	ids := make([]uuid.UUID, count)
	for i := range ids {
		ids[i] = uuid.New()
		reader.apps[ids[i]] = &cqrs.App{ID: ids[i]}
		reader.functions[ids[i]] = &cqrs.Function{ID: ids[i]}
	}

	batchReader := appFunctionReader{reader: reader}
	// Flush on capacity so this assertion does not depend on a timed batch window.
	apps := dataloader.NewBatchedLoader(batchReader.GetApps, dataloader.WithBatchCapacity(count), dataloader.WithWait(time.Hour))
	functions := dataloader.NewBatchedLoader(batchReader.GetFunctions, dataloader.WithBatchCapacity(count), dataloader.WithWait(time.Hour))
	appResults := make([]dataloader.Thunk, count)
	functionResults := make([]dataloader.Thunk, count)
	for i, id := range ids {
		appResults[i] = apps.Load(ctx, dataloader.StringKey(id.String()))
		functionResults[i] = functions.Load(ctx, dataloader.StringKey(id.String()))
	}
	for i, id := range ids {
		app, err := appResults[i]()
		require.NoError(t, err)
		require.Equal(t, id, app.(*cqrs.App).ID)
		function, err := functionResults[i]()
		require.NoError(t, err)
		require.Equal(t, id, function.(*cqrs.Function).ID)
	}
	require.Len(t, reader.appRequests, 1)
	require.ElementsMatch(t, ids, reader.appRequests[0])
	require.Len(t, reader.functionRequests, 1)
	require.ElementsMatch(t, ids, reader.functionRequests[0])
}
