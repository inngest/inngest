// sessions.go holds the Sessions domain's query-key registry entries --
// the first registrants in this package (see registry.go for the shared
// machinery). Sessions groups runs by the (key, id) pairs a triggering
// event tagged them with (pkg/event.Sessions), materialized on every row
// of `inngest.runs` as its native `sessions STRUCT(key VARCHAR, id
// VARCHAR)[]` column (pkg/db/duckdb/migrations/000001_baseline.sql).
//
// The reference Cloud implementation (pkg/applogic/sessions) expands this
// with ClickHouse's `ARRAY JOIN sessions AS session`; DuckDB's equivalent
// is a FROM-clause `UNNEST(sessions) AS s(session)`, which -- confirmed
// empirically -- both fans out one row per (key, id) pair and drops rows
// with a NULL/empty sessions array entirely, matching ARRAY JOIN's
// inner-join-like semantics (a run with no session tags never appears in
// a Sessions listing). A nested UUID column (app_id, inside the
// struct_pack'd function list below) decodes as a plain Go string, same
// as a top-level UUID column -- also confirmed empirically.
//
// Unlike the reference, which resolves a session's functions via a
// separate Postgres workflow join, `inngest.runs` already carries
// function_slug/app_id/app_name directly on every row, so no extra lookup
// is needed here -- see FunctionRef.
package dashboards

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/enums"
)

const (
	KeySessionKeys = "session_keys"
	KeySessions    = "sessions"
	KeySessionRuns = "session_runs"
)

// defaultSessionWindow matches PR #4530's ("sessions oss") own default
// window for ListSessions/ListSessionRuns when the caller specifies no
// time range at all -- both the GQL (pkg/coreapi/graph/resolvers/
// sessions.go) and REST (pkg/api/v2/endpoints_sessions.go) callers rely on
// this default rather than each hardcoding their own.
const defaultSessionWindow = 7 * 24 * time.Hour

// latestRunsCTE returns the `WITH latest AS (...)` prefix every session
// query builds on, appending (accountID, envID) to args in the order
// they're bound. inngest.runs is append-only (one row per lifecycle
// transition of the same run_id), so every session query must collapse to
// each run's latest row first -- otherwise a run with several lifecycle
// rows would multiply its session tags, over-counting run_count/
// failed_run_count and duplicating it in a session's run list. Mirrors
// pkg/cqrs/duckdbquery/runs.go's own runsQualify collapse, reimplemented
// rather than imported: that package depends on pkg/duckdb/insights
// already, and adding the reverse edge (insights-adjacent packages
// depending back on duckdbquery) for a two-line window function isn't
// worth the coupling.
func latestRunsCTE(scope Scope, args *[]any) string {
	*args = append(*args, scope.AccountID.String(), scope.EnvID.String())
	return fmt.Sprintf(`WITH latest AS (
  SELECT run_id, app_id, app_name, function_slug, status, queued_at, started_at, ended_at, sessions, inputs
  FROM %s.runs
  WHERE account_id = ? AND env_id = ?
  QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY COALESCE(ended_at, started_at, queued_at) DESC) = 1
)`, duckdb.DuckLakeAlias)
}

// escapeLikeSearch escapes a caller-supplied search string for embedding
// inside a `LIKE ... ESCAPE '\'` pattern, mirroring the reference
// implementation's own helper of the same name (pkg/applogic/sessions/keys.go).
func escapeLikeSearch(search string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
}

