package event

import "context"

type Publisher interface {
	// Publish publishes an event to some backing store for recording, and so on
	Publish(ctx context.Context, evt TrackedEvent) error
}
