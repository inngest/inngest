package manager

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestPostgresTraceRunReaderBenchmark compares the legacy trace_runs readers
// with their spans-backed replacements against the same data. It is opt-in
// because preparing the PostgreSQL fixture is intentionally expensive.
func TestPostgresTraceRunReaderBenchmark(t *testing.T) {
	if testing.Short() || os.Getenv("RUN_TRACE_READER_BENCHMARK") != "1" {
		t.Skip("set RUN_TRACE_READER_BENCHMARK=1 to run the trace reader benchmark")
	}

	t.Setenv(EnvTestDatabase, "postgres")
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()

	const (
		runCount   = 10_000
		spanFanout = 10
	)
	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	eventID := ulid.MustNew(ulid.Timestamp(baseTime), rand.Reader)
	pointRunID := ulid.MustParse(strings.Repeat("0", 22) + "5000")

	db := cm.(wrapper).adapter.Conn()
	_, err := db.ExecContext(ctx, `
		INSERT INTO trace_runs (
			run_id, account_id, workspace_id, app_id, function_id, trace_id,
			queued_at, started_at, ended_at, status, source_id, trigger_ids,
			output, is_debounce, batch_id, cron_schedule, has_ai
		)
		SELECT
			LPAD(i::text, 26, '0'), $1, $2, $3, $4, decode(md5(i::text), 'hex'),
			$5::bigint + i, $5::bigint + i, $5::bigint + i + 100, $6, '',
			convert_to(CASE WHEN i = 5000 THEN $7 ELSE '[]' END, 'UTF8'),
			'{}', false, NULL, NULL, false
		FROM generate_series(1, $8::int) AS i
	`, accountID, workspaceID, appID, functionID, baseTime.UnixMilli(), enums.RunStatusCompleted.ToCode(), fmt.Sprintf(`["%s"]`, eventID), runCount)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO spans (
			span_id, trace_id, parent_span_id, name, start_time, end_time,
			attributes, links, dynamic_span_id, account_id, app_id, function_id,
			run_id, env_id, output, status, event_ids
		)
		SELECT
			SUBSTRING(md5(i::text || ':' || fragment::text), 1, 16), md5(i::text), NULL,
			CASE WHEN fragment = 0 THEN 'executor.run' ELSE 'EXTEND' END,
			$5::timestamptz + i * interval '1 millisecond' + fragment * interval '1 microsecond',
			$5::timestamptz + i * interval '1 millisecond' + interval '100 milliseconds',
			'{}'::jsonb, '[]'::jsonb, 'dyn-' || i::text, $1, $3, $4,
			LPAD(i::text, 26, '0'), $2, CASE WHEN fragment = 0 THEN '{}'::jsonb END,
			'Completed', CASE WHEN i = 5000 AND fragment = 0 THEN $6::jsonb ELSE '[]'::jsonb END
		FROM generate_series(1, $7::int) AS i
		CROSS JOIN generate_series(0, $8::int - 1) AS fragment
	`, accountID.String(), workspaceID.String(), appID.String(), functionID.String(), baseTime, fmt.Sprintf(`["%s"]`, eventID), runCount, spanFanout)
	require.NoError(t, err)

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: workspaceID,
			TimeField: enums.TraceRunTimeStartedAt,
			From:      baseTime, Until: baseTime.Add(2 * time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc}},
		Items: 50,
	}

	bench := func(name string, fn func() error) {
		t.Helper()
		result := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				if err := fn(); err != nil {
					b.Fatal(err)
				}
			}
		})
		t.Logf("%s: %s", name, result.String())
	}

	bench("legacy list", func() error {
		_, err := cm.GetTraceRuns(ctx, opt)
		return err
	})
	bench("spans list", func() error {
		_, err := cm.GetRuns(ctx, opt)
		return err
	})
	bench("legacy count", func() error {
		_, err := cm.GetTraceRunsCount(ctx, opt)
		return err
	})
	bench("spans count", func() error {
		_, err := cm.(wrapper).getSpanRunsCount(ctx, opt)
		return err
	})

	endedOpt := opt
	endedOpt.Filter.TimeField = enums.TraceRunTimeEndedAt
	endedOpt.Order = []cqrs.GetTraceRunOrder{{Field: enums.TraceRunTimeEndedAt, Direction: enums.TraceRunOrderDesc}}
	bench("legacy ended list", func() error {
		_, err := cm.GetTraceRuns(ctx, endedOpt)
		return err
	})
	bench("spans ended list", func() error {
		_, err := cm.GetRuns(ctx, endedOpt)
		return err
	})
	bench("legacy ended count", func() error {
		_, err := cm.GetTraceRunsCount(ctx, endedOpt)
		return err
	})
	bench("spans ended count", func() error {
		_, err := cm.(wrapper).getSpanRunsCount(ctx, endedOpt)
		return err
	})

	statusOpt := opt
	statusOpt.Filter.Status = []enums.RunStatus{enums.RunStatusCompleted}
	bench("legacy status list", func() error {
		_, err := cm.GetTraceRuns(ctx, statusOpt)
		return err
	})
	bench("spans status list", func() error {
		_, err := cm.GetRuns(ctx, statusOpt)
		return err
	})
	bench("legacy status count", func() error {
		_, err := cm.GetTraceRunsCount(ctx, statusOpt)
		return err
	})
	bench("spans status count", func() error {
		_, err := cm.(wrapper).getSpanRunsCount(ctx, statusOpt)
		return err
	})

	bench("legacy point", func() error {
		_, err := cm.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: pointRunID})
		return err
	})
	bench("spans point", func() error {
		_, err := cm.GetRunSpanByRunID(ctx, pointRunID, accountID, workspaceID)
		return err
	})
	bench("legacy event", func() error {
		_, err := cm.GetTraceRunsByTriggerID(ctx, eventID)
		return err
	})
	eventOpt := opt
	eventOpt.Filter.EventID = []ulid.ULID{eventID}
	bench("spans event", func() error {
		_, err := cm.GetRuns(ctx, eventOpt)
		return err
	})
}