// buildSessionKeysSQL lists every distinct session key observed, with its
// earliest-seen run's queued_at as created_at. Unlike the reference, which
// maintains a dedicated ingest-time Postgres table for this (pkg/applogic/
// sessions/keys.go's session_keys table + upserter), this queries
// inngest.runs directly -- a dev server's data volume doesn't warrant a
// second write path just to keep key listing cheap.
func buildSessionKeysSQL(scope Scope) (string, []any, error) {
	var args []any
	query := latestRunsCTE(scope, &args)

	where := ""
	if search := escapeLikeSearch(strings.TrimSpace(scope.Search)); search != "" {
		where = "\nWHERE session.key ILIKE ? ESCAPE '\\'"
		args = append(args, "%"+search+"%")
	}

	having := ""
	if scope.Cursor != "" {
		cur, err := decodeSessionCursor(scope.Cursor, "created_at")
		if err != nil {
			return "", nil, err
		}
		if cur != nil {
			having = "\nHAVING (created_at < ? OR (created_at = ? AND key < ?))"
			args = append(args, cur.at, cur.at, cur.id)
		}
	}

	query += fmt.Sprintf(`
SELECT session.key AS key, MIN(queued_at) AS created_at
FROM latest, UNNEST(sessions) AS s(session)%s
GROUP BY session.key%s
ORDER BY created_at DESC, key DESC
LIMIT ?;`, where, having)
	args = append(args, resolveLimit(scope.Limit)+1)

	return query, args, nil
}

// buildSessionsSQL groups one session key's runs by session id: run
// counts, failure rate inputs, last activity, and every distinct
// (function, app) pair a run in that group belongs to.
func buildSessionsSQL(scope Scope) (string, []any, error) {
	if scope.SessionKey == "" {
		return "", nil, fmt.Errorf("session key is required")
	}

	var args []any
	query := latestRunsCTE(scope, &args)

	where := "\nWHERE session.key = ?"
	args = append(args, scope.SessionKey)
	if search := escapeLikeSearch(strings.TrimSpace(scope.Search)); search != "" {
		where += " AND session.id ILIKE ? ESCAPE '\\'"
		args = append(args, "%"+search+"%")
	}
	where += " AND queued_at BETWEEN ? AND ?"
	args = append(args, scope.From, scope.To)

	having := ""
	if scope.Cursor != "" {
		cur, err := decodeSessionCursor(scope.Cursor, "last_active_at")
		if err != nil {
			return "", nil, err
		}
		if cur != nil {
			having = "\nHAVING (last_active_at < ? OR (last_active_at = ? AND session_id < ?))"
			args = append(args, cur.at, cur.at, cur.id)
		}
	}

	query += fmt.Sprintf(`
SELECT
  session.id AS session_id,
  COUNT(*) AS run_count,
  COUNT(*) FILTER (WHERE status = 'Failed') AS failed_run_count,
  MAX(queued_at) AS last_active_at,
  list_distinct(list(struct_pack(function_slug, app_id, app_name))) AS functions
FROM latest, UNNEST(sessions) AS s(session)%s
GROUP BY session.id%s
ORDER BY last_active_at DESC, session_id DESC
LIMIT ?;`, where, having)
	args = append(args, resolveLimit(scope.Limit)+1)

	return query, args, nil
}

// buildSessionRunsSQL lists the individual runs carrying one (key, id)
// session pair. event_name is pulled from the triggering event(s) already
// stored on inputs (a JSON array of full event.Event objects) -- the
// reference reads a dedicated triggering_event_name column that has no
// OSS equivalent, so this extracts the first triggering event's name
// instead, matching runs.go's own inputs-is-the-event-source convention.
func buildSessionRunsSQL(scope Scope) (string, []any, error) {
	if scope.SessionKey == "" || scope.SessionID == "" {
		return "", nil, fmt.Errorf("session key and session id are required")
	}

	var args []any
	query := latestRunsCTE(scope, &args)

	where := "\nWHERE session.key = ? AND session.id = ? AND queued_at BETWEEN ? AND ?"
	args = append(args, scope.SessionKey, scope.SessionID, scope.From, scope.To)

	if scope.Cursor != "" {
		cur, err := decodeSessionCursor(scope.Cursor, "queued_at")
		if err != nil {
			return "", nil, err
		}
		if cur != nil {
			where += "\n  AND (queued_at < ? OR (queued_at = ? AND run_id < ?))"
			args = append(args, cur.at, cur.at, cur.id)
		}
	}

	query += fmt.Sprintf(`
SELECT
  run_id, app_id, app_name, function_slug, status, queued_at, started_at, ended_at,
  json_extract_string(inputs, '$[0].name') AS event_name
FROM latest, UNNEST(sessions) AS s(session)%s
ORDER BY queued_at DESC, run_id DESC
LIMIT ?;`, where)
	args = append(args, resolveLimit(scope.Limit)+1)

	return query, args, nil
}

