package db

import (
	"database/sql"
	"strings"
)

// normalizedSpan accessor methods on SpanRow for use in manager span processing.

func (r *SpanRow) GetTraceID() string               { return r.TraceID }
func (r *SpanRow) GetRunID() string                 { return r.RunID }
func (r *SpanRow) GetDynamicSpanID() sql.NullString { return r.DynamicSpanID }
func (r *SpanRow) GetParentSpanID() sql.NullString  { return r.ParentSpanID }
func (r *SpanRow) GetStartTime() interface{}        { return r.StartTime }
func (r *SpanRow) GetEndTime() interface{}          { return r.EndTime }
func (r *SpanRow) GetSpanFragments() any            { return r.SpanFragments }

// EventIDs parses the comma-separated trigger IDs from a TraceRun.
func (tr *TraceRun) EventIDs() []string {
	if len(tr.TriggerIds) == 0 {
		return []string{}
	}
	return strings.Split(string(tr.TriggerIds), ",")
}

// HasEventIDs checks if the trace run includes any of the provided event IDs.
func (tr *TraceRun) HasEventIDs(ids []string) bool {
	idmap := map[string]bool{}
	for _, id := range ids {
		idmap[id] = true
	}
	for _, rid := range tr.EventIDs() {
		if idmap[rid] {
			return true
		}
	}
	return false
}
