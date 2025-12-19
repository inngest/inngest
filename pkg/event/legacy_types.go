package event

import (
	"crypto/rand"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/oklog/ulid/v2"
)

func NewBaseTrackedEvent(e Event, seed *SeededID) TrackedEvent {
	// Never use e.ID as the internal ID, since it's specified by the sender
	internalID := ulid.MustNew(ulid.Now(), rand.Reader)

	if seed != nil {
		newInternalID, err := seed.ToULID()
		if err == nil {
			// IMPORTANT: This means it's possible for duplicate internal IDs in
			// the event store. This is not ideal but it's the best we can do
			// until we add first-class event idempotency (it's currently
			// enforced when scheduling runs).
			internalID = newInternalID
		}
	}

	if e.ID == "" {
		e.ID = internalID.String()
	}
	return BaseTrackedEvent{
		ID:    internalID,
		Event: e,
	}
}

func NewBaseTrackedEventWithID(e Event, id ulid.ULID) TrackedEvent {
	return BaseTrackedEvent{
		ID:    id,
		Event: e,
	}
}

func NewBaseTrackedEventFromString(data string) (*BaseTrackedEvent, error) {
	evt := &BaseTrackedEvent{}
	if err := json.Unmarshal([]byte(data), evt); err != nil {
		return nil, err
	}

	return evt, nil
}

type BaseTrackedEvent struct {
	ID          ulid.ULID `json:"internal_id"`
	AccountID   uuid.UUID `json:"account_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Event       Event     `json:"event"`
}

func (o BaseTrackedEvent) GetEvent() Event {
	return o.Event
}

func (o BaseTrackedEvent) GetInternalID() ulid.ULID {
	return o.ID
}

func (o BaseTrackedEvent) GetAccountID() uuid.UUID {
	// There are no accounts in OSS yet.
	return consts.DevServerAccountID
}

func (o BaseTrackedEvent) GetWorkspaceID() uuid.UUID {
	// There are no workspaces in OSS yet.
	return consts.DevServerEnvID
}
