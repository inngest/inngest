package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQuerierGetRuns(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	batchID := ulid.Make()
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	endedAt := startedAt.Add(time.Second)

	sql.Register("postgres-get-runs", &getRunsDriver{
		rows: [][]driver.Value{{
			runID.String(),
			functionID.String(),
			appID.String(),
			startedAt,
			endedAt,
			enums.StepStatusCompleted.String(),
			`{"data":{"ok":true}}`,
			"event-runs-function",
			"Event Runs Function",
			"event-runs-app",
			batchID.String(),
			"*/5 * * * *",
		}},
	})

	conn, err := sql.Open("postgres-get-runs", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	rows, err := New(conn).Q().GetRuns(context.Background(), db.GetRunsParams{
		EventID:       eventID,
		Limit:         1,
		IncludeOutput: true,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Equal(t, eventID, rows[0].FunctionRun.EventID)
	require.Equal(t, batchID, rows[0].FunctionRun.BatchID)
	require.Equal(t, "*/5 * * * *", rows[0].FunctionRun.Cron.String)
	require.Equal(t, "Completed", rows[0].FunctionFinish.Status.String)
	require.True(t, rows[0].FunctionFinish.CreatedAt.Valid)
	require.Equal(t, "event-runs-function", rows[0].FunctionSlug)
	require.Equal(t, "Event Runs Function", rows[0].FunctionName)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.JSONEq(t, `{"data":{"ok":true}}`, string(rows[0].Output))
}

func TestQuerierGetRunsSkipsOutput(t *testing.T) {
	runID := ulid.Make()
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)

	sql.Register("postgres-get-runs-no-output", &getRunsDriver{
		rows: [][]driver.Value{{
			runID.String(),
			functionID.String(),
			appID.String(),
			startedAt,
			startedAt.Add(time.Second),
			enums.StepStatusCompleted.String(),
			`{"data":{"ok":true}}`,
			"event-runs-function",
			"Event Runs Function",
			"event-runs-app",
			"",
			"",
		}},
	})

	conn, err := sql.Open("postgres-get-runs-no-output", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	rows, err := New(conn).Q().GetRuns(context.Background(), db.GetRunsParams{
		EventID: ulid.Make(),
		Limit:   1,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Empty(t, rows[0].Output)
}

func TestQuerierGetRunsError(t *testing.T) {
	sql.Register("postgres-get-runs-error", &getRunsDriver{queryErr: errors.New("query failed")})

	conn, err := sql.Open("postgres-get-runs-error", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	_, err = New(conn).Q().GetRuns(context.Background(), db.GetRunsParams{
		EventID: ulid.Make(),
		Limit:   1,
	})
	require.Error(t, err)
}

type getRunsDriver struct {
	rows     [][]driver.Value
	queryErr error
}

func (d *getRunsDriver) Open(name string) (driver.Conn, error) {
	return getRunsConn{driver: d}, nil
}

type getRunsConn struct {
	driver *getRunsDriver
}

func (c getRunsConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c getRunsConn) Close() error                              { return nil }
func (c getRunsConn) Begin() (driver.Tx, error)                 { return nil, nil }

func (c getRunsConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.driver.queryErr != nil {
		return nil, c.driver.queryErr
	}
	return &getRunsRows{
		columns: []string{
			"run_id",
			"function_id",
			"app_id",
			"start_time",
			"end_time",
			"status",
			"output",
			"function_slug",
			"function_name",
			"app_name",
			"batch_id",
			"cron_schedule",
		},
		rows: c.driver.rows,
	}, nil
}

func (c getRunsConn) CheckNamedValue(value *driver.NamedValue) error {
	return nil
}

type getRunsRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *getRunsRows) Columns() []string { return r.columns }
func (r *getRunsRows) Close() error      { return nil }

func (r *getRunsRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

var _ driver.QueryerContext = getRunsConn{}
var _ driver.NamedValueChecker = getRunsConn{}