type sessionCursor struct {
	at time.Time
	id string
}

// decodeSessionCursor decodes a request cursor into the keyset predicate
// every builder above appends, keyed on field (e.g. "last_active_at"),
// with the row/group identifier as the final tiebreak. Reuses
// cqrs.TracePageCursor (the same codec pkg/cqrs/duckdbquery's own runs/
// events pagination already uses) rather than a bespoke cursor type per
// entry. Returns nil, nil for the first page (no cursor, or nothing under
// this field).
func decodeSessionCursor(cursorStr, field string) (*sessionCursor, error) {
	if cursorStr == "" {
		return nil, nil
	}
	cur := &cqrs.TracePageCursor{}
	if err := cur.Decode(cursorStr); err != nil {
		return nil, fmt.Errorf("dashboard: decoding cursor: %w", err)
	}
	tc := cur.Find(field)
	if tc == nil || cur.ID == "" {
		return nil, nil
	}
	return &sessionCursor{at: time.UnixMilli(tc.Value), id: cur.ID}, nil
}

// encodeSessionCursor builds one response row's cursor, matching
// decodeSessionCursor's shape exactly.
func encodeSessionCursor(id, field string, at time.Time) (string, error) {
	c := cqrs.TracePageCursor{
		ID: id,
		Cursors: map[string]cqrs.TraceCursor{
			field: {Field: field, Value: at.UnixMilli()},
		},
	}
	return c.Encode()
}

// resolveSessionTimeRange applies defaultSessionWindow when the caller
// specifies no range at all, matching the reference's own
// sessions.resolveTimeRange default; an explicit range is validated
// (from < to) but otherwise passed through unbounded by any retention
// window -- OSS has no billing-tier history limit to enforce.
func resolveSessionTimeRange(from, until time.Time) (time.Time, time.Time, error) {
	if from.IsZero() && until.IsZero() {
		to := time.Now().UTC()
		return to.Add(-defaultSessionWindow), to, nil
	}
	if !from.Before(until) {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"invalid time range: from (%s) must be before until (%s)",
			from.Format(time.RFC3339), until.Format(time.RFC3339),
		)
	}
	return from, until, nil
}

// SessionKey is one distinct session key observed in the environment.
type SessionKey struct {
	Key       string
	CreatedAt time.Time
	Cursor    string
}

// SessionKeysOpts are ListSessionKeys' caller-provided parameters.
type SessionKeysOpts struct {
	Search string
	Cursor string
	Limit  int
}

// SessionKeysResult is ListSessionKeys' output.
type SessionKeysResult struct {
	Keys    []SessionKey
	HasMore bool
}

// ListSessionKeys lists every distinct session key observed in
// (accountID, envID), most-recently-first-seen first.
func ListSessionKeys(ctx context.Context, db *sql.DB, accountID, envID uuid.UUID, opts SessionKeysOpts) (*SessionKeysResult, error) {
	scope := Scope{AccountID: accountID, EnvID: envID, Search: opts.Search, Cursor: opts.Cursor, Limit: opts.Limit}
	result, err := Get(ctx, db, scope, KeySessionKeys)
	if err != nil {
		return nil, fmt.Errorf("dashboard: listing session keys: %w", err)
	}

	limit := resolveLimit(opts.Limit)
	idx := columnIndex(result.Columns)
	keys := make([]SessionKey, 0, len(result.Rows))
	for _, row := range result.Rows {
		key, err := asString(col(row, idx, "key"), "key")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		createdAt, err := asTime(col(row, idx, "created_at"), "created_at")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		cursor, err := encodeSessionCursor(key, "created_at", createdAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard: encoding session key cursor: %w", err)
		}
		keys = append(keys, SessionKey{Key: key, CreatedAt: createdAt, Cursor: cursor})
	}

	hasMore := len(keys) > limit
	if hasMore {
		keys = keys[:limit]
	}
	return &SessionKeysResult{Keys: keys, HasMore: hasMore}, nil
}

