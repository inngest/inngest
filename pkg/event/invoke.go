package event

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

func NewInvocationEvent(opts NewInvocationEventOpts) BaseTrackedEvent {
	evt := opts.Event
	evt.Name = InvokeFnName

	if evt.Timestamp == 0 {
		evt.Timestamp = time.Now().UnixMilli()
	}
	if evt.Data == nil {
		evt.Data = make(map[string]any)
	}

	internalID := ulid.MustNew(uint64(evt.Timestamp), rand.Reader)
	if evt.ID == "" {
		evt.ID = internalID.String()
	}

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
		DebugSessionID:      opts.DebugSessionID,
		DebugRunID:          opts.DebugRunID,
	}

	return BaseTrackedEvent{
		ID:          internalID,
		Event:       evt,
		WorkspaceID: opts.EnvID,
		AccountID:   opts.AccountID,
	}
}

// InngestMetadata represents metadata for an event that is used to invoke a
// function. Note that this metadata is not present on all functions. For
// accessing an event's correlation ID, prefer using `Event.CorrelationID()`.
type InngestMetadata struct {
	// InvokeType represents the invoke type, eg. "step", "api", and so on.
	// This allows us to differentiate function invocations and handle things
	// specifically.
	InvokeType           string `json:"type,omitempty"`
	InvokeIdempotencyKey string `json:"invoke_idempotency,omitempty"`
	SourceAppID          string `json:"source_app_id"`
	SourceFnID           string `json:"source_fn_id"`
	SourceFnVersion      int    `json:"source_fn_v"`
	InvokeFnID           string `json:"fn_id"`
	InvokeCorrelationId  string `json:"correlation_id,omitempty"`
	// InvokeTraceCarrier is for v1 traces and InvokeSpanRef is for v2
	// Used for linking invoked runIDs to the caller
	InvokeTraceCarrier *itrace.TraceCarrier `json:"tc,omitempty"`
	InvokeSpanRef      *meta.SpanReference  `json:"isr,omitempty"`
	InvokeExpiresAt    int64                `json:"expire"`
	InvokeGroupID      string               `json:"gid"`
	InvokeDisplayName  string               `json:"name"`
	DebugSessionID     *ulid.ULID           `json:"debug_session_id,omitempty"`
	DebugRunID         *ulid.ULID           `json:"debug_run_id,omitempty"`
}

func (m *InngestMetadata) Decode(data any) error {
	switch v := data.(type) {
	case InngestMetadata:
		*m = v
		return nil
	case *InngestMetadata:
		if v != nil {
			*m = *v
		}
		return nil
	}
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

// SetInvokeSpanRef sets InvokeSpanRef on the InngestMetadata stored in this event's data bag.
// Returns false if the event has no inngest metadata (not an invocation event).
func (e *Event) SetInvokeSpanRef(ref *meta.SpanReference) bool {
	if e.Data == nil {
		return false
	}
	// The metadata sits under an `any` slot in Event.Data, so
	// we can't mutate one of its fields in place - we have to type assert it out,
	// modify the copy, and put it back.
	md, ok := e.Data[consts.InngestEventDataPrefix].(InngestMetadata)
	if !ok {
		return false
	}
	md.InvokeSpanRef = ref
	e.Data[consts.InngestEventDataPrefix] = md
	return true
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

type NewInvocationEventOpts struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID

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
	DebugSessionID  *ulid.ULID
	DebugRunID      *ulid.ULID
}
