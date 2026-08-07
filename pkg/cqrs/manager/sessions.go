package manager

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

const sessionRunScanLimit = 2000

func (w wrapper) RecordSessionKeys(ctx context.Context, workspaceID uuid.UUID, eventSessions event.Sessions) error {
	if workspaceID == uuid.Nil || len(eventSessions) == 0 {
		return nil
	}

	keys := uniqueSessionKeys(eventSessions)
	var resultErr error
	for _, key := range keys {
		err := w.recordSessionKey(ctx, workspaceID, key)
		if err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	return resultErr
}

func (w wrapper) recordSessionKey(ctx context.Context, workspaceID uuid.UUID, key string) error {
	if workspaceID == uuid.Nil || key == "" {
		return nil
	}

	sqlQuery, args, err := sq.Dialect(w.dialect()).
		Insert("session_keys").
		Rows(sq.Record{
			"workspace_id": workspaceID.String(),
			"session_key":  key,
		}).
		OnConflict(sq.DoNothing()).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build record session key query: %w", err)
	}

	if _, err := w.adapter.RawDB().ExecContext(ctx, sqlQuery, args...); err != nil {
		return fmt.Errorf("record session key: %w", err)
	}
	return nil
}

func (w wrapper) GetSessionKeys(ctx context.Context, workspaceID uuid.UUID, search string) ([]*cqrs.SessionKey, error) {
	if workspaceID == uuid.Nil {
		return nil, nil
	}

	search = strings.TrimSpace(search)
	where := []sq.Expression{sq.C("workspace_id").Eq(workspaceID.String())}
	if search != "" {
		where = append(where, sq.L("LOWER(session_key) LIKE LOWER(?) ESCAPE '\\'", "%"+escapeLikeSearch(search)+"%"))
	}

	sqlQuery, args, err := sq.Dialect(w.dialect()).
		From("session_keys").
		Select("session_key", "created_at").
		Where(where...).
		Order(sq.C("created_at").Desc()).
		Limit(uint(cqrs.SessionKeysLimit)).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build get session keys query: %w", err)
	}

	rows, err := w.adapter.RawDB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("get session keys: %w", err)
	}
	defer rows.Close()

	out := []*cqrs.SessionKey{}
	for rows.Next() {
		var key string
		var createdAt time.Time
		if err := rows.Scan(&key, &createdAt); err != nil {
			return nil, fmt.Errorf("scan session key: %w", err)
		}
		out = append(out, &cqrs.SessionKey{SessionKey: key, CreatedAt: createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session keys: %w", err)
	}
	return out, nil
}

func (w wrapper) GetSessions(ctx context.Context, workspaceID uuid.UUID, sessionKey string, sessionIDSearch string, tr cqrs.SessionTimeRange) ([]*cqrs.SessionGroup, error) {
	runs, err := w.sessionRuns(ctx, workspaceID, tr)
	if err != nil {
		return nil, err
	}

	sessionIDSearch = strings.ToLower(strings.TrimSpace(sessionIDSearch))
	groups := map[string]*cqrs.SessionGroup{}
	for _, run := range runs {
		for _, pair := range run.Sessions {
			if pair.Key != sessionKey {
				continue
			}
			if sessionIDSearch != "" && !strings.Contains(strings.ToLower(pair.ID), sessionIDSearch) {
				continue
			}

			group := groups[pair.ID]
			if group == nil {
				group = &cqrs.SessionGroup{
					SessionKey:   sessionKey,
					SessionID:    pair.ID,
					LastActiveAt: run.QueuedAt,
				}
				groups[pair.ID] = group
			}

			group.RunCount++
			if run.Status == enums.RunStatusFailed {
				group.FailedRunCount++
			}
			if run.QueuedAt.After(group.LastActiveAt) {
				group.LastActiveAt = run.QueuedAt
			}
			addSessionFunction(group, run.FunctionSlug, run.FunctionName)
		}
	}

	out := make([]*cqrs.SessionGroup, 0, len(groups))
	for _, group := range groups {
		if group.RunCount > 0 {
			group.FailureRate = float64(group.FailedRunCount) / float64(group.RunCount)
		}
		out = append(out, group)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastActiveAt.After(out[j].LastActiveAt)
	})
	if len(out) > cqrs.SessionsLimit {
		out = out[:cqrs.SessionsLimit]
	}
	return out, nil
}

func (w wrapper) GetSessionRuns(ctx context.Context, workspaceID uuid.UUID, sessionKey string, sessionID string, tr cqrs.SessionTimeRange) ([]*cqrs.SessionRun, error) {
	runs, err := w.sessionRuns(ctx, workspaceID, tr)
	if err != nil {
		return nil, err
	}

	out := make([]*cqrs.SessionRun, 0, len(runs))
	for _, run := range runs {
		if !run.hasSession(sessionKey, sessionID) {
			continue
		}

		var eventName *string
		if run.EventName != "" {
			eventName = &run.EventName
		}
		out = append(out, &cqrs.SessionRun{
			ID:           run.RunID,
			FunctionSlug: run.FunctionSlug,
			EventName:    eventName,
			Status:       run.Status,
			QueuedAt:     run.QueuedAt,
			StartedAt:    nullableUnixMilli(run.StartedAtMS),
			EndedAt:      nullableUnixMilli(run.EndedAtMS),
		})
		if len(out) >= cqrs.SessionRunsLimit {
			break
		}
	}
	return out, nil
}

