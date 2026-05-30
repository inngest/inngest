package sqlc_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/postgres/sqlc"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestGetRuns(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	batchID := ulid.Make()
	originalRunID := ulid.Make()
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	finishedAt := startedAt.Add(time.Second)

	registerGetRunsDriver(t, "postgres-sqlc-get-runs", getRunsDriver{rows: [][]driver.Value{{
		runID.String(),
		startedAt,
		functionID.String(),
		int64(1),
		"event",
		eventID.String(),
		batchID.String(),
		originalRunID.String(),
		nil,
		"completed",
		"",
		int64(1),
		finishedAt,
		"event-runs-app-event-runs-function",
		"Event Runs Function",
		`{"name":"Event Runs Function","slug":"event-runs-function"}`,
		appID.String(),
		"event-runs-app",
		[]byte{},
	}}})

	conn, err := sql.Open("postgres-sqlc-get-runs", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	rows, err := sqlc.New(conn).GetRuns(context.Background(), sqlc.GetRunsParams{
		EventID:    eventID[:],
		RunIds:     [][]byte{runID[:]},
		OffsetRows: 0,
		LimitRows:  1,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Equal(t, "completed", rows[0].FinishStatus)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.Empty(t, rows[0].RunOutput)
}

func TestGetTraceRunOutputs(t *testing.T) {
	runID := ulid.Make()

	registerGetRunsDriver(t, "postgres-sqlc-get-trace-run-outputs", getRunsDriver{outputRows: [][]driver.Value{{
		runID.String(),
		[]byte(`{"data":{"ok":true}}`),
	}}})

	conn, err := sql.Open("postgres-sqlc-get-trace-run-outputs", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	rows, err := sqlc.New(conn).GetTraceRunOutputs(context.Background(), []string{runID.String()})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].RunID)
	require.Equal(t, []byte(`{"data":{"ok":true}}`), rows[0].Output)
}

func TestGetTraceRunOutputsErrors(t *testing.T) {
	runID := ulid.Make()

	tests := []struct {
		name   string
		driver getRunsDriver
	}{
		{
			name: "query error",
			driver: getRunsDriver{
				queryErr: errors.New("query failed"),
			},
		},
		{
			name: "scan error",
			driver: getRunsDriver{outputRows: [][]driver.Value{{
				"not-a-run-id",
			}}},
		},
		{
			name: "row error",
			driver: getRunsDriver{
				nextErr: errors.New("rows failed"),
			},
		},
		{
			name: "close error",
			driver: getRunsDriver{
				closeErr: errors.New("close failed"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driverName := "postgres-sqlc-get-trace-run-outputs-" + test.name
			registerGetRunsDriver(t, driverName, test.driver)

			conn, err := sql.Open(driverName, "")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })

			_, err = sqlc.New(conn).GetTraceRunOutputs(context.Background(), []string{runID.String()})
			require.Error(t, err)
		})
	}
}

func TestGetRunsErrors(t *testing.T) {
	eventID := ulid.Make()

	tests := []struct {
		name   string
		driver getRunsDriver
	}{
		{
			name: "query error",
			driver: getRunsDriver{
				queryErr: errors.New("query failed"),
			},
		},
		{
			name: "scan error",
			driver: getRunsDriver{rows: [][]driver.Value{{
				"not-a-run-id",
			}}},
		},
		{
			name: "row error",
			driver: getRunsDriver{
				nextErr: errors.New("rows failed"),
			},
		},
		{
			name: "close error",
			driver: getRunsDriver{
				closeErr: errors.New("close failed"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driverName := "postgres-sqlc-get-runs-" + test.name
			registerGetRunsDriver(t, driverName, test.driver)

			conn, err := sql.Open(driverName, "")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })

			_, err = sqlc.New(conn).GetRuns(context.Background(), sqlc.GetRunsParams{
				EventID:    eventID[:],
				OffsetRows: 0,
				LimitRows:  1,
			})
			require.Error(t, err)
		})
	}
}

func registerGetRunsDriver(t *testing.T, name string, driver getRunsDriver) {
	t.Helper()

	sql.Register(name, driver)
}

type getRunsDriver struct {
	rows       [][]driver.Value
	outputRows [][]driver.Value
	queryErr   error
	nextErr    error
	closeErr   error
}

func (d getRunsDriver) Open(name string) (driver.Conn, error) {
	return getRunsConn(d), nil
}

type getRunsConn getRunsDriver

func (c getRunsConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c getRunsConn) Close() error                              { return nil }
func (c getRunsConn) Begin() (driver.Tx, error)                 { return nil, nil }

func (c getRunsConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.queryErr != nil {
		return nil, c.queryErr
	}

	if strings.Contains(query, "FROM trace_runs") {
		return &getRunsRows{
			columns:  []string{"run_id", "output"},
			rows:     c.outputRows,
			nextErr:  c.nextErr,
			closeErr: c.closeErr,
		}, nil
	}

	return &getRunsRows{
		columns:  getRunsColumns(),
		rows:     c.rows,
		nextErr:  c.nextErr,
		closeErr: c.closeErr,
	}, nil
}

type getRunsRows struct {
	columns  []string
	rows     [][]driver.Value
	pos      int
	nextErr  error
	closeErr error
}

func (r *getRunsRows) Columns() []string {
	return r.columns
}

func getRunsColumns() []string {
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
