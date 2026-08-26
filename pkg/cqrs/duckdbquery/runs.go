package duckdbquery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/run"
)

const runColumns = "account_id, env_id, app_id, function_id, run_id, queued_at, started_at, ended_at, status, output, event_ids"

// runStatusToStepStatusString maps a cqrs.TraceRun filter's RunStatus back
// to the single StepStatus string inngest.runs.status actually stores for
// it. Only covers the five values pkg/execution/dualwrite/listener.go ever
// writes (Queued/Running/Completed/Failed/Cancelled) — dual-write never
// produces Waiting/Sleeping/Invoking/Errored/TimedOut/Skipped rows for the
// runs table, so a RunStatus that only maps from those (e.g.
// RunStatusSkipped) has nothing to match and is silently dropped from the
// filter.
func runStatusToStepStatusString(s enums.RunStatus) (string, bool) {
	switch s {
	case enums.RunStatusScheduled:
		return enums.StepStatusQueued.String(), true
	case enums.RunStatusRunning:
		return enums.StepStatusRunning.String(), true
	case enums.RunStatusCompleted:
		return enums.StepStatusCompleted.String(), true
	case enums.RunStatusFailed:
		return enums.StepStatusFailed.String(), true
	case enums.RunStatusCancelled:
		return enums.StepStatusCancelled.String(), true
	default:
		return "", false
	}
}

// latestRunsCTE builds the "collapse to one row per run_id" CTE every runs
// query starts from. account_id/env_id/app_id/function_id are safe to
// filter before the collapse (they never vary across a run's lifecycle
// rows); status and time-range filtering happen after, against the
// collapsed row, and are the caller's responsibility to append.
func latestRunsCTE(filter cqrs.GetTraceRunFilter) (string, []any) {
	where := []string{"account_id = ?", "env_id = ?"}
	args := []any{filter.AccountID.String(), filter.WorkspaceID.String()}

	if len(filter.AppID) > 0 {
		placeholders := make([]string, len(filter.AppID))
		for i, id := range filter.AppID {
			placeholders[i] = "?"
			args = append(args, id.String())
		}
		where = append(where, fmt.Sprintf("app_id IN (%s)", strings.Join(placeholders, ", ")))
	}
	if len(filter.FunctionID) > 0 {
		placeholders := make([]string, len(filter.FunctionID))
		for i, id := range filter.FunctionID {
			placeholders[i] = "?"
			args = append(args, id.String())
		}
		where = append(where, fmt.Sprintf("function_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Tiebreak by COALESCE(ended_at, started_at, queued_at) DESC, not
	// inserted_at: the batcher flushes a whole batch of lifecycle rows in
	// one INSERT statement (pkg/execution/dualwrite/batch.go), and DuckDB
	// evaluates a column DEFAULT like current_timestamp once per statement,
	// not once per row — every row in the same flush gets an *identical*
	// inserted_at, making it useless as a tiebreak (verified empirically:
	// all three of a run's scheduled/started/finished rows land with the
	// same inserted_at when flushed together). COALESCE picks the furthest
	// state a row's own columns encode: a finished row's ended_at is always
	// >= a started row's started_at, which is always >= every row's
	// queued_at — so this ranks a run's rows by lifecycle progress
	// regardless of flush/insertion timing.
	query := fmt.Sprintf(
		`SELECT *, ROW_NUMBER() OVER (
			PARTITION BY run_id ORDER BY COALESCE(ended_at, started_at, queued_at) DESC
		 ) AS rn
		 FROM %s.runs WHERE %s`,
		duckdb.DuckLakeAlias, strings.Join(where, " AND "),
	)
	return query, args
}

// postCollapseFilter builds the status/time-range WHERE clause applied
// against the already-collapsed (rn = 1) row.
func postCollapseFilter(filter cqrs.GetTraceRunFilter) (string, []any) {
	where := []string{"rn = 1"}
	args := []any{}

	if len(filter.Status) > 0 {
		var statusValues []string
		for _, s := range filter.Status {
			if v, ok := runStatusToStepStatusString(s); ok {
				statusValues = append(statusValues, v)
			}
		}
		if len(statusValues) > 0 {
			placeholders := make([]string, len(statusValues))
			for i, v := range statusValues {
				placeholders[i] = "?"
				args = append(args, v)
			}
			where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
		}
	}

	tsField := strings.ToLower(filter.TimeField.String())
	where = append(where, tsField+" >= ?", tsField+" < ?")
	until := filter.Until
	if until.IsZero() {
		until = time.Now()
	}
	args = append(args, filter.From, until)

	return strings.Join(where, " AND "), args
}

// buildRunsCursorSeek decodes a request cursor (if any) into the
// keyset-pagination predicate GetTraceRuns appends to postCollapseFilter's
// WHERE clause, keyed on the same (orderCol, orderDir) pair used for
// ORDER BY — with run_id as the final tiebreak, always ascending,
// matching pkg/cqrs/manager's own newRunsQueryBuilder convention. Returns
// "", nil, nil for the first page (no cursor, or a cursor with nothing
// under this field — same as manager's own tolerant Find/Add pattern).
func buildRunsCursorSeek(cursorStr, orderCol, orderDir string) (string, []any, error) {
	if cursorStr == "" {
		return "", nil, nil
	}
	cur := &cqrs.TracePageCursor{}
	if err := cur.Decode(cursorStr); err != nil {
		return "", nil, fmt.Errorf("duckdbquery: decoding runs cursor: %w", err)
	}
	tc := cur.Find(orderCol)
	if tc == nil || cur.ID == "" {
		return "", nil, nil
	}
	cmp := ">"
	if orderDir == "DESC" {
		cmp = "<"
	}
	val := time.UnixMilli(tc.Value)
	return fmt.Sprintf("(%s %s ? OR (%s = ? AND run_id > ?))", orderCol, cmp, orderCol), []any{val, val, cur.ID}, nil
}

// encodeRunsCursor builds the per-row response cursor for keyset
// pagination, matching buildRunsCursorSeek's decode shape exactly — the
// same field name (orderCol) and the same UnixMilli encoding.
func encodeRunsCursor(run *cqrs.TraceRun, orderCol string) (string, error) {
	var fieldTime time.Time
	switch orderCol {
	case "started_at":
		fieldTime = run.StartedAt
	case "ended_at":
		fieldTime = run.EndedAt
	default:
		fieldTime = run.QueuedAt
	}
	c := cqrs.TracePageCursor{
		ID: run.RunID,
		Cursors: map[string]cqrs.TraceCursor{
			orderCol: {Field: orderCol, Value: fieldTime.UnixMilli()},
		},
	}
	return c.Encode()
}

func (m *Manager) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM %s.runs WHERE run_id = ? ORDER BY COALESCE(ended_at, started_at, queued_at) DESC LIMIT 1;",
		runColumns, duckdb.DuckLakeAlias,
	)
	rows, err := m.db.QueryContext(ctx, query, id.RunID.String())
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying trace run: %w", err)
	}
	defer rows.Close()

	named, err := scanNamedRows(rows)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: scanning trace run row: %w", err)
	}
	if len(named) == 0 {
		return nil, fmt.Errorf("run not found: %s", id.RunID)
	}
	return traceRunFromRow(named[0])
}

