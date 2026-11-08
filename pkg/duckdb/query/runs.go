package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/run"
	"github.com/oklog/ulid/v2"
)

const runColumns = "account_id, env_id, app_id, function_id, run_id, queued_at, started_at, ended_at, status, output, event_ids, is_deferred, trace_id"

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

// latestRunOrder is the order every runs query picks a run_id's latest
// lifecycle row by -- see runsQualify's doc comment.
const latestRunOrder = `COALESCE(ended_at, started_at, queued_at) DESC,
				ended_at IS NOT NULL DESC,
				CASE
					WHEN status IN ('Completed', 'Failed', 'Errored', 'Cancelled', 'TimedOut', 'Skipped') THEN 2
					WHEN status IN ('Running', 'Waiting', 'Sleeping', 'Invoking') THEN 1
					ELSE 0
				END DESC,
				status DESC`

// runsQualify builds the QUALIFY clause every runs query uses to collapse
// each run_id's lifecycle rows down to its latest one and apply
// status/time-range filtering against that row — those columns vary across
// a run's lifecycle rows, so they must be evaluated together with the
// row-ranking window function, not a separate WHERE.
//
// Ordered by COALESCE(ended_at, started_at, queued_at) DESC: a batch flush
// writes all of a run's lifecycle rows in one INSERT, so no insert-time
// column could order them; COALESCE picks the furthest lifecycle state each
// row's own columns encode. Timestamps are ms precision so rows can still
// tie (a Queued and a Running row in the same ms), so the rest of the order
// prefers the furthest-along row, matching the inngest.insights_runs macro
// and dashboards.latestRunOrder.
func runsQualify(filter cqrs.GetTraceRunFilter) (string, []any) {
	where := []string{
		`ROW_NUMBER() OVER (
			PARTITION BY run_id ORDER BY ` + latestRunOrder + `
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
	where, args := runsSeek(orderCol, orderDir, time.UnixMilli(tc.Value), cur.ID)
	return where, args, nil
}

// runsSeek is the keyset-pagination predicate for rows after (val, runID)
// in (orderCol orderDir, run_id ASC) order.
func runsSeek(orderCol, orderDir string, val time.Time, runID string) (string, []any) {
	cmp := ">"
	if orderDir == "DESC" {
		cmp = "<"
	}
	return fmt.Sprintf("(%s %s ? OR (%s = ? AND run_id > ?))", orderCol, cmp, orderCol), []any{val, val, runID}
}

func runOrderTime(run *cqrs.TraceRun, orderCol string) time.Time {
	switch orderCol {
	case "started_at":
		return run.StartedAt
	case "ended_at":
		return run.EndedAt
	default:
		return run.QueuedAt
	}
}

// encodeRunsCursor builds the per-row response cursor for keyset
// pagination, matching buildRunsCursorSeek's decode shape exactly.
func encodeRunsCursor(run *cqrs.TraceRun, orderCol string) (string, error) {
	c := cqrs.TracePageCursor{
		ID: run.RunID,
		Cursors: map[string]cqrs.TraceCursor{
			orderCol: {Field: orderCol, Value: runOrderTime(run, orderCol).UnixMilli()},
		},
	}
	return c.Encode()
}

func (m *Manager) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM %s.runs WHERE run_id = ? ORDER BY %s LIMIT 1;",
		runColumns, driver.DuckLakeAlias, latestRunOrder,
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
			PARTITION BY run_id ORDER BY %s
		 ) = 1
		 ORDER BY queued_at ASC;`,
		runColumns, driver.DuckLakeAlias, latestRunOrder,
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
// all of them. inputs is a VARIANT column, so it's cast to JSON first:
// json_transform would otherwise stringify it with DuckDB's own struct
// syntax and fail to parse that (see pkg/duckdb/insights' handleJSONFilter).
func eventCELArrayMatchClause(fragment string) string {
	return fmt.Sprintf(`len(list_filter(json_transform(inputs::JSON, '["JSON"]'), lambda x: %s)) > 0`, fragment)
}

// runsQuery is the WHERE/QUALIFY halves of a runs query for one
// cqrs.GetTraceRunOpt, plus the Go matcher every row it selects still has
// to pass (nil when opt has no CEL filter).
type runsQuery struct {
	preWhere    string
	preArgs     []any
	qualify     string
	qualifyArgs []any
	matcher     *runCELMatcher
}

// buildRunsQuery converts filter.CEL into SQL: event.* predicates become an
// inputs-array match in the pre-collapse WHERE (see
// eventCELArrayMatchClause); output.*/error.* predicates are appended to
// the QUALIFY clause. A CEL string mixing both kinds is only divided along
// its top-level && conjuncts; a conjunct mixing both (e.g. an || across
// them) lands whole in the QUALIFY half, which matches its event.* parts
// against inputs itself -- see pkg/duckdb/insights' celExprsToSQL.
//
// That SQL is only a superset of what filter.CEL means (see celmatch.go),
// so callers must also check each row against the returned matcher.
func buildRunsQuery(ctx context.Context, filter cqrs.GetTraceRunFilter) (runsQuery, error) {
	expHandler, err := run.NewExpressionHandler(ctx, run.WithExpressionHandlerBlob(filter.CEL, "\n"))
	if err != nil {
		return runsQuery{}, fmt.Errorf("duckdbquery: parsing CEL filter: %w", err)
	}

	q := runsQuery{}
	q.preWhere, q.preArgs = latestRunsWhere(filter)
	q.qualify, q.qualifyArgs = runsQualify(filter)

	eventFilters, err := CELEventFilters(ctx, expHandler.EventExprList)
	if err != nil {
		return runsQuery{}, fmt.Errorf("duckdbquery: converting event CEL filter: %w", err)
	}
	if eventFrag, eventArgs, err := RenderWhereSQL(eventFilters); err != nil {
		return runsQuery{}, fmt.Errorf("duckdbquery: rendering event CEL filter: %w", err)
	} else if eventFrag != "" {
		q.preWhere += " AND " + eventCELArrayMatchClause(eventFrag)
		q.preArgs = append(q.preArgs, eventArgs...)
	}

	outputFilters, err := CELOutputFilters(ctx, expHandler.OutputExprList)
	if err != nil {
		return runsQuery{}, fmt.Errorf("duckdbquery: converting output CEL filter: %w", err)
	}
	if outputFrag, outputArgs, err := RenderWhereSQL(outputFilters); err != nil {
		return runsQuery{}, fmt.Errorf("duckdbquery: rendering output CEL filter: %w", err)
	} else if outputFrag != "" {
		q.qualify += " AND " + outputFrag
		q.qualifyArgs = append(q.qualifyArgs, outputArgs...)
	}

	q.matcher, err = newRunCELMatcher(ctx, expHandler)
	if err != nil {
		return runsQuery{}, err
	}
	return q, nil
}

// GetTraceRuns pages through runs in (TimeField, run_id) order. Without a
// CEL filter, LIMIT is pushed down directly. With one, the SQL is only a
// prefilter (see buildRunsQuery), so candidates are fetched in
// keyset-paginated batches and re-checked in Go until opt.Items runs match
// or the candidates run out.
func (m *Manager) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	resolvedFilter, noMatch, err := m.resolveAppAndFunctionFilters(ctx, opt.Filter)
	if err != nil {
		return nil, err
	}
	if noMatch {
		return []*cqrs.TraceRun{}, nil
	}
	opt.Filter = resolvedFilter

	rq, err := buildRunsQuery(ctx, opt.Filter)
	if err != nil {
		return nil, err
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

	// The query fetches Items+1 rows when it can push LIMIT down, but like
	// the SQL manager this returns at most Items: callers detect a next page
	// themselves (the GQL resolver compares len to Items; the REST run
	// provider asks for Limit+1).
	var limit uint
	if opt.Items > 0 {
		limit = opt.Items + 1
		if rq.matcher != nil {
			limit = celCandidateBatch(opt.Items)
		}
	}

	var out []*cqrs.TraceRun
	for {
		scanned, last, err := m.getTraceRunsBatch(ctx, opt, rq, orderCol, orderDir, seekWhere, seekArgs, limit, &out)
		if err != nil {
			return nil, err
		}
		full := opt.Items > 0 && uint(len(out)) >= opt.Items
		if full || rq.matcher == nil || limit == 0 || scanned < limit {
			return out, nil
		}
		// Seek from the last candidate's exact order value rather than its
		// millisecond-truncated response cursor, so no candidate between two
		// batches is skipped.
		seekWhere, seekArgs = runsSeek(orderCol, orderDir, runOrderTime(last, orderCol), last.RunID)
	}
}

// getTraceRunsBatch runs one GetTraceRuns query (at most limit rows, or
// all of them when limit is 0) after seekWhere, appending each row that
// passes rq.matcher to out until out holds opt.Items runs. Returns how many
// candidate rows it scanned and the last one.
func (m *Manager) getTraceRunsBatch(ctx context.Context, opt cqrs.GetTraceRunOpt, rq runsQuery, orderCol, orderDir, seekWhere string, seekArgs []any, limit uint, out *[]*cqrs.TraceRun) (uint, *cqrs.TraceRun, error) {
	qualify := rq.qualify
	qualifyArgs := append([]any{}, rq.qualifyArgs...)
	if seekWhere != "" {
		qualify += " AND " + seekWhere
		qualifyArgs = append(qualifyArgs, seekArgs...)
	}

	cols := runColumns
	if rq.matcher != nil {
		cols += ", inputs"
	}
	query := fmt.Sprintf(
		"SELECT %s FROM %s.runs WHERE %s QUALIFY %s ORDER BY %s %s, run_id ASC",
		cols, driver.DuckLakeAlias, rq.preWhere, qualify, orderCol, orderDir,
	)
	args := append(append([]any{}, rq.preArgs...), qualifyArgs...)
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	query += ";"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, nil, fmt.Errorf("duckdbquery: querying trace runs: %w", err)
	}
	defer rows.Close()

	var (
		scanned uint
		last    *cqrs.TraceRun
	)
	for rows.Next() {
		var (
			trun *cqrs.TraceRun
			err  error
		)
		if rq.matcher != nil {
			var rawInputs any
			trun, err = scanTraceRun(rows, &rawInputs)
			if err != nil {
				return 0, nil, fmt.Errorf("duckdbquery: scanning trace run row: %w", err)
			}
			scanned++
			last = trun
			ok, err := rq.matchRow(ctx, rawInputs, trun.Output)
			if err != nil {
				return 0, nil, err
			}
			if !ok {
				continue
			}
		} else {
			trun, err = scanTraceRun(rows)
			if err != nil {
				return 0, nil, fmt.Errorf("duckdbquery: scanning trace run row: %w", err)
			}
			scanned++
			last = trun
		}

		trun.Cursor, err = encodeRunsCursor(trun, orderCol)
		if err != nil {
			return 0, nil, fmt.Errorf("duckdbquery: encoding run cursor: %w", err)
		}
		*out = append(*out, trun)
		if opt.Items > 0 && uint(len(*out)) >= opt.Items {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("duckdbquery: reading trace run rows: %w", err)
	}
	return scanned, last, nil
}

// matchRow checks one candidate row's scanned inputs column and output
// against rq.matcher.
func (rq runsQuery) matchRow(ctx context.Context, rawInputs any, output []byte) (bool, error) {
	inputs, err := asJSON(rawInputs, "inputs")
	if err != nil {
		return false, err
	}
	return rq.matcher.match(ctx, inputs, output)
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

	rq, err := buildRunsQuery(ctx, opt.Filter)
	if err != nil {
		return 0, err
	}
	args := append(append([]any{}, rq.preArgs...), rq.qualifyArgs...)

	if rq.matcher != nil {
		return m.countTraceRunsInGo(ctx, rq, args)
	}

	// COUNT(*) with no GROUP BY can't reference raw columns even inside
	// QUALIFY (DuckDB requires every referenced column to be part of an
	// aggregate), so the collapse-and-filter step runs in a derived table
	// and the outer query only aggregates its already-filtered output.
	query := fmt.Sprintf(
		"SELECT COUNT(*) AS c FROM (SELECT run_id FROM %s.runs WHERE %s QUALIFY %s);",
		driver.DuckLakeAlias, rq.preWhere, rq.qualify,
	)
	n, err := scanCount(m.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: querying trace run count: %w", err)
	}
	return int(n), nil
}

// countTraceRunsInGo counts the runs rq selects that also pass rq.matcher,
// which (unlike rq's SQL) can't be pushed into a COUNT(*).
func (m *Manager) countTraceRunsInGo(ctx context.Context, rq runsQuery, args []any) (int, error) {
	query := fmt.Sprintf(
		"SELECT inputs, output FROM %s.runs WHERE %s QUALIFY %s;",
		driver.DuckLakeAlias, rq.preWhere, rq.qualify,
	)
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("duckdbquery: querying trace run count: %w", err)
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var rawInputs, rawOutput any
		if err := rows.Scan(&rawInputs, &rawOutput); err != nil {
			return 0, fmt.Errorf("duckdbquery: scanning trace run count row: %w", err)
		}
		output, err := asJSON(rawOutput, "output")
		if err != nil {
			return 0, err
		}
		ok, err := rq.matchRow(ctx, rawInputs, output)
		if err != nil {
			return 0, err
		}
		if ok {
			n++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("duckdbquery: reading trace run count rows: %w", err)
	}
	return n, nil
}

// scanTraceRun scans one row of a runColumns-shaped SELECT into a
// *cqrs.TraceRun. Destination order must match runColumns exactly; extra
// receives any columns selected after runColumns.
func scanTraceRun(rows *sql.Rows, extra ...any) (*cqrs.TraceRun, error) {
	var (
		rawAccountID, rawEnvID, rawAppID, rawFunctionID any
		runID                                           string
		rawQueuedAt, rawStartedAt, rawEndedAt           any
		status                                          string
		rawOutput, rawEventIDs, rawIsDeferred           any
		traceID                                         sql.NullString
	)
	dest := append([]any{
		&rawAccountID, &rawEnvID, &rawAppID, &rawFunctionID, &runID,
		&rawQueuedAt, &rawStartedAt, &rawEndedAt, &status, &rawOutput, &rawEventIDs, &rawIsDeferred,
		&traceID,
	}, extra...)
	if err := rows.Scan(dest...); err != nil {
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
		TraceID:     traceID.String, // NULL when the source span carried no trace ID
		QueuedAt:    queuedAt,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		Output:      output,
		Status:      enums.StepStatusToRunStatus(stepStatus),
		TriggerIDs:  triggerIDs,
		IsDeferred:  isDeferred,
	}, nil
}
