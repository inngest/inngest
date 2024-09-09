package batcher

import (
	"context"
	"time"
)

type LeaseManager interface {
	RaiseEventsTo(e Eventer)
	Provision(ctx context.Context) (err error)
	CreatePartitions(ctx context.Context, count int)
	LeasePartition(ctx context.Context, id string, index uint32) (leaseTime time.Duration)
}
