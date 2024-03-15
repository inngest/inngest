package telemetry

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/baggage"
)

// TraceCarrier stores the data that needs to be carried through systems.
// e.g. pubsub, queues, etc
type TraceCarrier struct {
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

func AddBaggageMap(ctx context.Context, attrs map[string]string) (context.Context, error) {
	for key, value := range attrs {
		newctx, err := AddBaggage(ctx, key, value)
		if err != nil {
			return ctx, err
		}
		ctx = newctx
	}
	return ctx, nil
}

func AddBaggage(ctx context.Context, key, value string) (context.Context, error) {
	bag := baggage.FromContext(ctx)

	multispanattr, err := baggage.NewMember(key, value)
	if err != nil {
		return ctx, fmt.Errorf("invalid span attr: %v", err)
	}

	bag, err = bag.SetMember(multispanattr)
	if err != nil {
		return ctx, fmt.Errorf("invalid baggage: %v", err)
	}

	return baggage.ContextWithBaggage(ctx, bag), nil
}
