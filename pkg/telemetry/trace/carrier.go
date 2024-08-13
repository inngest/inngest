package trace

import (
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const (
	sidkey = "sid"
)

type TraceCarrierOpt func(tc *TraceCarrier)

// TraceCarrier stores the data that needs to be carried through systems.
// e.g. pubsub, queues, etc
type TraceCarrier struct {
	// Context is used for text map propagation, this typically stores the traceID and the spanID
	// of the time of running Inject.
	Context   map[string]string `json:"ctx,omitempty"`
	Timestamp time.Time         `json:"ts"`
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

func (tc *TraceCarrier) CanResumePause() bool {
	sid := tc.SpanID()
	return tc.Context != nil && tc.Timestamp.UnixMilli() > 0 && sid.IsValid()
}

func (tc *TraceCarrier) SpanID() trace.SpanID {
	if tc.Context == nil {
		return trace.SpanID{}
	}

	if val, ok := tc.Context[sidkey]; ok {
		if sid, err := trace.SpanIDFromHex(val); err == nil {
			return sid
		}
	}

	return trace.SpanID{}
}

func WithTraceCarrierTimestamp(ts time.Time) TraceCarrierOpt {
	return func(tc *TraceCarrier) {
		tc.Timestamp = ts
	}
}

func WithTraceCarrierSpanID(sid *trace.SpanID) TraceCarrierOpt {
	return func(tc *TraceCarrier) {
		if tc.Context == nil {
			tc.Context = map[string]string{}
		}
		tc.Context[sidkey] = sid.String()
	}
}