// GetTraceRuns' CEL handling deliberately does not push filter.CEL into the
// SQL WHERE clause — it fetches the typed-filtered candidate set (below)
// and post-filters in Go, exactly like pkg/cqrs/manager's sqlite/postgres
// path already does for output-expression filters (see
// wrapper.getTraceRunsFromTable's expHandler.HasOutputFilters() branch).
//
// TODO: implement a CEL-to-DuckDB-SQL condition handler (an
// run.ExprSQLConverter, mirroring run.EventFieldConverter's role for
// SQLite/Postgres via run.WithExpressionSQLConverter) so simple CEL
// predicates push down into the WHERE clause instead of requiring a
// full-candidate-set fetch. Deferred: the typed-filter case (status/app/
// function/time range) is the common one, and this is strictly a
// performance follow-up, not a correctness gap.
//
// Event-expression filters (filter.CEL referencing `event.*`) are a no-op
// here rather than silently wrong: matching them requires resolving each
// run's triggering event via TraceRun.TriggerIDs, which is empty for
// DuckDB-backed runs — there is nothing to fetch and match against.
func (m *Manager) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	expHandler, err := run.NewExpressionHandler(ctx, run.WithExpressionHandlerBlob(opt.Filter.CEL, "\n"))
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: parsing CEL filter: %w", err)
	}

	cte, cteArgs := latestRunsCTE(opt.Filter)
	postWhere, postArgs := postCollapseFilter(opt.Filter)

	orderCol := strings.ToLower(opt.Filter.TimeField.String())
	orderDir := "DESC"
	for _, o := range opt.Order {
		if o.Field == opt.Filter.TimeField {
			if o.Direction == enums.TraceRunOrderAsc {
				orderDir = "ASC"
			}
			break
		}
	}

	seekWhere, seekArgs, err := buildRunsCursorSeek(opt.Cursor, orderCol, orderDir)
	if err != nil {
		return nil, err
	}
	if seekWhere != "" {
		postWhere += " AND " + seekWhere
		postArgs = append(postArgs, seekArgs...)
	}

	query := fmt.Sprintf(
		"WITH latest AS (%s) SELECT %s FROM latest WHERE %s ORDER BY %s %s, run_id ASC",
		cte, runColumns, postWhere, orderCol, orderDir,
	)
	args := append(cteArgs, postArgs...)
	// A CEL filter narrows the fetched set further in Go below, so the SQL
	// LIMIT can't be applied here without risking fewer than opt.Items
	// matches — skip it in that case, matching
	// wrapper.getTraceRunsFromTable's own "!expHandler.HasOutputFilters()"
	// guard on its LIMIT.
	if opt.Items > 0 && !expHandler.HasFilters() {
		query += " LIMIT ?"
		args = append(args, opt.Items+1)
	}
	query += ";"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying trace runs: %w", err)
	}
	defer rows.Close()

	named, err := scanNamedRows(rows)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: scanning trace run rows: %w", err)
	}

	out := make([]*cqrs.TraceRun, 0, len(named))
	for _, row := range named {
		trun, err := traceRunFromRow(row)
		if err != nil {
			return nil, err
		}

		if expHandler.HasOutputFilters() {
			ok, err := expHandler.MatchOutputExpressions(ctx, trun.Output)
			if err != nil {
				return nil, fmt.Errorf("duckdbquery: matching output CEL filter: %w", err)
			}
			if !ok {
				continue
			}
		}

		trun.Cursor, err = encodeRunsCursor(trun, orderCol)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: encoding run cursor: %w", err)
		}
		out = append(out, trun)
		if opt.Items > 0 && uint(len(out)) > opt.Items {
			break
		}
	}
	return out, nil
}

