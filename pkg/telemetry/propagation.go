package telemetry

import (
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type TraceCarrierOpt func(tc *TraceCarrier)

// TraceCarrier stores the data that needs to be carried through systems.
// e.g. pubsub, queues, etc
type TraceCarrier struct {
	Context   map[string]string `json:"ctx,omitempty"`
	Timestamp time.Time         `json:"ts"`
	SpanID    *trace.SpanID     `json:"sid,omitempty"`
}

func NewTraceCarrier(opts ...TraceCarrierOpt) *TraceCarrier {
	carrier := &TraceCarrier{
		Context: map[string]string{},
	}

	for _, opt := range opts {
		opt(carrier)
	}

	return carrier
}

func (tc *TraceCarrier) Unmarshal(data any) error {
	byt, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, tc)
}

func (tc *TraceCarrier) IsNonZero() bool {
	return tc.Context != nil && tc.Timestamp.UnixMilli() > 0 && tc.SpanID != nil
}

func WithTraceCarrierTimestamp(ts time.Time) TraceCarrierOpt {
	return func(tc *TraceCarrier) {
		tc.Timestamp = ts
	}
}

func WithTraceCarrierSpanID(sid *trace.SpanID) TraceCarrierOpt {
	return func(tc *TraceCarrier) {
		tc.SpanID = sid
	}
}
