package sqlc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
)

func (tr *TraceRun) EventIDs() []string {
	if len(tr.TriggerIds) == 0 {
		return []string{}
	}

	return strings.Split(string(tr.TriggerIds), ",")
}

// HasEventIDs checks if the run include any of the provided event IDs
func (tr *TraceRun) HasEventIDs(ids []string) bool {
	// map out IDs for quick look up
	idmap := map[string]bool{}
	for _, id := range ids {
		idmap[id] = true
	}

	for _, rid := range tr.EventIDs() {
		if _, ok := idmap[string(rid)]; ok {
			return true
		}
	}

	return false
}

// --- Event

func (e *Event) ToCQRS() (*cqrs.Event, error) {
	evt := &cqrs.Event{
		ID:         e.InternalID,
		ReceivedAt: e.ReceivedAt,
		EventID:    e.EventID,
		EventName:  e.EventName,
		EventTS:    e.EventTs.UnixMilli(),
	}

	if e.AccountID.Valid {
		id, err := uuid.Parse(e.AccountID.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing account ID: %w", err)
		}
		evt.AccountID = id
	}
	if e.WorkspaceID.Valid {
		id, err := uuid.Parse(e.WorkspaceID.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing environment ID: %w", err)
		}
		evt.WorkspaceID = id
	}

	// Event data
	if err := json.Unmarshal([]byte(e.EventData), &evt.EventData); err != nil {
		return nil, fmt.Errorf("error parsing event data: %w", err)
	}

	// Event user
	if err := json.Unmarshal([]byte(e.EventUser), &evt.EventUser); err != nil {
		return nil, fmt.Errorf("error parsing event user data: %w", err)
	}

	if e.EventV.Valid {
		evt.EventVersion = e.EventV.String
	}

	return evt, nil
}

// Interface methods for normalizedSpan
func (r *GetSpansByRunIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetSpansByRunIDRow) GetRunID() string                 { return r.RunID }
func (r *GetSpansByRunIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetSpansByRunIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetSpansByRunIDRow) GetStartTime() interface{}        { return r.StartTime }
func (r *GetSpansByRunIDRow) GetEndTime() interface{}          { return r.EndTime }
func (r *GetSpansByRunIDRow) GetSpanFragments() any            { return r.SpanFragments }

func (r *GetSpansByDebugRunIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetSpansByDebugRunIDRow) GetRunID() string                 { return r.RunID }
func (r *GetSpansByDebugRunIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetSpansByDebugRunIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetSpansByDebugRunIDRow) GetStartTime() interface{}        { return r.StartTime }
func (r *GetSpansByDebugRunIDRow) GetEndTime() interface{}          { return r.EndTime }
func (r *GetSpansByDebugRunIDRow) GetSpanFragments() any            { return r.SpanFragments }

func (r *GetSpansByDebugSessionIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetSpansByDebugSessionIDRow) GetRunID() string                 { return r.RunID }
func (r *GetSpansByDebugSessionIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetSpansByDebugSessionIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetSpansByDebugSessionIDRow) GetStartTime() interface{}        { return r.StartTime }
func (r *GetSpansByDebugSessionIDRow) GetEndTime() interface{}          { return r.EndTime }
func (r *GetSpansByDebugSessionIDRow) GetSpanFragments() any            { return r.SpanFragments }

func (r *GetRunSpanByRunIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetRunSpanByRunIDRow) GetRunID() string                 { return r.RunID }
func (r *GetRunSpanByRunIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetRunSpanByRunIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetRunSpanByRunIDRow) GetStartTime() any                { return r.StartTime }
func (r *GetRunSpanByRunIDRow) GetEndTime() any                  { return r.EndTime }

func (r *GetRunSpanByRunIDRow) GetSpanFragments() any { return r.SpanFragments }

func (r *GetStepSpanByStepIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetStepSpanByStepIDRow) GetRunID() string                 { return r.RunID }
func (r *GetStepSpanByStepIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetStepSpanByStepIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetStepSpanByStepIDRow) GetStartTime() any                { return r.StartTime }
func (r *GetStepSpanByStepIDRow) GetEndTime() any                  { return r.EndTime }

func (r *GetStepSpanByStepIDRow) GetSpanFragments() any { return r.SpanFragments }

func (r *GetExecutionSpanByStepIDAndAttemptRow) GetTraceID() string { return r.TraceID }
func (r *GetExecutionSpanByStepIDAndAttemptRow) GetRunID() string   { return r.RunID }
func (r *GetExecutionSpanByStepIDAndAttemptRow) GetDynamicSpanID() sql.NullString {
	return r.DynamicSpanID
}
func (r *GetExecutionSpanByStepIDAndAttemptRow) GetParentSpanID() sql.NullString {
	return r.ParentSpanID
}
func (r *GetExecutionSpanByStepIDAndAttemptRow) GetStartTime() any { return r.StartTime }
func (r *GetExecutionSpanByStepIDAndAttemptRow) GetEndTime() any   { return r.EndTime }

func (r *GetExecutionSpanByStepIDAndAttemptRow) GetSpanFragments() any { return r.SpanFragments }

func (r *GetLatestExecutionSpanByStepIDRow) GetTraceID() string { return r.TraceID }
func (r *GetLatestExecutionSpanByStepIDRow) GetRunID() string   { return r.RunID }
func (r *GetLatestExecutionSpanByStepIDRow) GetDynamicSpanID() sql.NullString {
	return r.DynamicSpanID
}
func (r *GetLatestExecutionSpanByStepIDRow) GetParentSpanID() sql.NullString {
	return r.ParentSpanID
}
func (r *GetLatestExecutionSpanByStepIDRow) GetStartTime() any { return r.StartTime }
func (r *GetLatestExecutionSpanByStepIDRow) GetEndTime() any   { return r.EndTime }

func (r *GetLatestExecutionSpanByStepIDRow) GetSpanFragments() any { return r.SpanFragments }

func (r *GetSpanBySpanIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetSpanBySpanIDRow) GetRunID() string                 { return r.RunID }
func (r *GetSpanBySpanIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetSpanBySpanIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetSpanBySpanIDRow) GetStartTime() any                { return r.StartTime }
func (r *GetSpanBySpanIDRow) GetEndTime() any                  { return r.EndTime }

func (r *GetSpanBySpanIDRow) GetSpanFragments() any { return r.SpanFragments }
