package event

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/oklog/ulid/v2"
)

const (
	EventReceivedName  = consts.EventReceivedName
	InternalNamePrefix = consts.InternalNamePrefix
	FnFailedName       = consts.FnFailedName
	FnFinishedName     = consts.FnFinishedName
	FnCancelledName    = consts.FnCancelledName
	// InvokeEventName is the event name used to invoke specific functions via an
	// API.  Note that invoking functions still sends an event in the usual manner.
	InvokeFnName = consts.FnInvokeName
	FnCronName   = consts.FnCronName
)

var (
	startTimestamp = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	endTimestamp   = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
)

// TrackedEvent represents an event created for a specific workspace.
type TrackedEvent interface {
	GetAccountID() uuid.UUID
	GetWorkspaceID() uuid.UUID
	GetInternalID() ulid.ULID
	GetEvent() Event
}

// NewEvent unmarshals a byte slice into a concrete event type.
func NewEvent(data []byte) (*Event, error) {
	evt := &Event{}
	if err := json.Unmarshal(data, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// Event represents an event sent to Inngest.
type Event struct {
	// ID represents the unique ID for this particular event.  If supplied, we should attempt
	// to only ingest this event once.
	ID string `json:"id,omitempty"`

	Name string         `json:"name"`
	Data map[string]any `json:"data"`

	// Timestamp is the time the event occurred, at millisecond precision.
	// If this is not provided, we will insert the current time upon receipt of the event
	Timestamp int64  `json:"ts,omitempty"`
	Version   string `json:"v,omitempty"`

	// User represents user-specific information for the event.
	//
	// Deprecated:  this will be removed in favour of storing everything within data.
	User map[string]any `json:"user,omitempty"`
}

func (e Event) Time() time.Time {
	return time.UnixMilli(e.Timestamp)
}

func (e Event) Map() map[string]any {
	if e.Data == nil {
		e.Data = make(map[string]any)
	}
	if e.User == nil {
		e.User = make(map[string]any)
	}

	data := map[string]any{
		"name": e.Name,
		"data": e.Data,
		"user": e.User,
		"id":   e.ID,
		// We cast to float64 because marshalling and unmarshalling from
		// JSON automatically uses float64 as its type;  JS has no notion
		// of ints.
		"ts": float64(e.Timestamp),
	}

	if e.Version != "" {
		data["v"] = e.Version
	}

	return data
}

func (e Event) Validate(ctx context.Context) error {
	if e.Name == "" {
		return errors.New("event name is empty")
	}

	if e.Timestamp != 0 {
		// Convert milliseconds to nanosecond precision
		t := time.Unix(0, e.Timestamp*1_000_000)
		if t.Before(startTimestamp) {
			return errors.New("timestamp is before Jan 1, 1980")
		}
		if t.After(endTimestamp) {
			return errors.New("timestamp is after Jan 1, 2100")
		}
	}

	return nil
}

// CorrelationID returns the correlation ID for the event.
func (e Event) CorrelationID() string {
	if e.Name == InvokeFnName {
		if metadata, err := e.InngestMetadata(); err == nil {
			return metadata.InvokeCorrelationId
		}
	}

	if e.IsFinishedEvent() {
		if corrId, ok := e.Data[consts.InvokeCorrelationId].(string); ok {
			return corrId
		}
	}

	return ""
}

func (e Event) IsInternal() bool {
	return strings.HasPrefix(e.Name, InternalNamePrefix)
}

// IsFinishedEvent returns true if the event is a function finished event.
func (e Event) IsFinishedEvent() bool {
	return e.Name == FnFinishedName
}

func (e Event) IsInvokeEvent() bool {
	return e.Name == InvokeFnName
}

func (e Event) IsCron() bool {
	return IsCron(e.Name)
}

func (e Event) CronSchedule() *string {
	if !IsCron(e.Name) {
		return nil
	}
	return CronSchedule(e.Data)
}

// InternalEvent is a representation of an [Event] that was received by the API
// and annotated with additional metadata.
type InternalEvent struct {
	// ID is the internal ID for the event.
	ID ulid.ULID `json:"internal_id"`
	// AccountID is the account ID for the event.
	AccountID uuid.UUID `json:"account_id"`
	// WorkspaceID is the ID of the environment that the event belongs to
	WorkspaceID uuid.UUID `json:"workspace_id"`
	// Event is the underlying event received.
	Event Event `json:"event"`
}

func (i InternalEvent) GetAccountID() uuid.UUID {
	return i.AccountID
}

func (i InternalEvent) GetWorkspaceID() uuid.UUID {
	return i.WorkspaceID
}

func (i InternalEvent) GetInternalID() ulid.ULID {
	return i.ID
}

func (i InternalEvent) GetEvent() Event {
	return i.Event
}

func IsCron(evtName string) bool {
	return evtName == FnCronName
}

func CronSchedule(evtData map[string]any) *string {
	if cron, ok := evtData["cron"].(string); ok && cron != "" {
		return &cron
	}
	return nil
}
