package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQuerierGetRuns(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	batchID := ulid.Make()
	originalRunID := ulid.Make()
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	finishedAt := startedAt.Add(time.Second)

	sql.Register("postgres-get-runs", &getRunsDriver{
		rows: [][]driver.Value{{
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
		}},
		outputRows: [][]driver.Value{{
			runID.String(),
			[]byte(`{"data":{"ok":true}}`),
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
	require.Equal(t, "completed", rows[0].FunctionFinish.Status.String)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.JSONEq(t, `{"data":{"ok":true}}`, string(rows[0].Output))
}

func TestQuerierGetRunsSkipsOutputQuery(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)

	driver := &getRunsDriver{
		rows: [][]driver.Value{{
			runID.String(),
			startedAt,
			functionID.String(),
			int64(1),
			"event",
			eventID.String(),
			nil,
			nil,
			nil,
			"completed",
			"",
			int64(1),
			startedAt.Add(time.Second),
			"event-runs-app-event-runs-function",
			"Event Runs Function",
			`{"name":"Event Runs Function","slug":"event-runs-function"}`,
			appID.String(),
			"event-runs-app",
			[]byte{},
		}},
		outputRows: [][]driver.Value{{
			runID.String(),
			[]byte(`{"data":{"ok":true}}`),
		}},
	}
	sql.Register("postgres-get-runs-no-output", driver)

	conn, err := sql.Open("postgres-get-runs-no-output", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	rows, err := New(conn).Q().GetRuns(context.Background(), db.GetRunsParams{
		EventID: eventID,
		Limit:   1,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Empty(t, rows[0].Output)
	require.Zero(t, driver.outputQueries())
}

func TestQuerierGetRunsIncludesBatchRunIDs(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	batchID := ulid.Make()
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)

	driver := &getRunsDriver{
		batchRows: [][]driver.Value{{
			batchID.String(),
			uuid.New().String(),
			uuid.New().String(),
			appID.String(),
			functionID.String(),
			runID.String(),
			startedAt,
			startedAt.Add(time.Second),
			[]byte(eventID.String()),
		}},
		rows: [][]driver.Value{{
			runID.String(),
			startedAt,
			functionID.String(),
			int64(1),
			"event",
			ulid.Make().String(),
			batchID.String(),
			nil,
			nil,
			"completed",
			"",
			int64(1),
			startedAt.Add(time.Second),
			"event-runs-app-event-runs-function",
			"Event Runs Function",
			`{"name":"Event Runs Function","slug":"event-runs-function"}`,
			appID.String(),
			"event-runs-app",
			[]byte{},
		}},
	}
	sql.Register("postgres-get-runs-batch", driver)

	conn, err := sql.Open("postgres-get-runs-batch", "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	rows, err := New(conn).Q().GetRuns(context.Background(), db.GetRunsParams{
		EventID: eventID,
		Limit:   1,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
}

func TestQuerierGetRunsError(t *testing.T) {
	eventID := ulid.Make()

	tests := []struct {
		name   string
		driver *getRunsDriver
	}{
		{
			name: "batch lookup error",
			driver: &getRunsDriver{
				batchQueryErr: errors.New("batch query failed"),
			},
		},
		{
			name: "runs query error",
			driver: &getRunsDriver{
				queryErr: errors.New("query failed"),
			},
		},
		{
			name: "output query error",
			driver: &getRunsDriver{
				rows: [][]driver.Value{{
					ulid.Make().String(),
					time.Now().UTC(),
					uuid.New().String(),
					int64(1),
					"event",
					eventID.String(),
					nil,
					nil,
					nil,
					"completed",
					"",
					int64(1),
					time.Now().UTC(),
					"event-runs-app-event-runs-function",
					"Event Runs Function",
					`{"name":"Event Runs Function","slug":"event-runs-function"}`,
					uuid.New().String(),
					"event-runs-app",
					[]byte{},
				}},
				outputQueryErr: errors.New("output query failed"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driverName := "postgres-get-runs-error-" + strings.ReplaceAll(test.name, " ", "-")
			sql.Register(driverName, test.driver)

			conn, err := sql.Open(driverName, "")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })

			_, err = New(conn).Q().GetRuns(context.Background(), db.GetRunsParams{
				EventID:       eventID,
				Limit:         1,
				IncludeOutput: test.name == "output query error",
			})
			require.Error(t, err)
		})
	}
}

type getRunsDriver struct {
	mu             sync.Mutex
	batchRows      [][]driver.Value
	rows           [][]driver.Value
	outputRows     [][]driver.Value
	batchQueryErr  error
	queryErr       error
	outputQueryErr error
	outputQueryN   int
}

func (d *getRunsDriver) outputQueries() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.outputQueryN
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
	if strings.Contains(query, "FROM event_batches") {
		if c.driver.batchQueryErr != nil {
			return nil, c.driver.batchQueryErr
		}

		return &getRunsRows{
			columns: []string{"id", "account_id", "workspace_id", "app_id", "workflow_id", "run_id", "started_at", "executed_at", "event_ids"},
			rows:    c.driver.batchRows,
		}, nil
	}

	if strings.Contains(query, "FROM trace_runs") {
		c.driver.mu.Lock()
		c.driver.outputQueryN++
		c.driver.mu.Unlock()

		if c.driver.outputQueryErr != nil {
			return nil, c.driver.outputQueryErr
		}

		return &getRunsRows{
			columns: []string{"run_id", "output"},
			rows:    c.driver.outputRows,
		}, nil
	}

	if c.driver.queryErr != nil {
		return nil, c.driver.queryErr
	}

	return &getRunsRows{columns: getRunsColumns(), rows: c.driver.rows}, nil
}

type getRunsRows struct {
	columns []string
	rows    [][]driver.Value
	pos     int
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

func (r *getRunsRows) Close() error { return nil }

func (r *getRunsRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}

	copy(dest, r.rows[r.pos])
	r.pos++
	return nil
}
