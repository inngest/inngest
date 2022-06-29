package queue

import (
	"context"
	"time"
)

type Queue interface {
	Producer
	Consumer
}

type Producer interface {
	// Enqueue allows an item to be enqueued ton run at the given time.
	Enqueue(context.Context, Item, time.Time) error
}

type Consumer interface {
	// Run is a blocking function which listens to the queue and executes the
	// given function each time a new Item becomes available.
	//
	// This must only return an error if we fail to subscribe to the queue's
	// implementation and can no longer process jobs.
	Run(context.Context, func(context.Context, Item) error) error
}
