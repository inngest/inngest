package telemetry

import (
	"encoding/json"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

const psidKey = "psid"

// TraceCarrier stores the data that needs to be carried through systems.
// e.g. pubsub, queues, etc
type TraceCarrier struct {
	sync.Mutex
	Context map[string]string `json:"ctx,omitempty"`
}

func NewTraceCarrier() *TraceCarrier {
	return &TraceCarrier{
		Context: map[string]string{},
	}
}

func (tc *TraceCarrier) Unmarshal(data any) error {
	byt, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(byt, tc)
}

// Embed the parent spanID for propagation purposes
func (tc *TraceCarrier) AddParentSpanID(psc trace.SpanContext) {
	tc.Lock()
	defer tc.Unlock()

	if psc.IsValid() {
		tc.Context[psidKey] = psc.SpanID().String()
	}
}

// ParentSpanID returns the embedded spanID if it's available
func (tc *TraceCarrier) ParentSpanID() (*trace.SpanID, error) {
	val, ok := tc.Context[psidKey]
	if !ok {
		return nil, fmt.Errorf("spanID is not stored in carrier")
	}

	sid, err := trace.SpanIDFromHex(val)
	return &sid, err
}
