package manager

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestPostgresV2RunProjectionBenchmark(t *testing.T) {
	if testing.Short() || os.Getenv("RUN_TRACE_READER_BENCHMARK") != "1" {
		t.Skip("set RUN_TRACE_READER_BENCHMARK=1 to run the trace reader benchmark")
	}

	t.Setenv(EnvTestDatabase, "postgres")
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()
	db := cm.(wrapper).adapter.Conn()

	const (
		runCount   = 10_000
		spanFanout = 10
	)
	accountID := uuid.New().String()
	envID := uuid.New().String()
	appID := uuid.New().String()
	functionID := uuid.New().String()
	eventID := ulid.Make().String()
	baseTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	insertSpans := func() time.Duration {
		t.Helper()
		started := time.Now()
		_, err := db.ExecContext(ctx, `
			INSERT INTO spans (
				span_id, trace_id, parent_span_id, name, start_time, end_time,
				attributes, links, dynamic_span_id, account_id, app_id, function_id,
				run_id, env_id, output, status, event_ids
			)
			SELECT
				SUBSTRING(md5(i::text || ':' || fragment::text), 1, 16), md5(i::text), NULL,
				CASE WHEN fragment = 0 THEN 'executor.run' ELSE 'EXTEND' END,
				$5::timestamptz + i * interval '1 millisecond' + fragment * interval '1 microsecond',
				$5::timestamptz + i * interval '1 millisecond' + interval '100 milliseconds'
					+ fragment * interval '1 microsecond',
				'{}'::jsonb, '[]'::jsonb, 'dyn-' || i::text, $1, $3, $4,
				LPAD(i::text, 26, '0'), $2,
				CASE WHEN fragment = $8::int - 1 THEN jsonb_build_object('result', i) END,
				CASE WHEN fragment = $8::int - 1 THEN 'Completed' ELSE 'Running' END,
				CASE WHEN i = 5000 AND fragment = 0 THEN $6::jsonb ELSE '[]'::jsonb END
			FROM generate_series(1, $7::int) AS i
			CROSS JOIN generate_series(0, $8::int - 1) AS fragment
		`, accountID, envID, appID, functionID, baseTime, fmt.Sprintf(`["%s"]`, eventID), runCount, spanFanout)
		require.NoError(t, err)
		return time.Since(started)
	}

	withoutProjection := insertSpans()

	_, err := db.ExecContext(ctx, `
		CREATE TABLE trace_runs_v2 (
			run_id TEXT PRIMARY KEY,
			account_id TEXT NOT NULL,
			env_id TEXT NOT NULL,
			app_id TEXT NOT NULL,
			function_id TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			queued_at TIMESTAMPTZ NOT NULL,
			started_at TIMESTAMPTZ NOT NULL,
			ended_at TIMESTAMPTZ,
			status TEXT NOT NULL,
			status_updated_at TIMESTAMPTZ NOT NULL,
			event_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
			output JSONB,
			output_updated_at TIMESTAMPTZ
		);
		CREATE INDEX trace_runs_v2_started
			ON trace_runs_v2 (account_id, env_id, started_at DESC, run_id);
		CREATE INDEX trace_runs_v2_ended
			ON trace_runs_v2 (account_id, env_id, ended_at DESC, run_id);
		CREATE INDEX trace_runs_v2_status_started
			ON trace_runs_v2 (account_id, env_id, status, started_at DESC, run_id);
		CREATE INDEX trace_runs_v2_event_ids ON trace_runs_v2 USING GIN (event_ids);

		CREATE FUNCTION project_v2_run() RETURNS trigger AS $$
		DECLARE
			root_dynamic_span_id TEXT;
		BEGIN
			IF NEW.debug_run_id IS NOT NULL THEN
				RETURN NEW;
			END IF;

			PERFORM pg_advisory_xact_lock(hashtextextended(NEW.run_id, 0));

			SELECT dynamic_span_id INTO root_dynamic_span_id
			FROM spans
			WHERE run_id = NEW.run_id AND name = 'executor.run' AND debug_run_id IS NULL
			ORDER BY start_time ASC
			LIMIT 1;

			IF root_dynamic_span_id IS NULL THEN
				RETURN NEW;
			END IF;

			IF NEW.name = 'executor.run' THEN
				INSERT INTO trace_runs_v2 (
					run_id, account_id, env_id, app_id, function_id, trace_id,
					queued_at, started_at, ended_at, status, status_updated_at,
					event_ids, output, output_updated_at
				)
				SELECT
					root.run_id, root.account_id, root.env_id, root.app_id, root.function_id, root.trace_id,
					root.start_time,
					MIN(fragment.start_time),
					MAX(fragment.end_time),
					(ARRAY_AGG(fragment.status ORDER BY fragment.end_time DESC, fragment.span_id DESC)
						FILTER (WHERE fragment.status IS NOT NULL))[1],
					MAX(fragment.end_time) FILTER (WHERE fragment.status IS NOT NULL),
					CASE jsonb_typeof(root.event_ids)
						WHEN 'string' THEN COALESCE((root.event_ids #>> '{}')::jsonb, '[]'::jsonb)
						ELSE COALESCE(root.event_ids, '[]'::jsonb)
					END,
					(ARRAY_AGG(fragment.output ORDER BY fragment.end_time DESC)
						FILTER (WHERE fragment.output IS NOT NULL))[1],
					MAX(fragment.end_time) FILTER (WHERE fragment.output IS NOT NULL)
				FROM spans root
				JOIN spans fragment
					ON fragment.run_id = root.run_id
					AND fragment.dynamic_span_id = root.dynamic_span_id
				WHERE root.run_id = NEW.run_id
					AND root.name = 'executor.run'
					AND root.debug_run_id IS NULL
				GROUP BY root.run_id, root.account_id, root.env_id, root.app_id,
					root.function_id, root.trace_id, root.start_time, root.event_ids
				ON CONFLICT (run_id) DO UPDATE SET
					account_id = EXCLUDED.account_id,
					env_id = EXCLUDED.env_id,
					app_id = EXCLUDED.app_id,
					function_id = EXCLUDED.function_id,
					trace_id = EXCLUDED.trace_id,
					queued_at = EXCLUDED.queued_at,
					started_at = LEAST(trace_runs_v2.started_at, EXCLUDED.started_at),
					ended_at = GREATEST(trace_runs_v2.ended_at, EXCLUDED.ended_at),
					status = CASE WHEN EXCLUDED.status_updated_at >= trace_runs_v2.status_updated_at
						THEN EXCLUDED.status ELSE trace_runs_v2.status END,
					status_updated_at = GREATEST(trace_runs_v2.status_updated_at, EXCLUDED.status_updated_at),
					event_ids = EXCLUDED.event_ids,
					output = CASE WHEN EXCLUDED.output_updated_at >= trace_runs_v2.output_updated_at
						THEN EXCLUDED.output ELSE trace_runs_v2.output END,
					output_updated_at = GREATEST(trace_runs_v2.output_updated_at, EXCLUDED.output_updated_at);
			ELSIF NEW.dynamic_span_id = root_dynamic_span_id THEN
				UPDATE trace_runs_v2 SET
					started_at = LEAST(started_at, NEW.start_time),
					ended_at = GREATEST(ended_at, NEW.end_time),
					status = CASE WHEN NEW.status IS NOT NULL AND NEW.end_time >= status_updated_at
						THEN NEW.status ELSE status END,
					status_updated_at = CASE WHEN NEW.status IS NOT NULL
						THEN GREATEST(status_updated_at, NEW.end_time) ELSE status_updated_at END,
					output = CASE WHEN NEW.output IS NOT NULL
							AND (output_updated_at IS NULL OR NEW.end_time >= output_updated_at)
						THEN NEW.output ELSE output END,
					output_updated_at = CASE WHEN NEW.output IS NOT NULL
						THEN GREATEST(output_updated_at, NEW.end_time) ELSE output_updated_at END
				WHERE run_id = NEW.run_id;
			END IF;

			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER spans_project_v2_run
		AFTER INSERT ON spans
		FOR EACH ROW EXECUTE FUNCTION project_v2_run();
	`)
	require.NoError(t, err)

	backfillStarted := time.Now()
	_, err = db.ExecContext(ctx, `
		INSERT INTO trace_runs_v2 (
			run_id, account_id, env_id, app_id, function_id, trace_id,
			queued_at, started_at, ended_at, status, status_updated_at,
			event_ids, output, output_updated_at
		)
		SELECT
			root.run_id, root.account_id, root.env_id, root.app_id, root.function_id, root.trace_id,
			root.start_time,
			MIN(fragment.start_time),
			MAX(fragment.end_time),
			(ARRAY_AGG(fragment.status ORDER BY fragment.end_time DESC, fragment.span_id DESC)
				FILTER (WHERE fragment.status IS NOT NULL))[1],
			MAX(fragment.end_time) FILTER (WHERE fragment.status IS NOT NULL),
			CASE jsonb_typeof(root.event_ids)
				WHEN 'string' THEN COALESCE((root.event_ids #>> '{}')::jsonb, '[]'::jsonb)
				ELSE COALESCE(root.event_ids, '[]'::jsonb)
			END,
			(ARRAY_AGG(fragment.output ORDER BY fragment.end_time DESC)
				FILTER (WHERE fragment.output IS NOT NULL))[1],
			MAX(fragment.end_time) FILTER (WHERE fragment.output IS NOT NULL)
		FROM spans root
		JOIN spans fragment
			ON fragment.run_id = root.run_id
			AND fragment.dynamic_span_id = root.dynamic_span_id
		WHERE root.name = 'executor.run' AND root.debug_run_id IS NULL
		GROUP BY root.run_id, root.account_id, root.env_id, root.app_id,
			root.function_id, root.trace_id, root.start_time, root.event_ids
	`)
	require.NoError(t, err)
	backfillDuration := time.Since(backfillStarted)
	var projected int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trace_runs_v2`).Scan(&projected))
	require.Equal(t, runCount, projected)
	t.Logf("set-based projection backfill:  %s", backfillDuration)
	if os.Getenv("TRACE_PROJECTION_BACKFILL_ONLY") == "1" {
		return
	}

	_, err = db.ExecContext(ctx, `TRUNCATE spans, trace_runs_v2`)
	require.NoError(t, err)
	withProjection := insertSpans()
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trace_runs_v2`).Scan(&projected))
	require.Equal(t, runCount, projected)

	_, err = db.ExecContext(ctx, `VACUUM ANALYZE trace_runs_v2`)
	require.NoError(t, err)
	t.Logf("span ingest without projection: %s", withoutProjection)
	t.Logf("span ingest with projection:    %s (%.2fx)", withProjection, float64(withProjection)/float64(withoutProjection))

	benchRows := func(name, query string, args ...any) {
		t.Helper()
		result := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				rows, err := db.QueryContext(ctx, query, args...)
				if err != nil {
					b.Fatal(err)
				}
				for rows.Next() {
					var runID string
					if err := rows.Scan(&runID); err != nil {
						b.Fatal(err)
					}
				}
				if err := rows.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
		t.Logf("%s: %s", name, result.String())
	}
	benchCount := func(name, query string, args ...any) {
		t.Helper()
		result := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				var count int
				if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
					b.Fatal(err)
				}
			}
		})
		t.Logf("%s: %s", name, result.String())
	}

	from := baseTime.Add(-time.Hour)
	until := baseTime.Add(2 * time.Hour)
	benchRows("projection list", `
		SELECT run_id FROM trace_runs_v2
		WHERE account_id = $1 AND env_id = $2 AND started_at >= $3 AND started_at < $4
		ORDER BY started_at DESC, run_id LIMIT 50`, accountID, envID, from, until)
	benchCount("projection count", `
		SELECT COUNT(*) FROM trace_runs_v2
		WHERE account_id = $1 AND env_id = $2 AND started_at >= $3 AND started_at < $4`, accountID, envID, from, until)
	benchRows("projection ended list", `
		SELECT run_id FROM trace_runs_v2
		WHERE account_id = $1 AND env_id = $2 AND ended_at >= $3 AND ended_at < $4
		ORDER BY ended_at DESC, run_id LIMIT 50`, accountID, envID, from, until)
	benchCount("projection ended count", `
		SELECT COUNT(*) FROM trace_runs_v2
		WHERE account_id = $1 AND env_id = $2 AND ended_at >= $3 AND ended_at < $4`, accountID, envID, from, until)
	benchRows("projection status list", `
		SELECT run_id FROM trace_runs_v2
		WHERE account_id = $1 AND env_id = $2 AND status = 'Completed'
			AND started_at >= $3 AND started_at < $4
		ORDER BY started_at DESC, run_id LIMIT 50`, accountID, envID, from, until)
	benchCount("projection status count", `
		SELECT COUNT(*) FROM trace_runs_v2
		WHERE account_id = $1 AND env_id = $2 AND status = 'Completed'
			AND started_at >= $3 AND started_at < $4`, accountID, envID, from, until)
	benchRows("projection point", `SELECT run_id FROM trace_runs_v2 WHERE run_id = $1`, "000000000000000000005000")
	benchRows("projection event", `SELECT run_id FROM trace_runs_v2 WHERE event_ids @> jsonb_build_array($1::text)`, eventID)

	const runtimeRunCount = 1_000
	stmt, err := db.PrepareContext(ctx, `
		INSERT INTO spans (
			span_id, trace_id, parent_span_id, name, start_time, end_time,
			attributes, links, dynamic_span_id, account_id, app_id, function_id,
			run_id, env_id, output, status, event_ids
		) VALUES (
			SUBSTRING(md5($1 || ':' || $2::text), 1, 16), md5($1), NULL, $3,
			$4::timestamptz + $2::int * interval '1 microsecond',
			$4::timestamptz + interval '100 milliseconds' + $2::int * interval '1 microsecond',
			'{}'::jsonb, '[]'::jsonb, $5, $6, $7, $8, $1, $9,
			CASE WHEN $2::int = $10::int - 1 THEN jsonb_build_object('result', $1) END,
			CASE WHEN $2::int = $10::int - 1 THEN 'Completed' ELSE 'Running' END,
			CASE WHEN $2::int = 0 THEN jsonb_build_array($11::text) ELSE '[]'::jsonb END
		)
	`)
	require.NoError(t, err)
	defer stmt.Close()

	insertRuntimeShape := func() time.Duration {
		t.Helper()
		started := time.Now()
		for i := 1; i <= runtimeRunCount; i++ {
			runID := fmt.Sprintf("%026d", i)
			for fragment := 0; fragment < spanFanout; fragment++ {
				name := "EXTEND"
				if fragment == 0 {
					name = "executor.run"
				}
				_, err := stmt.ExecContext(ctx,
					runID, fmt.Sprint(fragment), name, baseTime.Add(time.Duration(i)*time.Millisecond),
					"dyn-"+runID, accountID, appID, functionID, envID, spanFanout, eventID,
				)
				require.NoError(t, err)
			}
		}
		return time.Since(started)
	}

	_, err = db.ExecContext(ctx, `
		TRUNCATE spans, trace_runs_v2;
		ALTER TABLE spans DISABLE TRIGGER spans_project_v2_run;
	`)
	require.NoError(t, err)
	runtimeWithoutProjection := insertRuntimeShape()
	_, err = db.ExecContext(ctx, `
		TRUNCATE spans, trace_runs_v2;
		ALTER TABLE spans ENABLE TRIGGER spans_project_v2_run;
	`)
	require.NoError(t, err)
	runtimeWithProjection := insertRuntimeShape()
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trace_runs_v2`).Scan(&projected))
	require.Equal(t, runtimeRunCount, projected)

	var status, output string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT status, output->>'result' FROM trace_runs_v2 WHERE run_id = $1
	`, fmt.Sprintf("%026d", runtimeRunCount/2)).Scan(&status, &output))
	require.Equal(t, "Completed", status)
	require.Equal(t, fmt.Sprintf("%026d", runtimeRunCount/2), output)

	outOfOrderRunID := fmt.Sprintf("%026d", runtimeRunCount+1)
	_, err = stmt.ExecContext(ctx,
		outOfOrderRunID, fmt.Sprint(spanFanout-1), "EXTEND", baseTime,
		"dyn-"+outOfOrderRunID, accountID, appID, functionID, envID, spanFanout, eventID,
	)
	require.NoError(t, err)
	_, err = stmt.ExecContext(ctx,
		outOfOrderRunID, "0", "executor.run", baseTime,
		"dyn-"+outOfOrderRunID, accountID, appID, functionID, envID, spanFanout, eventID,
	)
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT status, output->>'result' FROM trace_runs_v2 WHERE run_id = $1
	`, outOfOrderRunID).Scan(&status, &output))
	require.Equal(t, "Completed", status)
	require.Equal(t, outOfOrderRunID, output)

	t.Logf("runtime-shaped ingest without projection: %s", runtimeWithoutProjection)
	t.Logf("runtime-shaped ingest with projection:    %s (%.2fx)", runtimeWithProjection, float64(runtimeWithProjection)/float64(runtimeWithoutProjection))
}
