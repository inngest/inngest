package sqlc

import (
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
