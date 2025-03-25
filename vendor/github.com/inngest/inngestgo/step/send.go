package step

import (
	"context"
	"errors"

	"github.com/inngest/inngestgo/internal"
	"github.com/inngest/inngestgo/internal/event"
)

// Send sends an event to Inngest.
func Send[DATA any](
	ctx context.Context,
	id string,
	event event.GenericEvent[DATA],
) (string, error) {
	return Run(ctx, id, func(ctx context.Context) (string, error) {
		sender, ok := internal.EventSenderFromContext(ctx)
		if !ok {
			return "", errors.New("no event sender found in context")
		}

		return sender.Send(ctx, event)
	})
}

// SendMany sends a batch of events to Inngest.
func SendMany[DATA any](
	ctx context.Context,
	id string,
	events []event.GenericEvent[DATA],
) ([]string, error) {
	return Run(ctx, id, func(ctx context.Context) ([]string, error) {
		sender, ok := internal.EventSenderFromContext(ctx)
		if !ok {
			return nil, errors.New("no event sender found in context")
		}

		many := make([]any, len(events))
		for i, event := range events {
			many[i] = event
		}
		return sender.SendMany(ctx, many)
	})
}
