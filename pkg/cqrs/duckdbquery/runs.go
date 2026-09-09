package duckdbquery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/run"
	"github.com/oklog/ulid/v2"
)

const runColumns = "account_id, env_id, app_id, function_id, run_id, queued_at, started_at, ended_at, status, output, event_ids, is_deferred"

// runStatusToStepStatusString maps a cqrs.TraceRun filter's RunStatus back
// to the StepStatus string inngest.runs.status actually stores. Only
// covers the five values dual-write ever writes — a RunStatus that maps
// only from an unwritten status has nothing to match and is silently
// dropped from the filter.
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

// latestRunsWhere builds the pre-collapse WHERE clause every runs query
// starts from. account_id/env_id/app_id/function_id are safe to filter here
// (they never vary across a run's lifecycle rows); status and time-range
// filtering happen post-collapse, via runsQualify, since those columns do
// vary across a run's lifecycle rows.
func latestRunsWhere(filter cqrs.GetTraceRunFilter) (string, []any) {
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
	if len(filter.EventID) > 0 {
		// event_ids never varies across a run's lifecycle rows, so this is
		// safe pre-collapse. list_has_any matches any of the given IDs.
		ids := make([]string, len(filter.EventID))
		for i, id := range filter.EventID {
			ids[i] = id.String()
		}
		where = append(where, "list_has_any(event_ids, ?)")
		args = append(args, ids)
	}

	return strings.Join(where, " AND "), args
}

// resolveAppAndFunctionFilters resolves filter.AppName/FunctionSlug into
// concrete AppID/FunctionID values via the embedded primary manager, since
// inngest.runs only stores IDs, never names/slugs. Returns noMatch=true
// when a name/slug filter resolved to nothing — that must return zero
// runs, not silently fall back to "no filter".
func (m *Manager) resolveAppAndFunctionFilters(ctx context.Context, filter cqrs.GetTraceRunFilter) (resolved cqrs.GetTraceRunFilter, noMatch bool, err error) {
	resolved = filter

	if len(filter.AppName) > 0 {
		var ids []uuid.UUID
		for _, name := range filter.AppName {
			app, aerr := m.GetAppByName(ctx, filter.WorkspaceID, name)
			if aerr != nil {
				if errors.Is(aerr, sql.ErrNoRows) {
					continue
				}
				return cqrs.GetTraceRunFilter{}, false, fmt.Errorf("duckdbquery: resolving app name %q: %w", name, aerr)
			}
			ids = append(ids, app.ID)
		}
		if len(ids) == 0 {
			return cqrs.GetTraceRunFilter{}, true, nil
		}
		resolved.AppID = append(append([]uuid.UUID{}, filter.AppID...), ids...)
	}

	if len(filter.FunctionSlug) > 0 {
		fns, ferr := m.GetFunctions(ctx)
		if ferr != nil {
			return cqrs.GetTraceRunFilter{}, false, fmt.Errorf("duckdbquery: resolving function slugs: %w", ferr)
		}
		want := make(map[string]struct{}, len(filter.FunctionSlug))
		for _, slug := range filter.FunctionSlug {
			want[slug] = struct{}{}
		}
		var ids []uuid.UUID
		for _, fn := range fns {
			if _, ok := want[fn.Slug]; ok {
				ids = append(ids, fn.ID)
			}
		}
		if len(ids) == 0 {
			return cqrs.GetTraceRunFilter{}, true, nil
		}
		resolved.FunctionID = append(append([]uuid.UUID{}, resolved.FunctionID...), ids...)
	}

	return resolved, false, nil
}

// runsQualify builds the QUALIFY clause every runs query uses to collapse
// each run_id's lifecycle rows down to its latest one and apply
// status/time-range filtering against that row — those columns vary across
// a run's lifecycle rows, so they must be evaluated together with the
// row-ranking window function, not a separate WHERE.
//
// Tiebreak by COALESCE(ended_at, started_at, queued_at) DESC, not
// inserted_at: a batch flush writes all of a run's lifecycle rows in one
// INSERT, and DuckDB evaluates a DEFAULT like current_timestamp once per
// statement, so every row in that flush gets an identical inserted_at.
// COALESCE instead picks the furthest lifecycle state each row's own
// columns encode.
func runsQualify(filter cqrs.GetTraceRunFilter) (string, []any) {
	where := []string{
		`ROW_NUMBER() OVER (
			PARTITION BY run_id ORDER BY COALESCE(ended_at, started_at, queued_at) DESC
		 ) = 1`,
	}
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

	// is_deferred is BOOLEAN NOT NULL DEFAULT FALSE (migrations/
	// 000001_baseline.sql) -- every non-deferred row is an explicit FALSE,
	// not NULL, so both branches compare directly.
	if filter.IsDeferred != nil {
		if *filter.IsDeferred {
			where = append(where, "is_deferred")
		} else {
			where = append(where, "NOT is_deferred")
		}
	}

	return strings.Join(where, " AND "), args
}

