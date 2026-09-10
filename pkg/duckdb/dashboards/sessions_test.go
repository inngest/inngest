package dashboards_test

import (
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/dashboards"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

// newTestDuckDB duplicates pkg/cqrs/duckdbquery/testutil_test.go's
// unexported helper of the same name/shape (itself duplicated across
// several _test.go files that can't import a package-private helper from
// another package -- see pkg/duckdb/insights/views_test.go's own copy).
func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	db, err := duckdb.Open(t.Context(), duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	if err := duckdb.Migrate(t.Context(), db, true); err != nil {
		t.Fatalf("migrating duckdb: %v", err)
	}
	return db, func() { _ = db.Close() }
}

type testRun struct {
	runID        string
	accountID    uuid.UUID
	envID        uuid.UUID
	appID        uuid.UUID
	appName      string
	functionID   uuid.UUID
	functionSlug string
	status       string
	queuedAt     time.Time
	endedAt      *time.Time
	eventName    string
	sessions     []testSession
}

type testSession struct {
	key string
	id  string
}

// insertRun inserts one lifecycle row of inngest.runs -- a caller wanting
// multiple lifecycle rows for the same run_id (to exercise the
// latest-row-per-run_id collapse) calls this more than once with the same
// runID and a later queuedAt/status.
func insertRun(t *testing.T, db *sql.DB, r testRun) {
	t.Helper()

	sessionsLiteral := "NULL"
	if len(r.sessions) > 0 {
		parts := make([]string, len(r.sessions))
		for i, s := range r.sessions {
			parts[i] = fmt.Sprintf("{'key': %s, 'id': %s}", quoteLiteral(s.key), quoteLiteral(s.id))
		}
		sessionsLiteral = "[" + joinComma(parts) + "]"
	}

	inputs := fmt.Sprintf(`[{"name": %s, "data": {}}]`, jsonString(r.eventName))
	if r.eventName == "" {
		inputs = `[]`
	}

	query := fmt.Sprintf(`
INSERT INTO inngest.runs
  (account_id, env_id, run_id, queued_at, ended_at, app_id, app_name, function_id, function_slug, status, attributes, inputs, sessions)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, %s);`, sessionsLiteral)
	_, err := db.ExecContext(t.Context(), query,
		r.accountID.String(), r.envID.String(), r.runID, r.queuedAt, r.endedAt,
		r.appID.String(), r.appName, r.functionID.String(), r.functionSlug, r.status, inputs,
	)
	require.NoError(t, err)
}

func quoteLiteral(s string) string {
	return "'" + s + "'"
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func jsonString(s string) string {
	return fmt.Sprintf("%q", s)
}

func TestListSessionKeys(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now.Add(-2 * time.Hour), eventName: "e1",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now.Add(-time.Hour), eventName: "e2",
		sessions: []testSession{{key: "org", id: "o1"}},
	})
	// A different env's runs must never surface here.
	insertRun(t, db, testRun{
		runID: "r3", accountID: accountID, envID: uuid.New(), appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e3",
		sessions: []testSession{{key: "other-env", id: "x"}},
	})
	// A run with no sessions must never surface a key.
	insertRun(t, db, testRun{
		runID: "r4", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e4",
	})

	result, err := dashboards.ListSessionKeys(t.Context(), db, accountID, envID, dashboards.SessionKeysOpts{})
	require.NoError(t, err)
	require.False(t, result.HasMore)
	require.Len(t, result.Keys, 2)

	keys := map[string]bool{}
	for _, k := range result.Keys {
		keys[k.Key] = true
		require.False(t, k.CreatedAt.IsZero())
	}
	require.True(t, keys["user"])
	require.True(t, keys["org"])
	require.False(t, keys["other-env"])
}

func TestListSessionKeys_Search(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e1",
		sessions: []testSession{{key: "user-session", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e2",
		sessions: []testSession{{key: "org-session", id: "o1"}},
	})

	result, err := dashboards.ListSessionKeys(t.Context(), db, accountID, envID, dashboards.SessionKeysOpts{Search: "user"})
	require.NoError(t, err)
	require.Len(t, result.Keys, 1)
	require.Equal(t, "user-session", result.Keys[0].Key)
}

