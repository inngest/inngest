package batcher

import "context"

type RateLimiter interface {
	Eventer
	MaxCapacity() uint32
	Capacity() uint32
	GiveMe(target uint32)
	Start(ctx context.Context) error
}