// FunctionRef identifies one function/app pair a session's runs belong
// to. Unlike the reference (which only tracks a bare function slug and
// resolves app/name via a separate lookup), inngest.runs already carries
// app_id/app_name directly, so both are populated here with no extra
// round trip.
type FunctionRef struct {
	Slug    string
	AppID   uuid.UUID
	AppName string
}

// SessionGroup is one session id's aggregate view within a session key.
type SessionGroup struct {
	ID             string
	RunCount       int
	FailedRunCount int
	LastActiveAt   time.Time
	Functions      []FunctionRef
	Cursor         string
}

// FailureRate returns the fraction (0-1) of this group's runs that
// failed, 0 for a group with no runs (shouldn't occur -- every group here
// is built from at least one matching run).
func (g SessionGroup) FailureRate() float64 {
	if g.RunCount <= 0 {
		return 0
	}
	return float64(g.FailedRunCount) / float64(g.RunCount)
}

// SessionsOpts are ListSessions' caller-provided parameters. From/Until
// zero means "use the default window" (see resolveSessionTimeRange).
type SessionsOpts struct {
	Key      string
	IDSearch string
	From     time.Time
	Until    time.Time
	Cursor   string
	Limit    int
}

// SessionsResult is ListSessions' output.
type SessionsResult struct {
	Sessions []SessionGroup
	HasMore  bool
}

// ListSessions lists the session groups (one per session id) observed
// under one session key, most-recently-active first.
func ListSessions(ctx context.Context, db *sql.DB, accountID, envID uuid.UUID, opts SessionsOpts) (*SessionsResult, error) {
	if opts.Key == "" {
		return nil, fmt.Errorf("dashboard: session key is required")
	}
	from, until, err := resolveSessionTimeRange(opts.From, opts.Until)
	if err != nil {
		return nil, fmt.Errorf("dashboard: %w", err)
	}

	scope := Scope{
		AccountID: accountID, EnvID: envID,
		From: from, To: until,
		SessionKey: opts.Key, Search: opts.IDSearch,
		Cursor: opts.Cursor, Limit: opts.Limit,
	}
	result, err := Get(ctx, db, scope, KeySessions)
	if err != nil {
		return nil, fmt.Errorf("dashboard: listing sessions: %w", err)
	}

	limit := resolveLimit(opts.Limit)
	idx := columnIndex(result.Columns)
	groups := make([]SessionGroup, 0, len(result.Rows))
	for _, row := range result.Rows {
		id, err := asString(col(row, idx, "session_id"), "session_id")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		runCount, err := asInt64(col(row, idx, "run_count"), "run_count")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		failedRunCount, err := asInt64(col(row, idx, "failed_run_count"), "failed_run_count")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		lastActiveAt, err := asTime(col(row, idx, "last_active_at"), "last_active_at")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		functions, err := asFunctionRefs(col(row, idx, "functions"))
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		cursor, err := encodeSessionCursor(id, "last_active_at", lastActiveAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard: encoding session cursor: %w", err)
		}

		groups = append(groups, SessionGroup{
			ID:             id,
			RunCount:       int(runCount),
			FailedRunCount: int(failedRunCount),
			LastActiveAt:   lastActiveAt,
			Functions:      functions,
			Cursor:         cursor,
		})
	}

	hasMore := len(groups) > limit
	if hasMore {
		groups = groups[:limit]
	}
	return &SessionsResult{Sessions: groups, HasMore: hasMore}, nil
}

// SessionRun is one run carrying a specific (key, id) session pair.
type SessionRun struct {
	RunID     string
	Function  FunctionRef
	EventName string
	Status    enums.RunStatus
	QueuedAt  time.Time
	StartedAt time.Time
	EndedAt   time.Time
	Cursor    string
}

