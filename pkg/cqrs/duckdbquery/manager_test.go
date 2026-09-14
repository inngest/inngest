package duckdbquery

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

// fakeManager is a minimal cqrs.Manager stub — enough to prove Wrap's
// embedding falls through to the underlying manager for methods this
// package doesn't override, without needing a real SQLite-backed manager.
// apps/functions/appErr/functionsErr back resolveAppAndFunctionFilters'
// tests (runs_test.go): GetAppByName/GetFunctions are the only two methods
// that resolver calls, so a real manager is never needed for those either.
type fakeManager struct {
	cqrs.Manager
	getAppsCalled bool

	apps         map[string]*cqrs.App
	appErr       error
	functions    []*cqrs.Function
	functionsErr error
}

func (f *fakeManager) GetApps(ctx context.Context, envID uuid.UUID, filter *cqrs.FilterAppParam) ([]*cqrs.App, error) {
	f.getAppsCalled = true
	return nil, nil
}

func (f *fakeManager) GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*cqrs.App, error) {
	if f.appErr != nil {
		return nil, f.appErr
	}
	app, ok := f.apps[name]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return app, nil
}

func (f *fakeManager) GetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	if f.functionsErr != nil {
		return nil, f.functionsErr
	}
	return f.functions, nil
}

func TestWrapFallsThroughForUnoverriddenMethods(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	fake := &fakeManager{}
	wrapped := Wrap(fake, db)

	_, err := wrapped.GetApps(context.Background(), uuid.New(), nil)
	require.NoError(t, err)
	require.True(t, fake.getAppsCalled, "GetApps must fall through to the embedded manager")
}

func TestManagerImplementsFlatSpanSource(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	m := Wrap(&fakeManager{}, db)
	type flatSpanSource interface{ FlatSpans() bool }
	fs, ok := m.(flatSpanSource)
	require.True(t, ok, "duckdbquery.Manager must implement FlatSpans() bool")
	require.True(t, fs.FlatSpans())
}
