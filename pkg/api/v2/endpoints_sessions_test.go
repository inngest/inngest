package apiv2

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// insertSessionRun inserts one inngest.runs row scoped to
// consts.DevServerAccountID/EnvID, the account/env every Sessions endpoint
// hardcodes (see endpoints_sessions.go's doc comment) -- mirrors
// pkg/duckdb/dashboards/sessions_test.go's own insertRun helper, redefined
// here because it's a different package and scoped to the dev-server
// identity rather than a caller-supplied one.
func insertSessionRun(t *testing.T, db *sql.DB, appID, functionID uuid.UUID, runID, sessionKey, sessionID, status string, queuedAt time.Time) {
	t.Helper()
	sessions := fmt.Sprintf("[{'key': '%s', 'id': '%s'}]", sessionKey, sessionID)
	_, err := db.ExecContext(t.Context(), fmt.Sprintf(`
INSERT INTO inngest.runs
  (account_id, env_id, run_id, queued_at, scheduled_at, app_id, app_name, function_id, function_slug, status, attributes, inputs, sessions)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', '[{"name": "e1", "data": {}}]', %s);`, sessions),
		consts.DevServerAccountID.String(), consts.DevServerEnvID.String(), runID, queuedAt, queuedAt,
		appID.String(), "app", functionID.String(), "app-fn", status,
	)
	require.NoError(t, err)
}

func TestListSessionKeys_RequiresDuckDB(t *testing.T) {
	service := NewService(ServiceOptions{})

	resp, err := service.ListSessionKeys(context.Background(), &apiv2.ListSessionKeysRequest{})

	require.Nil(t, resp)
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.ErrorContains(t, err, "Sessions requires dual-write")
}

func TestListSessionKeys_ReturnsObservedKeys(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	insertSessionRun(t, db, appID, functionID, "r1", "user", "u1", "Completed", now)
	insertSessionRun(t, db, appID, functionID, "r2", "org", "o1", "Completed", now)

	resp, err := service.ListSessionKeys(context.Background(), &apiv2.ListSessionKeysRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Data, 2)
	require.NotNil(t, resp.Metadata.FetchedAt)
	require.False(t, resp.Page.HasMore)

	keys := map[string]bool{}
	for _, k := range resp.Data {
		keys[k.Id] = true
		require.NotNil(t, k.CreatedAt)
	}
	require.True(t, keys["user"])
	require.True(t, keys["org"])
}

func TestListSessions_RequiresSessionKey(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	resp, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{})
	require.Nil(t, resp)
	require.ErrorContains(t, err, "Session key is required")
}

func TestListSessions_ReturnsGroupsForKey(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	insertSessionRun(t, db, appID, functionID, "r1", "user", "u1", "Completed", now)
	insertSessionRun(t, db, appID, functionID, "r2", "user", "u1", "Failed", now)
	insertSessionRun(t, db, appID, functionID, "r3", "user", "u2", "Completed", now)

	resp, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{SessionKey: "user"})
	require.NoError(t, err)
	require.Len(t, resp.Data, 2)

	byID := map[string]*apiv2.SessionGroup{}
	for _, g := range resp.Data {
		byID[g.Id] = g
	}
	require.Equal(t, int32(2), byID["u1"].RunCount)
	require.Equal(t, int32(1), byID["u1"].FailedRunCount)
	require.Len(t, byID["u1"].Functions, 1)
	require.Equal(t, "fn", byID["u1"].Functions[0].Id)
	require.Equal(t, "app-fn", byID["u1"].Functions[0].GetSlug())
	require.Equal(t, "app", byID["u1"].Functions[0].App.Id)
	require.Equal(t, int32(1), byID["u2"].RunCount)
}

func TestListSessionRuns_RequiresSessionKeyAndID(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	resp, err := service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{SessionKey: "user"})
	require.Nil(t, resp)
	require.ErrorContains(t, err, "Session key and session ID are required")
}

func TestListSessionRuns_ReturnsMatchingRuns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	insertSessionRun(t, db, appID, functionID, "r1", "user", "u1", "Completed", now)
	insertSessionRun(t, db, appID, functionID, "r2", "user", "u2", "Completed", now)

	resp, err := service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{SessionKey: "user", SessionId: "u1"})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "r1", resp.Data[0].Id)
	require.Equal(t, "fn", resp.Data[0].Function.Id)
	require.Equal(t, "app-fn", resp.Data[0].Function.GetSlug())
	require.Equal(t, "app", resp.Data[0].Function.App.Id)
	require.Equal(t, apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED, resp.Data[0].Status)
}

// TestSessions_CallerErrorsAre400s covers caller mistakes that used to
// surface as 500s, and from w/o until, which now defaults until to now
// (like GQL) rather than erroring.
func TestSessions_CallerErrorsAre400s(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	insertSessionRun(t, db, appID, functionID, "r1", "user", "u1", "Completed", now.Add(-time.Minute))

	t.Run("from w/o until defaults until to now", func(t *testing.T) {
		resp, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{
			SessionKey: "user", From: timestamppb.New(now.Add(-time.Hour)),
		})
		require.NoError(t, err)
		require.Len(t, resp.Data, 1)

		runs, err := service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{
			SessionKey: "user", SessionId: "u1", From: timestamppb.New(now.Add(-time.Hour)),
		})
		require.NoError(t, err)
		require.Len(t, runs.Data, 1)
	})

	t.Run("from after until", func(t *testing.T) {
		_, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{
			SessionKey: "user", From: timestamppb.New(now), Until: timestamppb.New(now.Add(-time.Hour)),
		})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.ErrorContains(t, err, "Invalid time range")

		_, err = service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{
			SessionKey: "user", SessionId: "u1", From: timestamppb.New(now), Until: timestamppb.New(now),
		})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("malformed cursor", func(t *testing.T) {
		bad := "not-a-cursor!"
		_, err := service.ListSessionKeys(context.Background(), &apiv2.ListSessionKeysRequest{Cursor: &bad})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.ErrorContains(t, err, "Invalid cursor")

		_, err = service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{SessionKey: "user", Cursor: &bad})
		require.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{SessionKey: "user", SessionId: "u1", Cursor: &bad})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

// TestSessions_CursorFromAnotherListingIs400 covers a cursor that decodes
// fine but belongs to a different listing (here a sessions cursor sent to
// session runs), which used to silently restart at page 1.
func TestSessions_CursorFromAnotherListingIs400(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	insertSessionRun(t, db, appID, functionID, "r1", "user", "u1", "Completed", now.Add(-2*time.Minute))
	insertSessionRun(t, db, appID, functionID, "r2", "user", "u2", "Completed", now.Add(-time.Minute))

	limit := int32(1)
	sessions, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{SessionKey: "user", Limit: &limit})
	require.NoError(t, err)
	require.True(t, sessions.Page.HasMore)
	require.NotNil(t, sessions.Page.Cursor)

	_, err = service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{
		SessionKey: "user", SessionId: "u1", Cursor: sessions.Page.Cursor,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "Invalid cursor")

	_, err = service.ListSessionKeys(context.Background(), &apiv2.ListSessionKeysRequest{Cursor: sessions.Page.Cursor})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