// SessionRunsOpts are ListSessionRuns' caller-provided parameters.
type SessionRunsOpts struct {
	Key    string
	ID     string
	From   time.Time
	Until  time.Time
	Cursor string
	Limit  int
}

// SessionRunsResult is ListSessionRuns' output.
type SessionRunsResult struct {
	Runs    []SessionRun
	HasMore bool
}

// ListSessionRuns lists the individual runs carrying one (key, id)
// session pair, most-recently-queued first.
func ListSessionRuns(ctx context.Context, db *sql.DB, accountID, envID uuid.UUID, opts SessionRunsOpts) (*SessionRunsResult, error) {
	if opts.Key == "" || opts.ID == "" {
		return nil, fmt.Errorf("dashboard: session key and session id are required")
	}
	from, until, err := resolveSessionTimeRange(opts.From, opts.Until)
	if err != nil {
		return nil, fmt.Errorf("dashboard: %w", err)
	}

	scope := Scope{
		AccountID: accountID, EnvID: envID,
		From: from, To: until,
		SessionKey: opts.Key, SessionID: opts.ID,
		Cursor: opts.Cursor, Limit: opts.Limit,
	}
	result, err := Get(ctx, db, scope, KeySessionRuns)
	if err != nil {
		return nil, fmt.Errorf("dashboard: listing session runs: %w", err)
	}

	limit := resolveLimit(opts.Limit)
	idx := columnIndex(result.Columns)
	runs := make([]SessionRun, 0, len(result.Rows))
	for _, row := range result.Rows {
		run, err := parseSessionRunRow(row, idx)
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		runs = append(runs, run)
	}

	hasMore := len(runs) > limit
	if hasMore {
		runs = runs[:limit]
	}
	return &SessionRunsResult{Runs: runs, HasMore: hasMore}, nil
}

func parseSessionRunRow(row []any, idx map[string]int) (SessionRun, error) {
	runID, err := asString(col(row, idx, "run_id"), "run_id")
	if err != nil {
		return SessionRun{}, err
	}
	appID, err := asUUID(col(row, idx, "app_id"), "app_id")
	if err != nil {
		return SessionRun{}, err
	}
	appName, err := asString(col(row, idx, "app_name"), "app_name")
	if err != nil {
		return SessionRun{}, err
	}
	functionSlug, err := asString(col(row, idx, "function_slug"), "function_slug")
	if err != nil {
		return SessionRun{}, err
	}
	statusStr, err := asString(col(row, idx, "status"), "status")
	if err != nil {
		return SessionRun{}, err
	}
	stepStatus, err := enums.StepStatusString(statusStr)
	if err != nil {
		return SessionRun{}, fmt.Errorf("parsing run status %q: %w", statusStr, err)
	}
	queuedAt, err := asTime(col(row, idx, "queued_at"), "queued_at")
	if err != nil {
		return SessionRun{}, err
	}
	startedAt, err := asNullableTime(col(row, idx, "started_at"))
	if err != nil {
		return SessionRun{}, err
	}
	endedAt, err := asNullableTime(col(row, idx, "ended_at"))
	if err != nil {
		return SessionRun{}, err
	}
	eventName, err := asNullableString(col(row, idx, "event_name"))
	if err != nil {
		return SessionRun{}, err
	}

	cursor, err := encodeSessionCursor(runID, "queued_at", queuedAt)
	if err != nil {
		return SessionRun{}, fmt.Errorf("encoding session run cursor: %w", err)
	}

	return SessionRun{
		RunID:     runID,
		Function:  FunctionRef{Slug: functionSlug, AppID: appID, AppName: appName},
		EventName: eventName,
		Status:    enums.StepStatusToRunStatus(stepStatus),
		QueuedAt:  queuedAt,
		StartedAt: startedAt,
		EndedAt:   endedAt,
		Cursor:    cursor,
	}, nil
}