// buildRunsCursorSeek decodes a request cursor (if any) into the
// keyset-pagination predicate GetTraceRuns appends to runsQualify's
// QUALIFY clause, keyed on the same (orderCol, orderDir) pair used for
// ORDER BY, with run_id as the final tiebreak, always ascending. Returns
// "", nil, nil for the first page (no cursor, or nothing under this field).
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
// pagination, matching buildRunsCursorSeek's decode shape exactly.
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

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning trace run row: %w", err)
		}
		return nil, fmt.Errorf("run not found: %s", id.RunID)
	}
	return scanTraceRun(rows)
}

// GetTraceRunsByTriggerID returns every run triggered by the given event's
// internal ULID, collapsed to each run's latest lifecycle row exactly like
// GetTraceRuns — inngest.runs is append-only, so a run with multiple
// lifecycle rows must not surface more than once here.
func (m *Manager) GetTraceRunsByTriggerID(ctx context.Context, triggerID ulid.ULID) ([]*cqrs.TraceRun, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM %s.runs
		 WHERE list_contains(event_ids, ?)
		 QUALIFY ROW_NUMBER() OVER (
			PARTITION BY run_id ORDER BY COALESCE(ended_at, started_at, queued_at) DESC
		 ) = 1
		 ORDER BY queued_at ASC;`,
		runColumns, duckdb.DuckLakeAlias,
	)

	rows, err := m.db.QueryContext(ctx, query, triggerID.String())
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying trace runs by trigger id: %w", err)
	}
	defer rows.Close()

	out := []*cqrs.TraceRun{}
	for rows.Next() {
		trun, err := scanTraceRun(rows)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning trace run row: %w", err)
		}
		out = append(out, trun)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading trace run rows: %w", err)
	}
	return out, nil
}

// eventCELArrayMatchClause wraps an event.*-predicate SQL fragment
// (written against lambda parameter "x") in a check that at least one of
// the run's triggering events satisfies it. No join to inngest.events is
// needed: inngest.runs.inputs is already a JSON array of the run's
// triggering event(s). list_filter+lambda (rather than an OR of per-field
// checks) means every ANDed predicate must hold against the *same* array
// element — required for a batch trigger, where different events could
// otherwise each satisfy one predicate without any single event satisfying
// all of them.
func eventCELArrayMatchClause(fragment string) string {
	return fmt.Sprintf(`len(list_filter(json_transform(inputs, '["JSON"]'), lambda x: %s)) > 0`, fragment)
}

// GetTraceRuns pushes filter.CEL down into the SQL query rather than
// fetching a candidate set and post-filtering in Go: event.* predicates
// become an inputs-array match in the pre-collapse WHERE (see
// eventCELArrayMatchClause); output.*/error.* predicates are appended to
// the QUALIFY clause. A CEL string mixing both kinds via || has each side
// evaluated independently and ANDed back together here, which can accept a
// run neither side alone would — a pre-existing limitation shared with
// pkg/cqrs/manager's own CEL pushdown.
func (m *Manager) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	resolvedFilter, noMatch, err := m.resolveAppAndFunctionFilters(ctx, opt.Filter)
	if err != nil {
		return nil, err
	}
	if noMatch {
		return []*cqrs.TraceRun{}, nil
	}
	opt.Filter = resolvedFilter

	expHandler, err := run.NewExpressionHandler(ctx, run.WithExpressionHandlerBlob(opt.Filter.CEL, "\n"))
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: parsing CEL filter: %w", err)
	}

	preWhere, preArgs := latestRunsWhere(opt.Filter)
	qualify, qualifyArgs := runsQualify(opt.Filter)

	eventFilters, err := insights.CELEventFilters(ctx, expHandler.EventExprList)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: converting event CEL filter: %w", err)
	}
	if eventFrag, eventArgs, err := insights.RenderWhereSQL(eventFilters); err != nil {
		return nil, fmt.Errorf("duckdbquery: rendering event CEL filter: %w", err)
	} else if eventFrag != "" {
		preWhere += " AND " + eventCELArrayMatchClause(eventFrag)
		preArgs = append(preArgs, eventArgs...)
	}

	outputFilters, err := insights.CELOutputFilters(ctx, expHandler.OutputExprList)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: converting output CEL filter: %w", err)
	}
	if outputFrag, outputArgs, err := insights.RenderWhereSQL(outputFilters); err != nil {
		return nil, fmt.Errorf("duckdbquery: rendering output CEL filter: %w", err)
	} else if outputFrag != "" {
		qualify += " AND " + outputFrag
		qualifyArgs = append(qualifyArgs, outputArgs...)
	}

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
		qualify += " AND " + seekWhere
		qualifyArgs = append(qualifyArgs, seekArgs...)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s.runs WHERE %s QUALIFY %s ORDER BY %s %s, run_id ASC",
		runColumns, duckdb.DuckLakeAlias, preWhere, qualify, orderCol, orderDir,
	)
	args := append(preArgs, qualifyArgs...)
	// Every CEL predicate is applied in SQL above, so LIMIT can always be
	// pushed down.
	if opt.Items > 0 {
		query += " LIMIT ?"
		args = append(args, opt.Items+1)
	}
	query += ";"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying trace runs: %w", err)
	}
	defer rows.Close()

	var out []*cqrs.TraceRun
	for rows.Next() {
		trun, err := scanTraceRun(rows)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning trace run row: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading trace run rows: %w", err)
	}
	return out, nil
}

func (m *Manager) GetTraceRunsCount(ctx context.Context, opt cqrs.GetTraceRunOpt) (int, error) {
	resolvedFilter, noMatch, err := m.resolveAppAndFunctionFilters(ctx, opt.Filter)
	if err != nil {
		return 0, err
	}
	if noMatch {
		return 0, nil
	}
	opt.Filter = resolvedFilter

	expHandler, err := run.NewExpressionHandler(ctx, run.WithExpressionHandlerBlob(opt.Filter.CEL, "\n"))
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: parsing CEL filter: %w", err)
	}

	preWhere, preArgs := latestRunsWhere(opt.Filter)
	qualify, qualifyArgs := runsQualify(opt.Filter)

	eventFilters, err := insights.CELEventFilters(ctx, expHandler.EventExprList)
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: converting event CEL filter: %w", err)
	}
	if eventFrag, eventArgs, err := insights.RenderWhereSQL(eventFilters); err != nil {
		return 0, fmt.Errorf("duckdbquery: rendering event CEL filter: %w", err)
	} else if eventFrag != "" {
		preWhere += " AND " + eventCELArrayMatchClause(eventFrag)
		preArgs = append(preArgs, eventArgs...)
	}

	outputFilters, err := insights.CELOutputFilters(ctx, expHandler.OutputExprList)
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: converting output CEL filter: %w", err)
	}
	if outputFrag, outputArgs, err := insights.RenderWhereSQL(outputFilters); err != nil {
		return 0, fmt.Errorf("duckdbquery: rendering output CEL filter: %w", err)
	} else if outputFrag != "" {
		qualify += " AND " + outputFrag
		qualifyArgs = append(qualifyArgs, outputArgs...)
	}

	// COUNT(*) with no GROUP BY can't reference raw columns even inside
	// QUALIFY (DuckDB requires every referenced column to be part of an
	// aggregate), so the collapse-and-filter step runs in a derived table
	// and the outer query only aggregates its already-filtered output.
	query := fmt.Sprintf(
		"SELECT COUNT(*) AS c FROM (SELECT run_id FROM %s.runs WHERE %s QUALIFY %s);",
		duckdb.DuckLakeAlias, preWhere, qualify,
	)
	args := append(preArgs, qualifyArgs...)

	n, err := scanCount(m.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: querying trace run count: %w", err)
	}
	return int(n), nil
}

// scanTraceRun scans one row of a runColumns-shaped SELECT into a
// *cqrs.TraceRun. Destination order must match runColumns exactly.
func scanTraceRun(rows *sql.Rows) (*cqrs.TraceRun, error) {
	var (
		rawAccountID, rawEnvID, rawAppID, rawFunctionID any
		runID                                           string
		rawQueuedAt, rawStartedAt, rawEndedAt           any
		status                                          string
		rawOutput, rawEventIDs, rawIsDeferred           any
	)
	if err := rows.Scan(
		&rawAccountID, &rawEnvID, &rawAppID, &rawFunctionID, &runID,
		&rawQueuedAt, &rawStartedAt, &rawEndedAt, &status, &rawOutput, &rawEventIDs, &rawIsDeferred,
	); err != nil {
		return nil, err
	}

	accountID, err := uuidColumn(rawAccountID, "account_id")
	if err != nil {
		return nil, err
	}
	envID, err := uuidColumn(rawEnvID, "env_id")
	if err != nil {
		return nil, err
	}
	appID, err := uuidColumn(rawAppID, "app_id")
	if err != nil {
		return nil, err
	}
	functionID, err := uuidColumn(rawFunctionID, "function_id")
	if err != nil {
		return nil, err
	}
	queuedAt, err := asTimestamp(rawQueuedAt, "queued_at")
	if err != nil {
		return nil, err
	}
	startedAt, err := asNullableTimestamp(rawStartedAt, "started_at")
	if err != nil {
		return nil, err
	}
	endedAt, err := asNullableTimestamp(rawEndedAt, "ended_at")
	if err != nil {
		return nil, err
	}
	stepStatus, err := enums.StepStatusString(status)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: parsing run status %q: %w", status, err)
	}
	output, err := asJSON(rawOutput, "output")
	if err != nil {
		return nil, err
	}

	// event_ids is a VARCHAR[] (NULL for cron-only runs), decoded as []any.
	var triggerIDs []string
	if rawEventIDs != nil {
		items, ok := rawEventIDs.([]any)
		if !ok {
			return nil, fmt.Errorf("duckdbquery: event_ids has unexpected type %T", rawEventIDs)
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

	isDeferred, err := asNullableBool(rawIsDeferred, "is_deferred")
	if err != nil {
		return nil, err
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
		IsDeferred:  isDeferred,
	}, nil
}
