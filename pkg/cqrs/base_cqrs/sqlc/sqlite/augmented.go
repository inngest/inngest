package sqlc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/util"
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

	if v, ok := e.AccountID.(string); ok {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("error parsing account ID: %w", err)
		}
		evt.AccountID = id
	}
	if v, ok := e.WorkspaceID.(string); ok {
		id, err := uuid.Parse(v)
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
func (r *GetSpansByRunIDRow) GetStartTime() time.Time          { return util.AnyToTime(r.StartTime) }
func (r *GetSpansByRunIDRow) GetEndTime() time.Time            { return util.AnyToTime(r.EndTime) }
func (r *GetSpansByRunIDRow) GetSpanFragments() string {
	v, _ := r.SpanFragments.(string)
	return v
}

func (r *GetSpansByDebugRunIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetSpansByDebugRunIDRow) GetRunID() string                 { return r.RunID }
func (r *GetSpansByDebugRunIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetSpansByDebugRunIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetSpansByDebugRunIDRow) GetStartTime() time.Time          { return util.AnyToTime(r.StartTime) }
func (r *GetSpansByDebugRunIDRow) GetEndTime() time.Time            { return util.AnyToTime(r.EndTime) }
func (r *GetSpansByDebugRunIDRow) GetSpanFragments() string {
	v, _ := r.SpanFragments.(string)
	return v
}

func (r *GetSpansByDebugSessionIDRow) GetTraceID() string               { return r.TraceID }
func (r *GetSpansByDebugSessionIDRow) GetRunID() string                 { return r.RunID }
func (r *GetSpansByDebugSessionIDRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *GetSpansByDebugSessionIDRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *GetSpansByDebugSessionIDRow) GetStartTime() time.Time          { return util.AnyToTime(r.StartTime) }
func (r *GetSpansByDebugSessionIDRow) GetEndTime() time.Time            { return util.AnyToTime(r.EndTime) }
func (r *GetSpansByDebugSessionIDRow) GetSpanFragments() string {
	v, _ := r.SpanFragments.(string)
	return v
}
