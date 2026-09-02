package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
)

// DefaultTemplates returns a small, self-contained set of synthetic
// templates used when a source database has no real data to sample —
// GenerateRuns always has something usable to work from.
func DefaultTemplates() Templates {
	return Templates{
		Tenants: []Tenant{
			{
				AccountID:  uuid.New(),
				EnvID:      uuid.New(),
				AppID:      uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		Statuses: []string{"Queued", "Running", "Completed", "Completed", "Failed", "Cancelled"},
		Inputs:   []string{`{"event":{"name":"app/seeded.event","data":{}}}`},
		Outputs:  []string{`{"data":"ok"}`},
		Traces: []TraceTemplate{
			{
				TraceID: "seeded-trace",
				Spans: []SpanTemplate{
					{
						SpanID:      "seeded-root",
						Name:        "seeded-function",
						Attributes:  `{"sys.step.name":"seeded-function"}`,
						Output:      `{"data":"ok"}`,
						Input:       `{"event":{"name":"app/seeded.event","data":{}}}`,
						StartOffset: 100 * time.Millisecond,
						EndOffset:   600 * time.Millisecond,
					},
					{
						SpanID:       "seeded-step-1",
						ParentSpanID: strPtr("seeded-root"),
						Name:         "seeded-function-step-1",
						Attributes:   `{"sys.step.name":"seeded-function-step-1"}`,
						Output:       `{"data":"ok"}`,
						Input:        `{}`,
						StartOffset:  100 * time.Millisecond,
						EndOffset:    600 * time.Millisecond,
					},
				},
				StartedOffset: durationPtr(100 * time.Millisecond),
				EndedOffset:   durationPtr(600 * time.Millisecond),
			},
		},
		EventProfiles: [][]EventTemplate{
			{{Name: "app/seeded.event", Data: `{}`, Offset: -200 * time.Millisecond}},
		},
		MetadataProfiles: []MetadataProfile{
			{
				Items: []MetadataTemplateItem{
					{SpanID: "seeded-root", Scope: "run", Kind: "inngest.experiment", IsUser: false, Values: `{"seeded":true}`},
					{SpanID: "seeded-step-1", Scope: "step", StepID: strPtr("seeded-step-1"), StepIndex: intPtr(0), StepAttempt: intPtr(1), Kind: "user.custom", IsUser: true, Values: `{"seeded":true}`},
				},
			},
		},
	}
}

// BuildTemplates samples up to limit distinct runs from db (see
// SampleTemplates) and falls back to DefaultTemplates when the source has
// no runs at all, so callers never need to special-case an empty database.
// sampled reports which path was taken, so callers can log it.
func BuildTemplates(ctx context.Context, db *sql.DB, limit int) (tmpl Templates, sampled bool, err error) {
	tmpl, err = SampleTemplates(ctx, db, limit)
	if err != nil {
		return Templates{}, false, err
	}
	if len(tmpl.Tenants) == 0 {
		return DefaultTemplates(), false, nil
	}
	return tmpl, true, nil
}

// SampleTemplates samples up to limit distinct runs from each of
// inngest.runs/run_trace_spans/events/run_metadata and returns the
// distinct tenant tuples, statuses, and whole per-run shapes observed — a
// template GenerateRuns samples from to produce data that looks like it.
// Returns a zero-value Templates (no error) when the source tables are
// empty. Each of sampleRuns/sampleTraces/sampleEventProfiles/
// sampleMetadata picks its own bounded set of run_ids independently (see
// their own doc comments), so the four pools aren't guaranteed to be the
// same runs -- consistent with Templates' existing "independently sampled
// pools, mixed at generation time" design.
func SampleTemplates(ctx context.Context, db *sql.DB, limit int) (Templates, error) {
	var tmpl Templates

	tenants, statuses, inputs, outputs, err := sampleRuns(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.Tenants, tmpl.Statuses, tmpl.Inputs, tmpl.Outputs = tenants, statuses, inputs, outputs

	traces, err := sampleTraces(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.Traces = traces

	eventProfiles, err := sampleEventProfiles(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.EventProfiles = eventProfiles

	profiles, err := sampleMetadata(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.MetadataProfiles = profiles

	return tmpl, nil
}

// sampleRuns picks up to limit distinct run_ids first (a cheap scan, no
// sort or window function) and only then computes each one's latest
// lifecycle row -- rather than ranking every row in the whole table by
// run_id before limiting, which forces a full-table sort/window pass no
// matter how small limit is. inngest.runs is append-only (a run gets a new
// row at each lifecycle transition), so without this a source with a long
// history pays for ranking all of it on every seed run just to keep a
// handful of rows.
func sampleRuns(ctx context.Context, db *sql.DB, limit int) (tenants []Tenant, statuses, inputs, outputs []string, err error) {
	query := fmt.Sprintf(`
	WITH sampled_run_ids AS (
		SELECT DISTINCT run_id FROM %s.runs LIMIT ?
	)
	SELECT account_id, env_id, app_id, function_id, status, inputs, output
	FROM %s.runs
	WHERE run_id IN (SELECT run_id FROM sampled_run_ids)
	QUALIFY row_number() OVER (PARTITION BY run_id ORDER BY ended_at DESC NULLS LAST, started_at DESC NULLS LAST) = 1
	;`, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("duckdbseed: sampling runs: %w", err)
	}
	defer rows.Close()

	seenTenants := map[Tenant]bool{}
	for rows.Next() {
		var (
			rawAccountID, rawEnvID, rawAppID, rawFunctionID any
			status                                          any
			rawInputs, rawOutput                            any
		)
		if err := rows.Scan(&rawAccountID, &rawEnvID, &rawAppID, &rawFunctionID, &status, &rawInputs, &rawOutput); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("duckdbseed: scanning sampled run: %w", err)
		}

		tenant, err := tenantFromValues(rawAccountID, rawEnvID, rawAppID, rawFunctionID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !seenTenants[tenant] {
			seenTenants[tenant] = true
			tenants = append(tenants, tenant)
		}

		statuses = append(statuses, fmt.Sprint(status))
		if v := jsonText(rawInputs); v != "" {
			inputs = append(inputs, v)
		}
		if v := jsonText(rawOutput); v != "" {
			outputs = append(outputs, v)
		}
	}
	return tenants, statuses, inputs, outputs, rows.Err()
}

// sampleTraces reads every inngest.run_trace_spans row belonging to up to
// limit distinct sampled runs, each joined back onto its own run's
// trace_id/queued_at/started_at/ended_at, and groups them into whole
// per-run flat span sets (see TraceTemplate). Every offset (spans'
// StartOffset/EndOffset, the trace's own StartedOffset/EndedOffset) is
// relative to that one run's own queued_at, so generateRun can replay the
// whole thing by adding a single randomly placed queued_at to each -- one
// random offset per run, not independent jitter per timestamp.
//
// This does no join or window function to work out which span is the
// root or how the tree nests: SpanTemplate carries each span's real
// SpanID/ParentSpanID text completely unchanged (see its doc comment for
// why that's enough to replay any real nesting -- including a userland/
// extended-trace span nested under another userland span -- without this
// package ever resolving the tree itself), so sampling is a flat,
// single-pass read.
//
// limit bounds the number of distinct RUNS sampled, not rows: the query
// picks limit run_ids up front (a cheap scan, no join or window function)
// and scopes the rest of the read to just those, rather than reading
// every span the source has ever written on every seed run regardless of
// how small limit is.
func sampleTraces(ctx context.Context, db *sql.DB, limit int) ([]TraceTemplate, error) {
	query := fmt.Sprintf(`
	WITH sampled_runs AS (
		SELECT DISTINCT run_id FROM %s.run_trace_spans LIMIT ?
	),
	run_timing AS (
		SELECT run_id, queued_at, started_at, ended_at
		FROM %s.runs
		WHERE run_id IN (SELECT run_id FROM sampled_runs)
		QUALIFY row_number() OVER (PARTITION BY run_id ORDER BY ended_at DESC NULLS LAST, started_at DESC NULLS LAST) = 1
	)
	SELECT s.run_id, rt.queued_at, rt.started_at, rt.ended_at,
	       s.trace_id, s.span_id, s.parent_span_id,
	       s.name, s.attributes, s.output, s.input, s.start_time, s.end_time
	FROM %s.run_trace_spans s
	JOIN run_timing rt ON rt.run_id = s.run_id;`, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("duckdbseed: sampling spans: %w", err)
	}
	defer rows.Close()

	type building struct {
		queuedAt      time.Time
		startedOffset *time.Duration
		endedOffset   *time.Duration
		traceID       string
		spans         []SpanTemplate
	}
	byRun := map[string]*building{}
	var order []string

	for rows.Next() {
		var rawRunID, rawQueuedAt, rawStartedAt, rawEndedAt any
		var traceID, spanID string
		var rawParentSpanID any
		var name, rawAttributes, rawOutput, rawInput, rawStart, rawEnd any
		if err := rows.Scan(&rawRunID, &rawQueuedAt, &rawStartedAt, &rawEndedAt, &traceID, &spanID, &rawParentSpanID, &name, &rawAttributes, &rawOutput, &rawInput, &rawStart, &rawEnd); err != nil {
			return nil, fmt.Errorf("duckdbseed: scanning sampled span: %w", err)
		}

		runID := fmt.Sprint(rawRunID)
		b, ok := byRun[runID]
		if !ok {
			queuedAt, err := duckdb.AsTimestamp(rawQueuedAt)
			if err != nil {
				return nil, fmt.Errorf("duckdbseed: parsing sampled run queued_at: %w", err)
			}
			b = &building{queuedAt: queuedAt, traceID: traceID}
			if rawStartedAt != nil {
				startedAt, err := duckdb.AsTimestamp(rawStartedAt)
				if err != nil {
					return nil, fmt.Errorf("duckdbseed: parsing sampled run started_at: %w", err)
				}
				b.startedOffset = durationPtr(startedAt.Sub(queuedAt))
			}
			if rawEndedAt != nil {
				endedAt, err := duckdb.AsTimestamp(rawEndedAt)
				if err != nil {
					return nil, fmt.Errorf("duckdbseed: parsing sampled run ended_at: %w", err)
				}
				b.endedOffset = durationPtr(endedAt.Sub(queuedAt))
			}
			byRun[runID] = b
			order = append(order, runID)
		}

		start, err := duckdb.AsTimestamp(rawStart)
		if err != nil {
			return nil, fmt.Errorf("duckdbseed: parsing sampled span start_time: %w", err)
		}
		end, err := duckdb.AsTimestamp(rawEnd)
		if err != nil {
			return nil, fmt.Errorf("duckdbseed: parsing sampled span end_time: %w", err)
		}

		var parentSpanID *string
		if s, ok := rawParentSpanID.(string); ok {
			parentSpanID = &s
		}

		b.spans = append(b.spans, SpanTemplate{
			SpanID:       spanID,
			ParentSpanID: parentSpanID,
			Name:         fmt.Sprint(name),
			Attributes:   jsonText(rawAttributes),
			Output:       jsonText(rawOutput),
			Input:        jsonText(rawInput),
			StartOffset:  start.Sub(b.queuedAt),
			EndOffset:    end.Sub(b.queuedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbseed: reading sampled spans: %w", err)
	}

	traces := make([]TraceTemplate, 0, len(order))
	for _, runID := range order {
		b := byRun[runID]
		traces = append(traces, TraceTemplate{
			TraceID:       b.traceID,
			Spans:         b.spans,
			StartedOffset: b.startedOffset,
			EndedOffset:   b.endedOffset,
		})
	}
	return traces, nil
}

// sampleEventProfiles reads up to limit distinct sampled runs' own
// event_ids, joined back onto inngest.events, and groups them into whole
// per-run sets of triggering events (see EventTemplate) — usually one,
// but a batch-triggered run can have several received close together —
// so a run's trigger events can be replayed intact rather than each
// picked independently. Each item's Offset is its received_at relative to
// that same run's own queued_at (see sampleTraces' doc comment for why).
//
// limit bounds distinct RUNS, not rows: run_ids are picked up front (a
// cheap scan) before the QUALIFY window function and event join run, so a
// source with a long run history doesn't rank every lifecycle row it's
// ever had just to keep a handful.
func sampleEventProfiles(ctx context.Context, db *sql.DB, limit int) ([][]EventTemplate, error) {
	query := fmt.Sprintf(`
	WITH sampled_run_ids AS (
		SELECT DISTINCT run_id FROM %s.runs LIMIT ?
	),
	latest_runs AS (
		SELECT run_id, queued_at, event_ids
		FROM %s.runs
		WHERE run_id IN (SELECT run_id FROM sampled_run_ids)
		QUALIFY row_number() OVER (PARTITION BY run_id ORDER BY ended_at DESC NULLS LAST, started_at DESC NULLS LAST) = 1
	),
	run_events AS (
		SELECT run_id, queued_at, UNNEST(event_ids) AS event_internal_id FROM latest_runs
	)
	SELECT re.run_id, re.queued_at, e.event_name, e.event_data, e.received_at
	FROM run_events re
	JOIN %s.events e ON e.internal_id = re.event_internal_id;`, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("duckdbseed: sampling events: %w", err)
	}
	defer rows.Close()

	byRun := map[string][]EventTemplate{}
	var order []string
	for rows.Next() {
		var rawRunID, rawQueuedAt, eventName, rawEventData, rawReceivedAt any
		if err := rows.Scan(&rawRunID, &rawQueuedAt, &eventName, &rawEventData, &rawReceivedAt); err != nil {
			return nil, fmt.Errorf("duckdbseed: scanning sampled event: %w", err)
		}

		queuedAt, err := duckdb.AsTimestamp(rawQueuedAt)
		if err != nil {
			return nil, fmt.Errorf("duckdbseed: parsing sampled run queued_at: %w", err)
		}
		receivedAt, err := duckdb.AsTimestamp(rawReceivedAt)
		if err != nil {
			return nil, fmt.Errorf("duckdbseed: parsing sampled event received_at: %w", err)
		}

		runID := fmt.Sprint(rawRunID)
		if _, ok := byRun[runID]; !ok {
			order = append(order, runID)
		}
		byRun[runID] = append(byRun[runID], EventTemplate{
			Name:   fmt.Sprint(eventName),
			Data:   jsonText(rawEventData),
			Offset: receivedAt.Sub(queuedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbseed: reading sampled events: %w", err)
	}

	profiles := make([][]EventTemplate, len(order))
	for i, runID := range order {
		profiles[i] = byRun[runID]
	}
	return profiles, nil
}

// durationPtr returns a pointer to d, for TraceTemplate's optional
// StartedOffset/EndedOffset fields.
func durationPtr(d time.Duration) *time.Duration {
	return &d
}

// sampleMetadata reads every inngest.run_metadata row belonging to up to
// limit distinct sampled runs, joined only onto its own run's queued_at,
// and groups them into whole per-run metadata profiles (see
// MetadataProfile) -- each row's SpanID/Scope/StepID/StepIndex/
// StepAttempt reused completely verbatim, the same way SpanTemplate
// reuses SpanID/ParentSpanID (see its own doc comment), so no join against
// run_trace_spans is needed here at all: generateMetadata attaches an
// item to whichever replayed span happens to share its exact SpanID.
// Offset is each row's created_at relative to that run's own queued_at.
//
// limit bounds the number of distinct RUNS sampled, not rows: run_ids are
// picked up front (a cheap scan) before the timing join runs, so a source
// with a long history doesn't pay for more than a handful of runs' worth
// of metadata regardless of how much it has accumulated overall.
func sampleMetadata(ctx context.Context, db *sql.DB, limit int) ([]MetadataProfile, error) {
	query := fmt.Sprintf(`
	WITH sampled_runs AS (
		SELECT DISTINCT run_id FROM %s.run_metadata LIMIT ?
	),
	run_timing AS (
		SELECT run_id, queued_at
		FROM %s.runs
		WHERE run_id IN (SELECT run_id FROM sampled_runs)
		QUALIFY row_number() OVER (PARTITION BY run_id ORDER BY ended_at DESC NULLS LAST, started_at DESC NULLS LAST) = 1
	)
	SELECT m.run_id, rt.queued_at, m.span_id, m.scope, m.step_id, m.step_index, m.step_attempt, m.kind, m.is_user, m.values, m.created_at
	FROM %s.run_metadata m
	JOIN run_timing rt ON rt.run_id = m.run_id;`, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("duckdbseed: sampling metadata: %w", err)
	}
	defer rows.Close()

	byRun := map[string]*MetadataProfile{}
	var order []string
	for rows.Next() {
		var rawRunID, rawQueuedAt any
		var spanID, scope string
		var rawStepID, rawStepIndex, rawStepAttempt any
		var kind string
		var rawIsUser, rawValues, rawCreatedAt any
		if err := rows.Scan(&rawRunID, &rawQueuedAt, &spanID, &scope, &rawStepID, &rawStepIndex, &rawStepAttempt, &kind, &rawIsUser, &rawValues, &rawCreatedAt); err != nil {
			return nil, fmt.Errorf("duckdbseed: scanning sampled metadata: %w", err)
		}

		queuedAt, err := duckdb.AsTimestamp(rawQueuedAt)
		if err != nil {
			return nil, fmt.Errorf("duckdbseed: parsing sampled run queued_at: %w", err)
		}
		createdAt, err := duckdb.AsTimestamp(rawCreatedAt)
		if err != nil {
			return nil, fmt.Errorf("duckdbseed: parsing sampled metadata created_at: %w", err)
		}

		runID := fmt.Sprint(rawRunID)
		profile, ok := byRun[runID]
		if !ok {
			profile = &MetadataProfile{}
			byRun[runID] = profile
			order = append(order, runID)
		}

		var stepID *string
		if s, ok := rawStepID.(string); ok {
			stepID = &s
		}
		var stepIndex *int
		if rawStepIndex != nil {
			v := int(int64Value(rawStepIndex))
			stepIndex = &v
		}
		var stepAttempt *int
		if rawStepAttempt != nil {
			v := int(int64Value(rawStepAttempt))
			stepAttempt = &v
		}

		profile.Items = append(profile.Items, MetadataTemplateItem{
			SpanID:      spanID,
			Scope:       scope,
			StepID:      stepID,
			StepIndex:   stepIndex,
			StepAttempt: stepAttempt,
			Kind:        kind,
			IsUser:      boolValue(rawIsUser),
			Values:      jsonText(rawValues),
			Offset:      createdAt.Sub(queuedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbseed: reading sampled metadata: %w", err)
	}

	profiles := make([]MetadataProfile, len(order))
	for i, runID := range order {
		profiles[i] = *byRun[runID]
	}
	return profiles, nil
}

// strPtr returns a pointer to s, for SpanTemplate/MetadataTemplateItem's
// optional string fields.
func strPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to i, for MetadataTemplateItem's optional int
// fields.
func intPtr(i int) *int {
	return &i
}

// boolValue type-asserts a scanned BOOLEAN column's value, defaulting to
// false for a NULL or unexpected type rather than erroring — used only for
// is_root (computed by this file's own query, never NULL) and is_user
// (NOT NULL in the schema), so this is a defensive fallback, not an
// expected path.
func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

// int64Value type-asserts a scanned integer column's value, tolerating
// both int64 (the quack transport's shape for BIGINT) and float64 (the
// jsonlines transport decodes numbers via encoding/json) — mirroring
// pkg/db/duckdb.AsInt64's same dual handling for the same reason.
func int64Value(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func tenantFromValues(rawAccountID, rawEnvID, rawAppID, rawFunctionID any) (Tenant, error) {
	accountID, err := uuidValue(rawAccountID, "account_id")
	if err != nil {
		return Tenant{}, err
	}
	envID, err := uuidValue(rawEnvID, "env_id")
	if err != nil {
		return Tenant{}, err
	}
	appID, err := uuidValue(rawAppID, "app_id")
	if err != nil {
		return Tenant{}, err
	}
	functionID, err := uuidValue(rawFunctionID, "function_id")
	if err != nil {
		return Tenant{}, err
	}
	return Tenant{AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: functionID}, nil
}

// uuidValue parses a UUID column's scanned value — pkg/db/duckdb's driver
// (both the jsonlines and quack transports) renders a UUID column as its
// canonical string form, not raw bytes.
func uuidValue(v any, col string) (uuid.UUID, error) {
	s, ok := v.(string)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("duckdbseed: expected string for column %q, got %T", col, v)
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("duckdbseed: parsing column %q as UUID: %w", col, err)
	}
	return id, nil
}

// jsonText re-marshals a JSON column's already-decoded Go value (the
// duckdb driver auto-decodes JSON-typed columns into
// map[string]any/[]any/etc.) back into text, so it can be reused verbatim
// as a template value. Returns "" for SQL NULL or a marshal failure.
func jsonText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