func (m *Manager) GetTraceRunsCount(ctx context.Context, opt cqrs.GetTraceRunOpt) (int, error) {
	cte, cteArgs := latestRunsCTE(opt.Filter)
	postWhere, postArgs := postCollapseFilter(opt.Filter)

	query := fmt.Sprintf("WITH latest AS (%s) SELECT COUNT(*) AS c FROM latest WHERE %s;", cte, postWhere)
	args := append(cteArgs, postArgs...)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: querying trace run count: %w", err)
	}
	defer rows.Close()

	named, err := scanNamedRows(rows)
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: scanning trace run count: %w", err)
	}
	if len(named) == 0 {
		return 0, nil
	}
	n, err := duckdb.AsInt64(named[0]["c"])
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func traceRunFromRow(row map[string]any) (*cqrs.TraceRun, error) {
	accountID, err := uuidField(row, "account_id")
	if err != nil {
		return nil, err
	}
	envID, err := uuidField(row, "env_id")
	if err != nil {
		return nil, err
	}
	appID, err := uuidField(row, "app_id")
	if err != nil {
		return nil, err
	}
	functionID, err := uuidField(row, "function_id")
	if err != nil {
		return nil, err
	}
	runID, err := stringField(row, "run_id")
	if err != nil {
		return nil, err
	}
	queuedAt, err := timeField(row, "queued_at")
	if err != nil {
		return nil, err
	}
	startedAt, err := nullableTimeField(row, "started_at")
	if err != nil {
		return nil, err
	}
	endedAt, err := nullableTimeField(row, "ended_at")
	if err != nil {
		return nil, err
	}
	statusStr, err := stringField(row, "status")
	if err != nil {
		return nil, err
	}
	stepStatus, err := enums.StepStatusString(statusStr)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: parsing run status %q: %w", statusStr, err)
	}
	output, err := jsonField(row, "output")
	if err != nil {
		return nil, err
	}

	// event_ids is a real VARCHAR[] (NULL for cron-only runs, which have no
	// triggering event at all) — the driver hands it back as []any
	// regardless of transport (see pkg/db/duckdb/quack_protocol.go's LIST
	// decoding and the stdio transport's own JSON-lines auto-decoding).
	var triggerIDs []string
	if raw, ok := row["event_ids"]; ok && raw != nil {
		items, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("duckdbquery: event_ids has unexpected type %T", raw)
		}
		triggerIDs = make([]string, len(items))
		for i, item := range items {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("duckdbquery: event_ids element %d has unexpected type %T", i, item)
			}
			triggerIDs[i] = s
		}
	}

	return &cqrs.TraceRun{
		AccountID:   accountID,
		WorkspaceID: envID,
		AppID:       appID,
		FunctionID:  functionID,
		RunID:       runID,
		QueuedAt:    queuedAt,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		Output:      output,
		Status:      enums.StepStatusToRunStatus(stepStatus),
		TriggerIDs:  triggerIDs,
	}, nil
}
