package manager

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/inngest/inngest/pkg/util/ttlupsert"
	"github.com/oklog/ulid/v2"
)

const sessionAttrKey = "_inngest.event.sessions"

func newSessionKeyUpserter() ttlupsert.Upserter[cqrs.SessionKeyRecord] {
	return ttlupsert.NewWithKey(func(record cqrs.SessionKeyRecord) string {
		return record.WorkspaceID.String() + ":" + record.Key
	})
}

func (w wrapper) RecordSessionKeys(ctx context.Context, workspaceID uuid.UUID, eventSessions event.Sessions) error {
	if workspaceID == uuid.Nil || len(eventSessions) == 0 {
		return nil
	}

	upserts := w.sessionKeyUpserts
	if upserts == nil {
		upserts = newSessionKeyUpserter()
	}

	keys := uniqueSessionKeys(eventSessions)
	var resultErr error
	for _, key := range keys {
		record := cqrs.SessionKeyRecord{WorkspaceID: workspaceID, Key: key}
		_, err := upserts.Upsert(ctx, record, func(ctx context.Context) error {
			return w.recordSessionKey(ctx, record)
		})
		if err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	return resultErr
}

func (w wrapper) recordSessionKey(ctx context.Context, record cqrs.SessionKeyRecord) error {
	if record.WorkspaceID == uuid.Nil || record.Key == "" {
		return nil
	}

	sqlQuery, args, err := sq.Dialect(w.dialect()).
		Insert("session_keys").
		Rows(sq.Record{
			"workspace_id": record.WorkspaceID.String(),
			"session_key":  record.Key,
		}).
		OnConflict(sq.DoNothing()).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build record session key query: %w", err)
	}

	if _, err := w.adapter.Conn().ExecContext(ctx, sqlQuery, args...); err != nil {
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

	rows, err := w.adapter.Conn().QueryContext(ctx, sqlQuery, args...)
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
	runs, err := w.sessionRuns(ctx, workspaceID, tr, cqrs.SessionsLimit*20)
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
	runs, err := w.sessionRuns(ctx, workspaceID, tr, cqrs.SessionRunsLimit*10)
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

func (w wrapper) sessionRuns(ctx context.Context, workspaceID uuid.UUID, tr cqrs.SessionTimeRange, limit int) ([]storedSessionRun, error) {
	if workspaceID == uuid.Nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = cqrs.SessionRunsLimit
	}
	if tr.Until.IsZero() {
		tr.Until = time.Now()
	}

	traceRuns, err := w.sessionTraceRuns(ctx, workspaceID, tr, limit)
	if err != nil {
		return nil, err
	}

	sqlQuery, args, err := sq.Dialect(w.dialect()).
		From("spans").
		Select("run_id", "start_time", "end_time", "status", "attributes").
		Where(
			sq.C("env_id").Eq(workspaceID.String()),
			sq.C("name").Eq(meta.SpanNameRun),
			sq.C("debug_run_id").IsNull(),
		).
		Order(sq.C("start_time").Desc()).
		Limit(uint(limit)).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build session runs query: %w", err)
	}

	rows, err := w.adapter.Conn().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("get session runs: %w", err)
	}
	defer rows.Close()

	out := []storedSessionRun{}
	for rows.Next() {
		var runIDRaw string
		var startTimeRaw, endTimeRaw any
		var spanStatus sql.NullString
		var attrsRaw []byte
		if err := rows.Scan(&runIDRaw, &startTimeRaw, &endTimeRaw, &spanStatus, &attrsRaw); err != nil {
			return nil, fmt.Errorf("scan session run: %w", err)
		}

		sessions, functionSlug, functionName, eventName := parseSessionRunAttrs(attrsRaw)
		if len(sessions) == 0 {
			continue
		}
		runID, err := ulid.Parse(runIDRaw)
		if err != nil {
			continue
		}
		spanStart := parseDBTime(startTimeRaw)
		spanEnd := parseDBTime(endTimeRaw)
		traceRun, ok := traceRuns[runID.String()]
		if !ok {
			if !tr.From.IsZero() && spanStart.Before(tr.From) {
				continue
			}
			if spanStart.After(tr.Until) {
				continue
			}
			var spanStatusValue *string
			if spanStatus.Valid {
				spanStatusValue = &spanStatus.String
			}
			status, _ := runStatusFromSpanStatus(spanStatusValue)
			traceRun = sessionTraceRun{
				QueuedAtMS:  spanStart.UnixMilli(),
				StartedAtMS: spanStart.UnixMilli(),
				EndedAtMS:   spanEnd.UnixMilli(),
				Status:      status,
			}
		}

		out = append(out, storedSessionRun{
			RunID:        runID,
			QueuedAt:     time.UnixMilli(traceRun.QueuedAtMS),
			StartedAtMS:  traceRun.StartedAtMS,
			EndedAtMS:    traceRun.EndedAtMS,
			Status:       traceRun.Status,
			FunctionSlug: functionSlug,
			FunctionName: functionName,
			EventName:    eventName,
			Sessions:     sessions,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session runs: %w", err)
	}
	return out, nil
}

func (w wrapper) sessionTraceRuns(ctx context.Context, workspaceID uuid.UUID, tr cqrs.SessionTimeRange, limit int) (map[string]sessionTraceRun, error) {
	sqlQuery, args, err := sq.Dialect(w.dialect()).
		From("trace_runs").
		Select("run_id", "queued_at", "started_at", "ended_at", "status").
		Where(
			sq.C("workspace_id").Eq(workspaceID.String()),
			sq.C("queued_at").Gte(tr.From.UnixMilli()),
			sq.C("queued_at").Lte(tr.Until.UnixMilli()),
		).
		Order(sq.C("queued_at").Desc()).
		Limit(uint(limit)).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build session trace runs query: %w", err)
	}

	rows, err := w.adapter.Conn().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("get session trace runs: %w", err)
	}
	defer rows.Close()

	out := map[string]sessionTraceRun{}
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
		out[runID] = sessionTraceRun{
			QueuedAtMS:  queuedAtMS,
			StartedAtMS: startedAtMS,
			EndedAtMS:   endedAtMS,
			Status:      traceRunStatusFromDB(int64(statusCode)),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session trace runs: %w", err)
	}
	return out, nil
}

func parseSessionRunAttrs(raw []byte) (meta.EventSessions, string, string, string) {
	attrs := map[string]any{}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, "", "", ""
	}
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, "", "", ""
	}

	sessionsRaw, _ := attrs[sessionAttrKey].(string)
	if sessionsRaw == "" {
		return nil, attrString(attrs, "_inngest.function.slug"), attrString(attrs, "_inngest.function.name"), attrString(attrs, "_inngest.event.trigger.name")
	}

	sessions := meta.EventSessions{}
	if err := json.Unmarshal([]byte(sessionsRaw), &sessions); err != nil {
		return nil, "", "", ""
	}
	return sessions, attrString(attrs, "_inngest.function.slug"), attrString(attrs, "_inngest.function.name"), attrString(attrs, "_inngest.event.trigger.name")
}

func attrString(attrs map[string]any, key string) string {
	if val, ok := attrs[key].(string); ok {
		return val
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

func parseDBTime(raw any) time.Time {
	switch val := raw.(type) {
	case time.Time:
		return val
	case string:
		return parseDBTimeString(val)
	case []byte:
		return parseDBTimeString(string(val))
	default:
		return time.Time{}
	}
}

func parseDBTimeString(raw string) time.Time {
	raw = strings.Split(raw, " m=")[0]
	layouts := []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
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
