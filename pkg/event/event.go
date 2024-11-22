package event

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
)

const (
	EventReceivedName  = "event/event.received"
	InternalNamePrefix = "inngest/"
	FnFailedName       = InternalNamePrefix + "function.failed"
	FnFinishedName     = InternalNamePrefix + "function.finished"
	FnCancelledName    = InternalNamePrefix + "function.cancelled"
	// InvokeEventName is the event name used to invoke specific functions via an
	// API.  Note that invoking functions still sends an event in the usual manner.
	InvokeFnName = InternalNamePrefix + "function.invoked"
	FnCronName   = InternalNamePrefix + "scheduled.timer"
)

var (
	startTimestamp = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	endTimestamp   = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
)

type TrackedEvent interface {
	GetWorkspaceID() uuid.UUID
	GetInternalID() ulid.ULID
	GetEvent() Event
}

func NewEvent(data []byte) (*Event, error) {
	evt := &Event{}
	if err := json.Unmarshal(data, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// Event represents an event sent to Inngest.
type Event struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`

	// User represents user-specific information for the event.
	User map[string]any `json:"user,omitempty"`

	// ID represents the unique ID for this particular event.  If supplied, we should attempt
	// to only ingest this event once.
	ID string `json:"id,omitempty"`

	// Timestamp is the time the event occurred, at millisecond precision.
	// If this is not provided, we will insert the current time upon receipt of the event
	Timestamp int64  `json:"ts,omitempty"`
	Version   string `json:"v,omitempty"`
}

func (evt Event) Time() time.Time {
	return time.UnixMilli(evt.Timestamp)
}

func (evt Event) Map() map[string]any {
	if evt.Data == nil {
		evt.Data = make(map[string]any)
	}
	if evt.User == nil {
		evt.User = make(map[string]any)
	}

	data := map[string]any{
		"name": evt.Name,
		"data": evt.Data,
		"user": evt.User,
		"id":   evt.ID,
		// We cast to float64 because marshalling and unmarshalling from
		// JSON automatically uses float64 as its type;  JS has no notion
		// of ints.
		"ts": float64(evt.Timestamp),
	}

	if evt.Version != "" {
		data["v"] = evt.Version
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

// InngestMetadata represents metadata for an event that is used to invoke a
// function. Note that this metadata is not present on all functions. For
// accessing an event's correlation ID, prefer using `Event.CorrelationID()`.
type InngestMetadata struct {
	SourceAppID         string               `json:"source_app_id"`
	SourceFnID          string               `json:"source_fn_id"`
	SourceFnVersion     int                  `json:"source_fn_v"`
	InvokeFnID          string               `json:"fn_id"`
	InvokeCorrelationId string               `json:"correlation_id,omitempty"`
	InvokeTraceCarrier  *itrace.TraceCarrier `json:"tc,omitempty"`
	InvokeExpiresAt     int64                `json:"expire"`
	InvokeGroupID       string               `json:"gid"`
	InvokeDisplayName   string               `json:"name"`
}

func (m *InngestMetadata) Decode(data any) error {
	byt, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, m)
}

func (m *InngestMetadata) RunID() *ulid.ULID {
	if len(m.InvokeCorrelationId) == 0 {
		return nil
	}
	s := strings.Split(m.InvokeCorrelationId, ".")
	if len(s) != 2 {
		return nil
	}

	if id, err := ulid.Parse(s[0]); err == nil {
		return &id
	}
	return nil
}

func (e Event) InngestMetadata() (*InngestMetadata, error) {
	raw, ok := e.Data[consts.InngestEventDataPrefix]
	if !ok {
		return nil, fmt.Errorf("no data found in prefix '%s'", consts.InngestEventDataPrefix)
	}

	switch v := raw.(type) {
	case InngestMetadata:
		return &v, nil

	default:
		var metadata InngestMetadata
		jsonData, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(jsonData, &metadata); err != nil {
			return nil, err
		}
		return &metadata, nil
	}
}

func NewOSSTrackedEvent(e Event) TrackedEvent {
	// Never use e.ID as the internal ID, since it's specified by the sender
	internalID := ulid.MustNew(ulid.Now(), rand.Reader)
	if e.ID == "" {
		e.ID = internalID.String()
	}
	return ossTrackedEvent{
		Id:    internalID,
		Event: e,
	}
}

func NewOSSTrackedEventWithID(e Event, id ulid.ULID) TrackedEvent {
	return ossTrackedEvent{
		Id:    id,
		Event: e,
	}
}

func NewOSSTrackedEventFromString(data string) (*ossTrackedEvent, error) {
	evt := &ossTrackedEvent{}
	if err := json.Unmarshal([]byte(data), evt); err != nil {
		return nil, err
	}

	return evt, nil
}

type ossTrackedEvent struct {
	Id    ulid.ULID `json:"internal_id"`
	Event Event     `json:"event"`
}

func (o ossTrackedEvent) GetEvent() Event {
	return o.Event
}

func (o ossTrackedEvent) GetInternalID() ulid.ULID {
	return o.Id
}

func (o ossTrackedEvent) GetWorkspaceID() uuid.UUID {
	// There are no workspaces in OSS yet.
	return consts.DevServerEnvId
}

type NewInvocationEventOpts struct {
	SourceAppID     string
	SourceFnID      string
	SourceFnVersion int
	Event           Event
	FnID            string
	CorrelationID   *string
	TraceCarrier    *itrace.TraceCarrier
	ExpiresAt       int64
	GroupID         string
	DisplayName     string
}

func NewInvocationEvent(opts NewInvocationEventOpts) Event {
	evt := opts.Event

	if evt.Timestamp == 0 {
		evt.Timestamp = time.Now().UnixMilli()
	}
	if evt.ID == "" {
		evt.ID = ulid.MustNew(uint64(evt.Timestamp), rand.Reader).String()
	}
	if evt.Data == nil {
		evt.Data = make(map[string]interface{})
	}
	evt.Name = InvokeFnName

	correlationID := ""
	if opts.CorrelationID != nil {
		correlationID = *opts.CorrelationID
	}

	evt.Data[consts.InngestEventDataPrefix] = InngestMetadata{
		InvokeFnID:          opts.FnID,
		InvokeCorrelationId: correlationID,
		InvokeTraceCarrier:  opts.TraceCarrier,
		InvokeExpiresAt:     opts.ExpiresAt,
		InvokeGroupID:       opts.GroupID,
		InvokeDisplayName:   opts.DisplayName,
		SourceAppID:         opts.SourceAppID,
		SourceFnID:          opts.SourceFnID,
		SourceFnVersion:     opts.SourceFnVersion,
	}

	return evt
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

func IsCron(evtName string) bool {
	return evtName == FnCronName
}

func CronSchedule(evtData map[string]any) *string {
	if cron, ok := evtData["cron"].(string); ok && cron != "" {
		return &cron
	}
	return nil
}