type storedSessionRun struct {
	RunID        ulid.ULID
	QueuedAt     time.Time
	StartedAtMS  int64
	EndedAtMS    int64
	Status       enums.RunStatus
	FunctionSlug string
	FunctionName string
	EventName    string
	Sessions     meta.EventSessions
}

type sessionTraceRun struct {
	RunID       ulid.ULID
	QueuedAtMS  int64
	StartedAtMS int64
	EndedAtMS   int64
	Status      enums.RunStatus
}

func (r storedSessionRun) hasSession(sessionKey string, sessionID string) bool {
	for _, pair := range r.Sessions {
		if pair.Key == sessionKey && pair.ID == sessionID {
			return true
		}
	}
	return false
}

func (w wrapper) sessionRuns(ctx context.Context, workspaceID uuid.UUID, tr cqrs.SessionTimeRange) ([]storedSessionRun, error) {
	if workspaceID == uuid.Nil {
		return nil, nil
	}
	if tr.Until.IsZero() {
		tr.Until = time.Now()
	}

	traceRuns, err := w.sessionTraceRuns(ctx, workspaceID, tr)
	if err != nil {
		return nil, err
	}
	if len(traceRuns) == 0 {
		return []storedSessionRun{}, nil
	}

	runIDs := make([]ulid.ULID, len(traceRuns))
	for i, run := range traceRuns {
		runIDs[i] = run.RunID
	}
	spansByRunID, err := w.GetSpansByRunIDsAndName(ctx, runIDs, meta.SpanNameRun)
	if err != nil {
		return nil, fmt.Errorf("get session run spans: %w", err)
	}

	out := make([]storedSessionRun, 0, len(traceRuns))
	for _, traceRun := range traceRuns {
		for _, span := range spansByRunID[traceRun.RunID] {
			attrs := span.Attributes
			if attrs == nil || attrs.Sessions == nil || len(*attrs.Sessions) == 0 {
				continue
			}

			out = append(out, storedSessionRun{
				RunID:        traceRun.RunID,
				QueuedAt:     time.UnixMilli(traceRun.QueuedAtMS),
				StartedAtMS:  traceRun.StartedAtMS,
				EndedAtMS:    traceRun.EndedAtMS,
				Status:       traceRun.Status,
				FunctionSlug: stringValue(attrs.FunctionSlug),
				FunctionName: stringValue(attrs.FunctionName),
				EventName:    stringValue(attrs.TriggeringEventName),
				Sessions:     *attrs.Sessions,
			})
			break
		}
	}
	return out, nil
}

func (w wrapper) sessionTraceRuns(ctx context.Context, workspaceID uuid.UUID, tr cqrs.SessionTimeRange) ([]sessionTraceRun, error) {
	sqlQuery, args, err := sq.Dialect(w.dialect()).
		From("trace_runs").
		Select("run_id", "queued_at", "started_at", "ended_at", "status").
		Where(
			sq.C("workspace_id").Eq(workspaceID.String()),
			sq.C("queued_at").Gte(tr.From.UnixMilli()),
			sq.C("queued_at").Lte(tr.Until.UnixMilli()),
		).
		Order(sq.C("queued_at").Desc(), sq.C("run_id").Desc()).
		Limit(sessionRunScanLimit).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build session trace runs query: %w", err)
	}

	rows, err := w.adapter.RawDB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("get session trace runs: %w", err)
	}
	defer rows.Close()

	out := make([]sessionTraceRun, 0, sessionRunScanLimit)
	for rows.Next() {
		var runIDRaw any
		var queuedAtMS, startedAtMS, endedAtMS int64
		var statusCode int
		if err := rows.Scan(&runIDRaw, &queuedAtMS, &startedAtMS, &endedAtMS, &statusCode); err != nil {
			return nil, fmt.Errorf("scan session trace run: %w", err)
		}
		runID, ok := canonicalRunID(runIDRaw)
		if !ok {
			continue
		}
		out = append(out, sessionTraceRun{
			RunID:       ulid.MustParse(runID),
			QueuedAtMS:  queuedAtMS,
			StartedAtMS: startedAtMS,
			EndedAtMS:   endedAtMS,
			Status:      traceRunStatusFromDB(int64(statusCode)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session trace runs: %w", err)
	}
	return out, nil
}

func stringValue(value *string) string {
	if value != nil {
		return *value
	}
	return ""
}

func nullableUnixMilli(ms int64) *time.Time {
	if ms <= 0 {
		return nil
	}
	t := time.UnixMilli(ms)
	return &t
}

func canonicalRunID(raw any) (string, bool) {
	switch val := raw.(type) {
	case string:
		return canonicalRunIDBytes([]byte(val))
	case []byte:
		return canonicalRunIDBytes(val)
	default:
		return "", false
	}
}

func canonicalRunIDBytes(raw []byte) (string, bool) {
	if len(raw) == len(ulid.ULID{}) {
		var id ulid.ULID
		copy(id[:], raw)
		return id.String(), true
	}
	id, err := ulid.Parse(string(raw))
	if err != nil {
		return "", false
	}
	return id.String(), true
}

func uniqueSessionKeys(eventSessions event.Sessions) []string {
	keys := make([]string, 0, len(eventSessions))
	seen := map[string]struct{}{}
	for key := range eventSessions {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func addSessionFunction(group *cqrs.SessionGroup, slug string, name string) {
	if slug == "" {
		return
	}
	for _, fn := range group.Functions {
		if fn.Slug == slug {
			return
		}
	}
	group.Functions = append(group.Functions, cqrs.SessionFunction{Slug: slug, Name: name})
}

func escapeLikeSearch(search string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(search)
}
