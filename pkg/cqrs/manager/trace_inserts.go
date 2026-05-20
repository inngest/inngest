package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/doug-martin/goqu/v9"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/inngest/inngest/pkg/cqrs"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/oklog/ulid/v2"
)

// traceBulkChunkSize bounds the number of rows per INSERT so we stay under
// Postgres' 65535 bound parameters per query (19 cols * 500 rows = 9500 args).
const traceBulkChunkSize = 500

// traceCols and traceRunCols list columns in the same order as their
// generated single-row INSERT statements (postgres) so that bulk INSERTs
// produce equivalent SQL on either dialect.
var (
	traceCols = []any{
		"timestamp", "timestamp_unix_ms", "trace_id", "span_id", "parent_span_id",
		"trace_state", "span_name", "span_kind", "service_name", "resource_attributes",
		"scope_name", "scope_version", "span_attributes", "duration", "status_code",
		"status_message", "events", "links", "run_id",
	}

	traceRunCols = []any{
		"account_id", "workspace_id", "app_id", "function_id", "trace_id",
		"run_id", "queued_at", "started_at", "ended_at", "status",
		"source_id", "trigger_ids", "output", "batch_id", "is_debounce",
		"cron_schedule", "has_ai",
	}
)

func (w wrapper) InsertSpan(ctx context.Context, span *cqrs.Span) error {
	params := buildInsertTraceParams(span)
	return w.q.InsertTrace(ctx, *params)
}

func (w wrapper) InsertTraceRun(ctx context.Context, run *cqrs.TraceRun) error {
	params, err := buildInsertTraceRunParams(run)
	if err != nil {
		return err
	}
	return w.q.InsertTraceRun(ctx, *params)
}

func (w wrapper) InsertSpans(ctx context.Context, spans []*cqrs.Span) error {
	if len(spans) == 0 {
		return nil
	}
	dialect := w.dialect()
	rows := make([][]any, len(spans))
	for i, s := range spans {
		rows[i] = traceParamsToRow(buildInsertTraceParams(s), dialect)
	}
	return w.bulkInsert(ctx, "traces", traceCols, rows, nil)
}

func (w wrapper) InsertTraceRuns(ctx context.Context, runs []*cqrs.TraceRun) error {
	if len(runs) == 0 {
		return nil
	}
	dialect := w.dialect()
	rows := make([][]any, 0, len(runs))
	// Per-row build errors (e.g. malformed RunID ULID) skip that row but
	// must not abort the batch; the legacy single-row path logged the error
	// and continued, and the goal of switching to bulk inserts is to *avoid*
	// dropping otherwise-valid telemetry. Collected build errors are joined
	// and returned alongside the bulk insert error so callers can surface
	// them.
	var buildErrs []error
	for _, r := range runs {
		params, err := buildInsertTraceRunParams(r)
		if err != nil {
			buildErrs = append(buildErrs, fmt.Errorf("run_id=%q: %w", r.RunID, err))
			continue
		}
		rows = append(rows, traceRunParamsToRow(params, dialect))
	}
	var insertErr error
	if len(rows) > 0 {
		insertErr = w.bulkInsert(ctx, "trace_runs", traceRunCols, rows, w.traceRunUpsert())
	}
	return errors.Join(append(buildErrs, insertErr)...)
}

func buildInsertTraceParams(span *cqrs.Span) *dbpkg.InsertTraceParams {
	params := &dbpkg.InsertTraceParams{
		Timestamp:       span.Timestamp,
		TimestampUnixMs: span.Timestamp.UnixMilli(),
		TraceID:         span.TraceID,
		SpanID:          span.SpanID,
		SpanName:        span.SpanName,
		SpanKind:        span.SpanKind,
		ServiceName:     span.ServiceName,
		ScopeName:       span.ScopeName,
		ScopeVersion:    span.ScopeVersion,
		Duration:        int64(span.Duration / time.Millisecond),
		StatusCode:      span.StatusCode,
	}
	if span.RunID != nil {
		params.RunID = *span.RunID
	}
	if span.ParentSpanID != nil {
		params.ParentSpanID = sql.NullString{String: *span.ParentSpanID, Valid: true}
	}
	if span.TraceState != nil {
		params.TraceState = sql.NullString{String: *span.TraceState, Valid: true}
	}
	if byt, err := json.Marshal(span.ResourceAttributes); err == nil {
		params.ResourceAttributes = byt
	}
	if byt, err := json.Marshal(span.SpanAttributes); err == nil {
		params.SpanAttributes = byt
	}
	if byt, err := json.Marshal(span.Events); err == nil {
		params.Events = byt
	}
	if byt, err := json.Marshal(span.Links); err == nil {
		params.Links = byt
	}
	if span.StatusMessage != nil {
		params.StatusMessage = sql.NullString{String: *span.StatusMessage, Valid: true}
	}
	return params
}

func buildInsertTraceRunParams(run *cqrs.TraceRun) (*dbpkg.InsertTraceRunParams, error) {
	runid, err := ulid.Parse(run.RunID)
	if err != nil {
		return nil, fmt.Errorf("error parsing runID as ULID: %w", err)
	}

	params := &dbpkg.InsertTraceRunParams{
		AccountID:   run.AccountID,
		WorkspaceID: run.WorkspaceID,
		AppID:       run.AppID,
		FunctionID:  run.FunctionID,
		TraceID:     []byte(run.TraceID),
		SourceID:    run.SourceID,
		RunID:       runid,
		QueuedAt:    run.QueuedAt.UnixMilli(),
		StartedAt:   run.StartedAt.UnixMilli(),
		EndedAt:     run.EndedAt.UnixMilli(),
		Status:      run.Status.ToCode(),
		TriggerIds:  []byte{},
		Output:      run.Output,
		IsDebounce:  run.IsDebounce,
		HasAi:       run.HasAI,
	}
	if run.BatchID != nil {
		params.BatchID = *run.BatchID
	}
	if run.CronSchedule != nil {
		params.CronSchedule = sql.NullString{String: *run.CronSchedule, Valid: true}
	}
	if len(run.TriggerIDs) > 0 {
		params.TriggerIds = []byte(strings.Join(run.TriggerIDs, ","))
	}
	return params, nil
}

