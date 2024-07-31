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

	// TODO: Event data
	if err := json.Unmarshal([]byte(e.EventData), &evt.EventData); err != nil {
		return nil, fmt.Errorf("error parsing event data: %w", err)
	}

	// TODO: Event user
	if err := json.Unmarshal([]byte(e.EventUser), &evt.EventUser); err != nil {
		return nil, fmt.Errorf("error parsing event user data: %w", err)
	}

	if e.EventV.Valid {
		evt.EventVersion = e.EventV.String
	}

	return evt, nil
}
