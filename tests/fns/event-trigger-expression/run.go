package retries_go_test

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/testdsl"
)

func init() {
	testdsl.Register(Do)
}

// "if": "event.data.ok == true && size(event.data.cart_items.filter(i, i.price > 50)) > 2",

func Do(ctx context.Context) testdsl.Chain {

	return testdsl.Chain{
		// # First test
		//
		// Send an event which has ok set to true, but only one cart items matching the price.
		testdsl.SendEvent(event.Event{
			Name: "test/trigger-expression",
			Data: map[string]any{
				"ok": true,
				"cart_items": []map[string]any{
					{
						"price": 20,
					},
					{
						"price": 100,
					},
				},
			},
		}),
		testdsl.RequireOutputWithin("received message", 500*time.Millisecond),
		// Ensure runner consumes event.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"event":   "test/trigger-expression",
			"message": "received message",
		}, 5*time.Millisecond),
		testdsl.RequireNoLogFieldsWithin(map[string]any{
			"caller":   "runner",
			"function": "event-trigger-expression",
			"message":  "initializing fn",
		}, time.Second),

		// # Second test
		//
		// OK is false, cart items match.
		testdsl.SendEvent(event.Event{
			Name: "test/trigger-expression",
			Data: map[string]any{
				"ok": false,
				"cart_items": []map[string]any{
					{
						"price": 999,
					},
					{
						"price": 100,
					},
				},
			},
		}),
		testdsl.RequireOutputWithin("received message", 500*time.Millisecond),
		// Ensure runner consumes event.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"event":   "test/trigger-expression",
			"message": "received message",
		}, 5*time.Millisecond),
		testdsl.RequireNoLogFieldsWithin(map[string]any{
			"caller":   "runner",
			"function": "event-trigger-expression",
			"message":  "initializing fn",
		}, time.Second),

		// # Third test: success
		//
		// OK is false, cart items match.
		testdsl.SendEvent(event.Event{
			Name: "test/trigger-expression",
			Data: map[string]any{
				"ok": true,
				"cart_items": []map[string]any{
					{
						"price": 999,
					},
					{
						"price": 100,
					},
				},
			},
		}),
		testdsl.RequireOutputWithin("received message", 500*time.Millisecond),
		// Ensure runner consumes event.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"event":   "test/trigger-expression",
			"message": "received message",
		}, 5*time.Millisecond),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":   "runner",
			"function": "event-trigger-expression",
			"message":  "initializing fn",
		}, testdsl.DefaultDuration),
	}
}
