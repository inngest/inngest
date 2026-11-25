package queue

import (
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// KeyQueueCapacityLeaseRepairLimits bound repair work independently of customer
// execution slots. These are process-local limits for each Redis queue shard;
// adding executor replicas increases the aggregate repair budget.
type KeyQueueCapacityLeaseRepairLimits struct {
	Concurrency int
	Rate        float64 // Redis repair attempts per second, including rejected mutations.
	Burst       int
	MaxDuration time.Duration
}

func DefaultKeyQueueCapacityLeaseRepairLimits() KeyQueueCapacityLeaseRepairLimits {
	return KeyQueueCapacityLeaseRepairLimits{
		Concurrency: 2, Rate: 100, Burst: 100, MaxDuration: 250 * time.Millisecond,
	}
}

// KeyQueueCapacityLeaseRepairBudget shares concurrency and rate limits across
// processors and scanners. A custom budget should likewise be shared by all
// processors using a shard in the process, rather than allocated per function.
type KeyQueueCapacityLeaseRepairBudget struct {
	limits KeyQueueCapacityLeaseRepairLimits
	shards sync.Map
}

func NewKeyQueueCapacityLeaseRepairBudget(limits KeyQueueCapacityLeaseRepairLimits) (*KeyQueueCapacityLeaseRepairBudget, error) {
	if limits.Concurrency <= 0 || limits.Rate <= 0 || math.IsNaN(limits.Rate) || math.IsInf(limits.Rate, 0) || limits.Burst <= 0 || limits.MaxDuration <= 0 {
		return nil, fmt.Errorf("capacity lease repair limits must be positive and finite")
	}
	return &KeyQueueCapacityLeaseRepairBudget{limits: limits}, nil
}

var defaultKeyQueueCapacityLeaseRepairBudget = &KeyQueueCapacityLeaseRepairBudget{
	limits: DefaultKeyQueueCapacityLeaseRepairLimits(),
}

type capacityLeaseRepairShardBudget struct {
	sem     *semaphore.Weighted
	limiter *rate.Limiter
}

func (b *KeyQueueCapacityLeaseRepairBudget) shard(name string) *capacityLeaseRepairShardBudget {
	if existing, ok := b.shards.Load(name); ok {
		return existing.(*capacityLeaseRepairShardBudget)
	}
	created := &capacityLeaseRepairShardBudget{
		sem:     semaphore.NewWeighted(int64(b.limits.Concurrency)),
		limiter: rate.NewLimiter(rate.Limit(b.limits.Rate), b.limits.Burst),
	}
	actual, _ := b.shards.LoadOrStore(name, created)
	return actual.(*capacityLeaseRepairShardBudget)
}

func (b *capacityLeaseRepairShardBudget) retryAfter(now time.Time) time.Duration {
	missing := max(0, 1-b.limiter.TokensAt(now))
	return time.Duration(missing / float64(b.limiter.Limit()) * float64(time.Second))
}