// traceParamsToRow / traceRunParamsToRow flatten the per-dialect *Params
// structs into row slices for goqu bulk inserts. run_id needs an explicit
// dialect conversion because the postgres column is CHAR(26): pgQuerier
// passes runID.String() in the single-row path. batch_id is bytea on
// postgres and the ulid binary blob on sqlite, but ulid.ULID's driver.Valuer
// implementation returns MarshalBinary() ([]byte) so it round-trips
// correctly to bytea without an explicit conversion (goqu prepared-mode
// calls Value() before binding).
func traceParamsToRow(p *dbpkg.InsertTraceParams, dialect string) []any {
	return []any{
		p.Timestamp, p.TimestampUnixMs, p.TraceID, p.SpanID, p.ParentSpanID,
		p.TraceState, p.SpanName, p.SpanKind, p.ServiceName, p.ResourceAttributes,
		p.ScopeName, p.ScopeVersion, p.SpanAttributes, p.Duration, p.StatusCode,
		p.StatusMessage, p.Events, p.Links, runIDForDialect(p.RunID, dialect),
	}
}

func traceRunParamsToRow(p *dbpkg.InsertTraceRunParams, dialect string) []any {
	return []any{
		p.AccountID, p.WorkspaceID, p.AppID, p.FunctionID, p.TraceID,
		runIDForDialect(p.RunID, dialect), p.QueuedAt, p.StartedAt, p.EndedAt, p.Status,
		p.SourceID, p.TriggerIds, p.Output, p.BatchID, p.IsDebounce,
		p.CronSchedule, p.HasAi,
	}
}

func runIDForDialect(id ulid.ULID, dialect string) any {
	if dialect == "postgres" {
		return id.String()
	}
	return id
}

// traceRunUpsert returns an ON CONFLICT (run_id) DO UPDATE clause matching
// the existing single-row InsertTraceRun query for both postgres and sqlite.
// has_ai uses a CASE expression so a later upsert never clears an earlier
// HasAI=true; the boolean literal differs per dialect.
func (w wrapper) traceRunUpsert() sqexp.ConflictExpression {
	rec := sq.Record{
		"account_id":    sq.L("EXCLUDED.account_id"),
		"workspace_id":  sq.L("EXCLUDED.workspace_id"),
		"app_id":        sq.L("EXCLUDED.app_id"),
		"function_id":   sq.L("EXCLUDED.function_id"),
		"trace_id":      sq.L("EXCLUDED.trace_id"),
		"queued_at":     sq.L("EXCLUDED.queued_at"),
		"started_at":    sq.L("EXCLUDED.started_at"),
		"ended_at":      sq.L("EXCLUDED.ended_at"),
		"status":        sq.L("EXCLUDED.status"),
		"source_id":     sq.L("EXCLUDED.source_id"),
		"trigger_ids":   sq.L("EXCLUDED.trigger_ids"),
		"output":        sq.L("EXCLUDED.output"),
		"batch_id":      sq.L("EXCLUDED.batch_id"),
		"is_debounce":   sq.L("EXCLUDED.is_debounce"),
		"cron_schedule": sq.L("EXCLUDED.cron_schedule"),
	}
	switch w.dialect() {
	case "postgres":
		rec["has_ai"] = sq.L("CASE WHEN trace_runs.has_ai = TRUE THEN TRUE ELSE EXCLUDED.has_ai END")
	default:
		rec["has_ai"] = sq.L("CASE WHEN trace_runs.has_ai = 1 THEN 1 ELSE EXCLUDED.has_ai END")
	}
	return sq.DoUpdate("run_id", rec)
}

func (w wrapper) bulkInsert(
	ctx context.Context,
	table string,
	cols []any,
	rows [][]any,
	onConflict sqexp.ConflictExpression,
) error {
	// Continue past chunk-level failures to preserve the legacy
	// log-and-continue behavior of the per-row InsertSpan/InsertTraceRun
	// loops this function replaces. A single bad chunk (e.g. a unique
	// constraint violation on one row, or a transient DB error) should
	// not drop telemetry from subsequent chunks. All chunk errors are
	// collected and joined so the caller still sees them.
	var chunkErrs []error
	for i := 0; i < len(rows); i += traceBulkChunkSize {
		end := i + traceBulkChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		ds := sq.Dialect(w.dialect()).
			Insert(table).
			Cols(cols...).
			Vals(rows[i:end]...)
		if onConflict != nil {
			ds = ds.OnConflict(onConflict)
		}
		sqlStr, args, err := ds.ToSQL()
		if err != nil {
			chunkErrs = append(chunkErrs, fmt.Errorf("error building bulk %s insert (chunk %d-%d): %w", table, i, end, err))
			continue
		}
		if _, err := w.adapter.ExecContext(ctx, sqlStr, args...); err != nil {
			chunkErrs = append(chunkErrs, fmt.Errorf("error executing bulk %s insert (chunk %d-%d): %w", table, i, end, err))
			continue
		}
	}
	return errors.Join(chunkErrs...)
}
