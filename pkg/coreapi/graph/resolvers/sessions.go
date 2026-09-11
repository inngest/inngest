package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/duckdb/dashboards"
	"github.com/inngest/inngest/pkg/enums"
)

// Sessions backs Query.sessionKeys/sessions/sessionRuns -- the GQL
// counterpart of pkg/api/v2/endpoints_sessions.go, both calling directly
// into pkg/duckdb/dashboards' query-key registry against qr.DuckDB, the
// same dual-write connection Query.insights runs against
// (insights.go). Unlike Query.insights, there's no ad-hoc-SQL pipeline in
// between: these are backend-composed queries, not user-written SQL.
//
// Schema shape (SessionKey/SessionGroup/SessionRun/SessionFunction, plus
// field names/args) intentionally matches PR #4530 ("sessions oss",
// https://github.com/inngest/inngest/pull/4530) even though that PR's own
// resolvers read from the primary SQLite/Postgres store via
// pkg/cqrs/manager/sessions.go rather than DuckDB -- matching its schema
// means the already-shared dev-server-ui components (SessionKeys/
// SessionResults under ui/packages/components/src/Sessions) work against
// either backend unmodified.
//
// A nil qr.DuckDB means Sessions requires --duckdb dual-write to be
// enabled, the same "not wired up" contract Query.insights uses.

func (qr *queryResolver) SessionKeys(ctx context.Context, search *string) ([]*models.SessionKey, error) {
	if qr.DuckDB == nil {
		return nil, fmt.Errorf("sessions requires dual-write (--duckdb) to be enabled")
	}

	var query string
	if search != nil {
		query = *search
	}

	result, err := dashboards.ListSessionKeys(ctx, qr.DuckDB, consts.DevServerAccountID, consts.DevServerEnvID, dashboards.SessionKeysOpts{
		Search: query,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*models.SessionKey, len(result.Keys))
	for i, key := range result.Keys {
		out[i] = &models.SessionKey{
			SessionKey: key.Key,
			CreatedAt:  key.CreatedAt,
		}
	}
	return out, nil
}

func (qr *queryResolver) Sessions(ctx context.Context, sessionKey string, sessionIDSearch *string, timeRange *models.TimeRangeInput) ([]*models.SessionGroup, error) {
	if qr.DuckDB == nil {
		return nil, fmt.Errorf("sessions requires dual-write (--duckdb) to be enabled")
	}

	var idSearch string
	if sessionIDSearch != nil {
		idSearch = *sessionIDSearch
	}
	from, until := sessionTimeRange(timeRange)

	result, err := dashboards.ListSessions(ctx, qr.DuckDB, consts.DevServerAccountID, consts.DevServerEnvID, dashboards.SessionsOpts{
		Key:      sessionKey,
		IDSearch: idSearch,
		From:     from,
		Until:    until,
	})
	if err != nil {
		return nil, err
	}

	functionNames, err := qr.sessionFunctionNames(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*models.SessionGroup, len(result.Sessions))
	for i, group := range result.Sessions {
		functions := make([]*models.SessionFunction, len(group.Functions))
		for j, fn := range group.Functions {
			functions[j] = &models.SessionFunction{Slug: fn.Slug, Name: functionNames(fn.Slug)}
		}

		out[i] = &models.SessionGroup{
			SessionKey:     sessionKey,
			SessionID:      group.ID,
			RunCount:       group.RunCount,
			FailedRunCount: group.FailedRunCount,
			FailureRate:    group.FailureRate(),
			LastActiveAt:   group.LastActiveAt,
			Functions:      functions,
		}
	}
	return out, nil
}

func (qr *queryResolver) SessionRuns(ctx context.Context, sessionKey string, sessionID string, timeRange *models.TimeRangeInput) ([]*models.SessionRun, error) {
	if qr.DuckDB == nil {
		return nil, fmt.Errorf("sessions requires dual-write (--duckdb) to be enabled")
	}

	from, until := sessionTimeRange(timeRange)
	result, err := dashboards.ListSessionRuns(ctx, qr.DuckDB, consts.DevServerAccountID, consts.DevServerEnvID, dashboards.SessionRunsOpts{
		Key:   sessionKey,
		ID:    sessionID,
		From:  from,
		Until: until,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*models.SessionRun, len(result.Runs))
	for i, run := range result.Runs {
		sr := &models.SessionRun{
			ID:           run.RunID,
			FunctionSlug: run.Function.Slug,
			Status:       sessionRunStatusString(run.Status),
			QueuedAt:     run.QueuedAt,
		}
		if run.EventName != "" {
			eventName := run.EventName
			sr.EventName = &eventName
		}
		if !run.StartedAt.IsZero() {
			startedAt := run.StartedAt
			sr.StartedAt = &startedAt
		}
		if !run.EndedAt.IsZero() {
			endedAt := run.EndedAt
			sr.EndedAt = &endedAt
		}
		out[i] = sr
	}
	return out, nil
}

// sessionFunctionNames returns a lookup from function slug to its
// registered display name, built once from qr.Data.GetFunctions --
// inngest.runs (pkg/duckdb/dashboards' data source) only carries a
// function's slug, not its human-facing Name, so this resolver-level
// join is what fills models.SessionFunction.Name. Falls back to the slug
// itself for an unknown slug (e.g. a function since removed).
func (qr *queryResolver) sessionFunctionNames(ctx context.Context) (func(slug string) string, error) {
	fns, err := qr.Data.GetFunctions(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving function names: %w", err)
	}
	names := make(map[string]string, len(fns))
	for _, fn := range fns {
		names[fn.Slug] = fn.Name
	}
	return func(slug string) string {
		if name, ok := names[slug]; ok {
			return name
		}
		return slug
	}, nil
}

// sessionTimeRange applies dashboards' own default window (7 days,
// matching PR #4530's convention) when the caller specifies no range.
func sessionTimeRange(input *models.TimeRangeInput) (from, until time.Time) {
	if input == nil {
		return time.Time{}, time.Time{}
	}
	until = time.Now()
	if input.Until != nil {
		until = *input.Until
	}
	return input.From, until
}

// sessionRunStatusString renders run's status as one of the dev-server-ui
// shared FunctionRunStatus values (functionRun.ts's functionRunStatuses:
// FAILED/RUNNING/PAUSED/QUEUED/COMPLETED/CANCELLED/SKIPPED/WAITING/
// UNKNOWN) rather than enums.RunStatus's own String() -- that returns
// title-cased names straight off the enum ("Scheduled" for a queued run,
// not "Queued"), which would round-trip through the UI's
// isFunctionRunStatus type guard as an unrecognized value. Mirrors
// pkg/api/v2/endpoints_runs.go's toFunctionRunStatus mapping (same default
// on an unmapped/queued-shaped status).
func sessionRunStatusString(status enums.RunStatus) string {
	switch status {
	case enums.RunStatusCompleted:
		return "COMPLETED"
	case enums.RunStatusFailed:
		return "FAILED"
	case enums.RunStatusCancelled:
		return "CANCELLED"
	case enums.RunStatusRunning:
		return "RUNNING"
	default:
		return "QUEUED"
	}
}
