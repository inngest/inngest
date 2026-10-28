package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

// fakeFunctionsManager is a minimal cqrs.Manager stub -- only GetFunctions
// (the one method sessionFunctionNames calls) is implemented, matching
// runs_v2_test.go's fakeRunTriggerManager pattern.
type fakeFunctionsManager struct {
	cqrs.Manager
	fns []*cqrs.Function
}

func (f *fakeFunctionsManager) GetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	return f.fns, nil
}

// insertGQLSessionRun inserts one inngest.runs row scoped to
// consts.DevServerAccountID/EnvID, the account/env every Sessions resolver
// hardcodes (see sessions.go's doc comment) -- mirrors
// pkg/api/v2/endpoints_sessions_test.go's own insertSessionRun helper,
// redefined here because it's a different package.
func insertGQLSessionRun(t *testing.T, db *sql.DB, appID, functionID uuid.UUID, runID, sessionKey, sessionID, status string, queuedAt time.Time) {
	t.Helper()
	sessions := fmt.Sprintf("[{'key': '%s', 'id': '%s'}]", sessionKey, sessionID)
	_, err := db.ExecContext(t.Context(), fmt.Sprintf(`
INSERT INTO inngest.runs
  (account_id, env_id, run_id, queued_at, scheduled_at, app_id, app_name, function_id, function_slug, status, attributes, inputs, sessions)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', '[{"name": "e1", "data": {}}]', %s);`, sessions),
		consts.DevServerAccountID.String(), consts.DevServerEnvID.String(), runID, queuedAt, queuedAt,
		appID.String(), "app", functionID.String(), "fn", status,
	)
	require.NoError(t, err)
}

func TestSessionKeys_RequiresDuckDB(t *testing.T) {
	r := &Resolver{DuckDB: nil}
	qr := r.Query().(*queryResolver)

	_, err := qr.SessionKeys(t.Context(), nil, 20, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires dual-write")
}

func TestSessionsEndToEnd(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	appID, functionID := uuid.New(), uuid.New()
	r := &Resolver{DuckDB: db, Data: &fakeFunctionsManager{fns: []*cqrs.Function{
		{ID: functionID, Slug: "fn", Name: "Test Function"},
	}}}
	qr := r.Query().(*queryResolver)

	now := time.Now().UTC()
	insertGQLSessionRun(t, db, appID, functionID, "r1", "user", "u1", "Completed", now)
	insertGQLSessionRun(t, db, appID, functionID, "r2", "user", "u1", "Failed", now)
	insertGQLSessionRun(t, db, appID, functionID, "r3", "org", "o1", "Completed", now)

	t.Run("SessionKeys", func(t *testing.T) {
		keys, err := qr.SessionKeys(t.Context(), nil, 20, nil)
		require.NoError(t, err)
		require.False(t, keys.PageInfo.HasNextPage)
		names := map[string]bool{}
		for _, k := range keys.Nodes {
			names[k.SessionKey] = true
		}
		require.True(t, names["user"])
		require.True(t, names["org"])
	})

	t.Run("Sessions", func(t *testing.T) {
		conn, err := qr.Sessions(t.Context(), "user", nil, nil, 20, nil)
		require.NoError(t, err)
		groups := conn.Nodes
		require.Len(t, groups, 1)
		require.Equal(t, "u1", groups[0].SessionID)
		require.Equal(t, 2, groups[0].RunCount)
		require.Equal(t, 1, groups[0].FailedRunCount)
		require.Len(t, groups[0].Functions, 1)
		require.Equal(t, "fn", groups[0].Functions[0].Slug)
		require.Equal(t, "Test Function", groups[0].Functions[0].Name)
	})

	t.Run("SessionRuns", func(t *testing.T) {
		runs, err := qr.SessionRuns(t.Context(), "user", "u1", nil, 20, nil)
		require.NoError(t, err)
		require.Len(t, runs.Nodes, 2)
	})

	t.Run("Pagination", func(t *testing.T) {
		first, err := qr.SessionRuns(t.Context(), "user", "u1", nil, 1, nil)
		require.NoError(t, err)
		require.Len(t, first.Nodes, 1)
		require.True(t, first.PageInfo.HasNextPage)
		require.NotNil(t, first.PageInfo.EndCursor)

		second, err := qr.SessionRuns(t.Context(), "user", "u1", nil, 1, first.PageInfo.EndCursor)
		require.NoError(t, err)
		require.Len(t, second.Nodes, 1)
		require.False(t, second.PageInfo.HasNextPage)
		require.Nil(t, second.PageInfo.EndCursor)
		require.NotEqual(t, first.Nodes[0].ID, second.Nodes[0].ID)

		// totalCount is the whole session's, not the page's.
		cr := r.SessionRunsConnection()
		for _, page := range []*models.SessionRunsConnection{first, second} {
			total, err := cr.TotalCount(t.Context(), page)
			require.NoError(t, err)
			require.Equal(t, 2, total)
		}

		keys, err := qr.SessionKeys(t.Context(), nil, 1, nil)
		require.NoError(t, err)
		require.Len(t, keys.Nodes, 1)
		require.True(t, keys.PageInfo.HasNextPage)
		keys, err = qr.SessionKeys(t.Context(), nil, 1, keys.PageInfo.EndCursor)
		require.NoError(t, err)
		require.Len(t, keys.Nodes, 1)
		require.False(t, keys.PageInfo.HasNextPage)
	})

	t.Run("FirstOutOfRange", func(t *testing.T) {
		_, err := qr.SessionKeys(t.Context(), nil, 101, nil)
		require.ErrorContains(t, err, "cannot exceed 100")
		_, err = qr.Sessions(t.Context(), "user", nil, nil, 0, nil)
		require.ErrorContains(t, err, "at least 1")
	})
}