func TestListSessionKeys_Pagination(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	for i := range 3 {
		insertRun(t, db, testRun{
			runID: fmt.Sprintf("r%d", i), accountID: accountID, envID: envID, appID: appID, appName: "app",
			functionID: functionID, functionSlug: "fn", status: "Completed",
			queuedAt: now.Add(time.Duration(i) * time.Minute), eventName: "e",
			sessions: []testSession{{key: fmt.Sprintf("key-%d", i), id: "id"}},
		})
	}

	first, err := dashboards.ListSessionKeys(t.Context(), db, accountID, envID, dashboards.SessionKeysOpts{Limit: 2})
	require.NoError(t, err)
	require.True(t, first.HasMore)
	require.Len(t, first.Keys, 2)
	require.Equal(t, "key-2", first.Keys[0].Key)
	require.Equal(t, "key-1", first.Keys[1].Key)

	second, err := dashboards.ListSessionKeys(t.Context(), db, accountID, envID, dashboards.SessionKeysOpts{
		Limit: 2, Cursor: first.Keys[len(first.Keys)-1].Cursor,
	})
	require.NoError(t, err)
	require.False(t, second.HasMore)
	require.Len(t, second.Keys, 1)
	require.Equal(t, "key-0", second.Keys[0].Key)
}

func TestListSessions_GroupsByIDAndCollapsesLifecycleRows(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	ended := now.Add(time.Second)

	// Two lifecycle rows for the same run (queued, then completed) --
	// must collapse to exactly one run in run_count, not two.
	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn-a",
		status: "Running", queuedAt: now, eventName: "e1",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn-a",
		status: "Completed", queuedAt: now, endedAt: &ended, eventName: "e1",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn-b",
		status: "Failed", queuedAt: now.Add(time.Minute), eventName: "e2",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r3", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn-a",
		status: "Completed", queuedAt: now, eventName: "e3",
		sessions: []testSession{{key: "user", id: "u2"}},
	})

	result, err := dashboards.ListSessions(t.Context(), db, accountID, envID, dashboards.SessionsOpts{
		Key: "user", From: now.Add(-time.Hour), Until: now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.False(t, result.HasMore)
	require.Len(t, result.Sessions, 2)

	byID := map[string]dashboards.SessionGroup{}
	for _, g := range result.Sessions {
		byID[g.ID] = g
	}

	u1 := byID["u1"]
	require.Equal(t, 2, u1.RunCount, "the two r1 lifecycle rows must collapse to one run")
	require.Equal(t, 1, u1.FailedRunCount)
	require.InDelta(t, 0.5, u1.FailureRate(), 0.0001)
	require.Len(t, u1.Functions, 2)

	u2 := byID["u2"]
	require.Equal(t, 1, u2.RunCount)
	require.Equal(t, 0, u2.FailedRunCount)
	require.Len(t, u2.Functions, 1)
	require.Equal(t, "fn-a", u2.Functions[0].Slug)
	require.Equal(t, appID, u2.Functions[0].AppID)
	require.Equal(t, "app", u2.Functions[0].AppName)
}

func TestListSessions_IDSearchAndTimeRange(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e1",
		sessions: []testSession{{key: "user", id: "alice"}},
	})
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e2",
		sessions: []testSession{{key: "user", id: "bob"}},
	})
	// Outside the requested time range.
	insertRun(t, db, testRun{
		runID: "r3", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now.Add(-48 * time.Hour), eventName: "e3",
		sessions: []testSession{{key: "user", id: "alice"}},
	})

	result, err := dashboards.ListSessions(t.Context(), db, accountID, envID, dashboards.SessionsOpts{
		Key: "user", IDSearch: "ali", From: now.Add(-time.Hour), Until: now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.Len(t, result.Sessions, 1)
	require.Equal(t, "alice", result.Sessions[0].ID)
	require.Equal(t, 1, result.Sessions[0].RunCount, "the out-of-range alice run must not be counted")
}

func TestListSessions_DefaultsTimeRangeWhenUnset(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e1",
		sessions: []testSession{{key: "user", id: "alice"}},
	})
	// Outside the default 7-day window.
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now.Add(-10 * 24 * time.Hour), eventName: "e2",
		sessions: []testSession{{key: "user", id: "bob"}},
	})

	result, err := dashboards.ListSessions(t.Context(), db, accountID, envID, dashboards.SessionsOpts{Key: "user"})
	require.NoError(t, err)
	require.Len(t, result.Sessions, 1)
	require.Equal(t, "alice", result.Sessions[0].ID)
}

