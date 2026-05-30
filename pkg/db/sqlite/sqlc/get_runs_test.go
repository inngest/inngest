package sqlc_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/db/sqlite/sqlc"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestGetRuns(t *testing.T) {
	ctx := context.Background()
	conn, err := sqlite.Open(ctx, sqlite.Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	q := sqlc.New(conn)
	appID := uuid.New()
	_, err = q.UpsertApp(ctx, sqlc.UpsertAppParams{
		ID: appID, Name: "event-runs-app", SdkLanguage: "go", SdkVersion: "1.0.0",
		Metadata: "{}", Status: "active", Checksum: "checksum", Url: "http://example.com", Method: "POST",
	})
	require.NoError(t, err)

	fnID := uuid.New()
	_, err = q.UpsertFunction(ctx, sqlc.UpsertFunctionParams{
		ID:        fnID,
		AppID:     appID,
		Name:      "Event Runs Function",
		Slug:      "event-runs-app-event-runs-function",
		Config:    `{"name":"Event Runs Function","slug":"event-runs-function"}`,
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	eventID := ulid.Make()
	runID := ulid.Make()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, q.InsertFunctionRun(ctx, sqlc.InsertFunctionRunParams{
		RunID:           runID,
		RunStartedAt:    startedAt,
		FunctionID:      fnID,
		FunctionVersion: 1,
		TriggerType:     "event",
		EventID:         eventID,
		WorkspaceID:     uuid.New(),
	}))
	require.NoError(t, q.InsertFunctionFinish(ctx, sqlc.InsertFunctionFinishParams{
		RunID:              runID,
		Status:             sql.NullString{String: "completed", Valid: true},
		Output:             sql.NullString{String: "", Valid: true},
		CompletedStepCount: sql.NullInt64{Int64: 1, Valid: true},
		CreatedAt:          sql.NullTime{Time: startedAt.Add(time.Second), Valid: true},
	}))
	require.NoError(t, q.InsertTraceRun(ctx, sqlc.InsertTraceRunParams{
		RunID:       runID,
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       appID,
		FunctionID:  fnID,
		TraceID:     []byte("trace-id"),
		QueuedAt:    startedAt.UnixMilli(),
		StartedAt:   startedAt.UnixMilli(),
		EndedAt:     startedAt.Add(time.Second).UnixMilli(),
		Status:      300,
		SourceID:    "test",
		TriggerIds:  []byte(eventID.String()),
		Output:      []byte(`{"data":{"ok":true}}`),
	}))
	batchEventID := ulid.Make()
	require.NoError(t, q.InsertEventBatch(ctx, sqlc.InsertEventBatchParams{
		ID:          ulid.Make(),
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       appID,
		WorkflowID:  fnID,
		RunID:       runID,
		StartedAt:   startedAt,
		ExecutedAt:  startedAt.Add(time.Second),
		EventIds:    []byte(batchEventID.String() + "," + eventID.String()),
	}))

	rows, err := q.GetRuns(ctx, sqlc.GetRunsParams{EventIDText: eventID.String(), EventID: eventID, LimitRows: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Equal(t, "completed", rows[0].FunctionFinish.Status.String)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.Equal(t, []byte(`{"data":{"ok":true}}`), rows[0].RunOutput)

	rows, err = q.GetRuns(ctx, sqlc.GetRunsParams{
		EventIDText: batchEventID.String(),
		EventID:     batchEventID,
		LimitRows:   1,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
}

func TestGetRunsError(t *testing.T) {
	ctx := context.Background()
	conn, err := sqlite.Open(ctx, sqlite.Options{ForTest: true})
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	_, err = sqlc.New(conn).GetRuns(ctx, sqlc.GetRunsParams{
		EventIDText: ulid.Make().String(),
		EventID:     ulid.Make(),
		LimitRows:   1,
	})
	require.Error(t, err)
}

func TestGetRunsDriverErrors(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.Make()

	tests := []struct {
		driverName string
		driver     getRunsDriver
	}{
		{
			driverName: "sqlite-sqlc-get-runs-scan-error",
			driver: getRunsDriver{rows: [][]driver.Value{{
				"not-a-run-id",
			}}},
		},
		{
			driverName: "sqlite-sqlc-get-runs-row-error",
			driver: getRunsDriver{
				nextErr: errors.New("rows failed"),
			},
		},
		{
			driverName: "sqlite-sqlc-get-runs-close-error",
			driver: getRunsDriver{
				closeErr: errors.New("close failed"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.driverName, func(t *testing.T) {
			sql.Register(test.driverName, test.driver)

			conn, err := sql.Open(test.driverName, "")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })

			_, err = sqlc.New(conn).GetRuns(ctx, sqlc.GetRunsParams{
				EventIDText: eventID.String(),
				EventID:     eventID,
				LimitRows:   1,
			})
			require.Error(t, err)
		})
	}
}

type getRunsDriver struct {
	rows     [][]driver.Value
	nextErr  error
	closeErr error
}

func (d getRunsDriver) Open(name string) (driver.Conn, error) {
	return getRunsConn(d), nil
}

type getRunsConn getRunsDriver

func (c getRunsConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c getRunsConn) Close() error                              { return nil }
func (c getRunsConn) Begin() (driver.Tx, error)                 { return nil, nil }

func (c getRunsConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return &getRunsRows{
		rows:     c.rows,
		nextErr:  c.nextErr,
		closeErr: c.closeErr,
	}, nil
}

type getRunsRows struct {
	rows     [][]driver.Value
	pos      int
	nextErr  error
	closeErr error
}

func (r *getRunsRows) Columns() []string {
	return []string{
		"run_id",
		"run_started_at",
		"function_id",
		"function_version",
		"trigger_type",
		"event_id",
		"batch_id",
		"original_run_id",
		"cron",
		"workspace_id",
		"finish_run_id",
		"finish_status",
		"finish_output",
		"finish_completed_step_count",
		"finish_created_at",
		"function_slug",
		"function_name",
		"function_config",
		"function_app_id",
		"app_name",
		"run_output",
	}
}

func (r *getRunsRows) Close() error { return r.closeErr }

func (r *getRunsRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		if r.nextErr != nil {
			return r.nextErr
		}

		return io.EOF
	}

	copy(dest, r.rows[r.pos])
	r.pos++
	return nil
}
