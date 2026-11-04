package tracing

import (
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog/icebergtest"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/duckdb/query"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestJSONRoundTripsByteForByte checks a JSON payload w/ unsorted keys, odd
// whitespace, 1.10 and an int past INT64_MAX comes back exactly as written
// through the dual-write inserter (and materializeRuns) -> the Iceberg lake
// -> the UI/GQL read path and Insights, the way ClickHouse's String columns
// return it.
func TestJSONRoundTripsByteForByte(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	db, err := driver.Open(t.Context(), driver.Options{BinaryPath: binPath, DBFile: ":memory:", Iceberg: icebergtest.Start(t)})
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, duckdb.Migrate(t.Context(), db, duckdb.ModeRawLake))

	const payload = `{ "b" : 1 ,  "a":1.10, "n":12345678901234567890123, "z":{"y":[ 2, 1 ]} }`
	const input = `[{"name":"x" ,"data":` + payload + `}]`
	accountID, envID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.Make()
	now := time.Now().UTC().Truncate(time.Millisecond)
	// Output is already split (spanExportRow splits the envelope).
	span := func(spanID, name string, output, input string) Span {
		return Span{
			AccountID: accountID.String(), EnvID: envID.String(), AppID: appID.String(), AppName: "app",
			FunctionID: fnID.String(), FunctionSlug: "fn", RunID: runID.String(), RunQueuedAt: now,
			TraceID: "trace-1", SpanID: spanID, Name: name, StartTime: now, EndTime: now,
			Attributes: json.RawMessage(`{"_inngest.dynamic.status":"Completed","_inngest.step.userland.id":"s"}`),
			Output:     json.RawMessage(output), Input: json.RawMessage(input),
		}
	}
	require.NoError(t, NewDuckDBInserter(db).InsertSpans(t.Context(), []Span{
		span("run-span", "executor.run", payload, input),
		span("step-span", "executor.step", payload, payload),
	}))

	t.Run("ui", func(t *testing.T) {
		m := query.Wrap(nil, db)
		out, err := m.GetSpanOutput(t.Context(), cqrs.SpanIdentifier{RunID: runID.String(), SpanID: "step-span"})
		require.NoError(t, err)
		require.Equal(t, payload, string(out.Data))
		require.Equal(t, payload, string(out.Input))

		run, err := m.GetTraceRun(t.Context(), cqrs.TraceRunIdentifier{RunID: runID})
		require.NoError(t, err)
		// TraceRun.Output is re-wrapped in the envelope, value bytes as is.
		require.Equal(t, `{"data":`+payload+`}`, string(run.Output))
	})

	t.Run("insights", func(t *testing.T) {
		// A JSON cell itself is decoded by the driver (into Go values), so
		// read the stored text back w/ ::VARCHAR.
		tr, err := insights.Transpile(`SELECT s.output::VARCHAR, s.input::VARCHAR, r.output::VARCHAR, r.inputs::VARCHAR FROM steps s JOIN runs r ON s.run_id = r.run_id`, accountID, envID)
		require.NoError(t, err)
		res, err := insights.Execute(t.Context(), db, tr)
		require.NoError(t, err)
		require.Len(t, res.Rows, 1)
		require.Equal(t, []any{payload, payload, payload, input}, res.Rows[0])
	})
}