func TestListSessionRuns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	ended := now.Add(time.Second)

	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, endedAt: &ended, eventName: "orders/created",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Running", queuedAt: now.Add(time.Minute), eventName: "orders/updated",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	// Different session id -- must not appear.
	insertRun(t, db, testRun{
		runID: "r3", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "orders/other",
		sessions: []testSession{{key: "user", id: "u2"}},
	})

	result, err := dashboards.ListSessionRuns(t.Context(), db, accountID, envID, dashboards.SessionRunsOpts{
		Key: "user", ID: "u1", From: now.Add(-time.Hour), Until: now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.False(t, result.HasMore)
	require.Len(t, result.Runs, 2)

	// Ordered most-recently-queued first.
	require.Equal(t, "r2", result.Runs[0].RunID)
	require.Equal(t, "r1", result.Runs[1].RunID)

	completed := result.Runs[1]
	require.Equal(t, "fn", completed.Function.Slug)
	require.Equal(t, appID, completed.Function.AppID)
	require.Equal(t, "app", completed.Function.AppName)
	require.Equal(t, "orders/created", completed.EventName)
	require.Equal(t, enums.RunStatusCompleted, completed.Status)
	require.False(t, completed.EndedAt.IsZero())

	running := result.Runs[0]
	require.Equal(t, enums.RunStatusRunning, running.Status)
	require.True(t, running.EndedAt.IsZero())
}

func TestListSessionRuns_Pagination(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	for i := range 3 {
		insertRun(t, db, testRun{
			runID: fmt.Sprintf("r%d", i), accountID: accountID, envID: envID, appID: appID, appName: "app",
			functionID: functionID, functionSlug: "fn", status: "Completed",
			queuedAt: now.Add(time.Duration(i) * time.Minute), eventName: "e",
			sessions: []testSession{{key: "user", id: "u1"}},
		})
	}

	first, err := dashboards.ListSessionRuns(t.Context(), db, accountID, envID, dashboards.SessionRunsOpts{
		Key: "user", ID: "u1", From: now.Add(-time.Hour), Until: now.Add(time.Hour), Limit: 2,
	})
	require.NoError(t, err)
	require.True(t, first.HasMore)
	require.Len(t, first.Runs, 2)
	require.Equal(t, "r2", first.Runs[0].RunID)
	require.Equal(t, "r1", first.Runs[1].RunID)

	second, err := dashboards.ListSessionRuns(t.Context(), db, accountID, envID, dashboards.SessionRunsOpts{
		Key: "user", ID: "u1", From: now.Add(-time.Hour), Until: now.Add(time.Hour), Limit: 2,
		Cursor: first.Runs[len(first.Runs)-1].Cursor,
	})
	require.NoError(t, err)
	require.False(t, second.HasMore)
	require.Len(t, second.Runs, 1)
	require.Equal(t, "r0", second.Runs[0].RunID)
}

func TestListSessions_EnvIsolation(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID, otherEnv := uuid.New(), uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	insertRun(t, db, testRun{
		runID: "r1", accountID: accountID, envID: envID, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e1",
		sessions: []testSession{{key: "user", id: "u1"}},
	})
	insertRun(t, db, testRun{
		runID: "r2", accountID: accountID, envID: otherEnv, appID: appID, appName: "app", functionID: functionID, functionSlug: "fn",
		status: "Completed", queuedAt: now, eventName: "e2",
		sessions: []testSession{{key: "user", id: "u1"}},
	})

	result, err := dashboards.ListSessions(t.Context(), db, accountID, envID, dashboards.SessionsOpts{
		Key: "user", From: now.Add(-time.Hour), Until: now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.Len(t, result.Sessions, 1)
	require.Equal(t, 1, result.Sessions[0].RunCount, "the other env's run must not be counted")
}
