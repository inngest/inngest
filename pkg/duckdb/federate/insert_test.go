package federate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestInsertStream checks a streamed delta lands in the lake table, each
// column in its own (BY NAME) and cast to its lake type.
func TestInsertStream(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	done := f.base.Add(time.Minute)
	f.deltaRun("run-a", "Completed", f.base, &done, `{"ok":1}`)
	f.deltaRun("run-b", "Running", f.base.Add(time.Second), nil, "")
	for _, row := range f.streamer.rows[TableRuns] {
		// The lake requires these.
		row["scheduled_at"], row["is_deferred"] = row["queued_at"], false
	}
	r, err := f.streamer.Stream(ctx, DeltaRequest{Table: TableRuns, Columns: DeltaColumns(TableRuns)})
	require.NoError(t, err)

	conn, err := f.db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()
	n, err := InsertStream(ctx, conn, "inngest", TableRuns, r)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)

	rows, err := conn.QueryContext(ctx, `SELECT run_id, status, coalesce(output::JSON->>'ok', '') FROM inngest.runs ORDER BY run_id`)
	require.NoError(t, err)
	defer rows.Close()
	var got [][]string
	for rows.Next() {
		var id, status, ok string
		require.NoError(t, rows.Scan(&id, &status, &ok))
		got = append(got, []string{id, status, ok})
	}
	require.NoError(t, rows.Err())
	require.Equal(t, [][]string{{"run-a", "Completed", "1"}, {"run-b", "Running", ""}}, got)
}
