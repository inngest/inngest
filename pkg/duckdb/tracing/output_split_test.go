package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/duckdb/query"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/stretchr/testify/require"
)

// stepPayload has unsorted keys, 1.10 and an int past 2^64, so a re-marshal
// anywhere between the SDK and the reader would show up.
const stepPayload = `{"b":1,"a":1.10,"n":12345678901234567890123}`

// TestOutputSplitEndToEnd drives a successful step, a failed step and a
// failed run through the listener's hooks -> the lake -> the UI/GQL
// read path and Insights, checking output/error are split once at write
// time and read back as stored.
func TestOutputSplitEndToEnd(t *testing.T) {
	db := openOutputSplitLake(t)
	md := writeOutputSplitRun(t, db)
	ctx := t.Context()

	t.Run("columns", func(t *testing.T) {
		rows := selectRows(t, db, "SELECT span_id, output::VARCHAR AS out, error::VARCHAR AS err FROM inngest.run_trace_spans WHERE name = ?;", tracingv3.SpanNameStep)
		got := map[string][2]any{}
		for _, r := range rows {
			got[r["span_id"].(string)] = [2]any{r["out"], r["err"]}
		}
		require.Equal(t, map[string][2]any{
			stepSpanID(md, "step-ok"):   {stepPayload, nil},
			stepSpanID(md, "step-fail"): {nil, `{"name":"Error","message":"boom"}`},
		}, got)

		runs := selectRows(t, db, "SELECT output::VARCHAR AS out, error::VARCHAR AS err FROM inngest.runs WHERE status = ?;", enums.StepStatusFailed.String())
		require.Len(t, runs, 1)
		require.Nil(t, runs[0]["out"])
		require.Equal(t, `{"name":"Error","message":"fn boom"}`, runs[0]["err"])
	})

	t.Run("reader", func(t *testing.T) {
		m := query.Wrap(nil, db)

		out, err := m.GetSpanOutput(ctx, cqrs.SpanIdentifier{RunID: md.ID.RunID.String(), SpanID: stepSpanID(md, "step-ok")})
		require.NoError(t, err)
		require.False(t, out.IsError)
		require.Equal(t, stepPayload, string(out.Data))

		out, err = m.GetSpanOutput(ctx, cqrs.SpanIdentifier{RunID: md.ID.RunID.String(), SpanID: stepSpanID(md, "step-fail")})
		require.NoError(t, err)
		require.True(t, out.IsError)
		require.Equal(t, `{"name":"Error","message":"boom"}`, string(out.Data))

		run, err := m.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: md.ID.RunID})
		require.NoError(t, err)
		require.Equal(t, enums.RunStatusFailed, run.Status)
		require.Equal(t, `{"error":{"name":"Error","message":"fn boom"}}`, string(run.Output))
	})

	t.Run("insights", func(t *testing.T) {
		accountID, envID := md.ID.Tenant.AccountID, md.ID.Tenant.EnvID
		for _, tc := range []struct {
			sql  string
			want [][]any
		}{
			{
				`SELECT span_id, error.message AS msg FROM step_attempts WHERE error IS NOT NULL`,
				[][]any{{stepSpanID(md, "step-fail"), "boom"}},
			},
			{
				`SELECT span_id, output::VARCHAR AS out FROM step_attempts WHERE error IS NULL`,
				[][]any{{stepSpanID(md, "step-ok"), stepPayload}},
			},
			{
				`SELECT status, error.message AS msg, output FROM runs WHERE error IS NOT NULL`,
				[][]any{{enums.StepStatusFailed.String(), "fn boom", nil}},
			},
		} {
			t.Run(tc.sql, func(t *testing.T) {
				tr, err := insights.Transpile(tc.sql, accountID, envID)
				require.NoError(t, err)
				res, err := insights.Execute(ctx, db, tr)
				require.NoError(t, err)
				require.Equal(t, tc.want, res.Rows)
			})
		}
	})
}

func openOutputSplitLake(t *testing.T) *sql.DB {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	db, err := driver.Open(t.Context(), driver.Options{BinaryPath: binPath, DBFile: ":memory:", DuckLake: &driver.DuckLakeOptions{
		CatalogPath: filepath.Join(dir, "catalog.duckdb"),
		DataPath:    filepath.Join(dir, "data"),
	}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, duckdb.Migrate(t.Context(), db, duckdb.ModeRawLake))
	return db
}

// writeOutputSplitRun writes one run through the listener: a step.run that
// returns stepPayload, one that fails w/ "boom", and the run failing w/
// "fn boom". It returns once every span and the final run row landed.
func writeOutputSplitRun(t *testing.T, db *sql.DB) sv2.Metadata {
	t.Helper()
	ctx := context.Background()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })
	// Stop the batching workers (they run on context.Background()) before the
	// db cleanup registered earlier closes the db.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, l.(Closer).Close(ctx))
	})
	md := testMetadata(t)
	now := time.Now()

	l.OnStepRunFinished(ctx, md, queue.Item{}, inngest.Edge{}, statev1.GeneratorOpcode{
		Op: enums.OpcodeStepRun, ID: "step-ok", Name: "ok", Data: json.RawMessage(stepPayload),
	}, now)
	l.OnStepRunFailed(ctx, md, queue.Item{}, inngest.Edge{}, statev1.GeneratorOpcode{
		Op: enums.OpcodeStepError, ID: "step-fail", Name: "fail",
		Error: &statev1.UserError{Name: "Error", Message: "boom"},
	}, enums.StepStatusFailed, 3, now)
	fnErr := "fn boom"
	l.OnFunctionFinished(ctx, md, queue.Item{}, nil, statev1.DriverResponse{
		Err:    &fnErr,
		Output: `{"name":"Error","message":"fn boom"}`,
	}, now)

	require.Eventually(t, func() bool {
		var spans, runs int
		_ = db.QueryRowContext(ctx, "SELECT count(*) FROM inngest.run_trace_spans WHERE name = ?;", tracingv3.SpanNameStep).Scan(&spans)
		_ = db.QueryRowContext(ctx, "SELECT count(*) FROM inngest.runs;").Scan(&runs)
		return spans == 2 && runs == 1
	}, 10*time.Second, 20*time.Millisecond, "step spans and the run row should land after a batch flush")
	return md
}

func stepSpanID(md sv2.Metadata, stepID string) string {
	return tracing.FinalizedStepSpanRefFromMetadataAndStepID(&md, stepID).DynamicSpanID
}
