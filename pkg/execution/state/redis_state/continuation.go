package redis_state

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

func newShadowContinuation(shardName string) *shadowCont {
	return &shadowCont{
		continues: map[string]shadowContinuation{},
		cooldown:  map[string]time.Time{},
		limit:     0,
		shardName: shardName,
	}
}

// shadowCont represents the continuation for shadow partitions
type shadowCont struct {
	sync.Mutex

	continues map[string]shadowContinuation
	cooldown  map[string]time.Time
	limit     uint

	shardName string
}

func (sc *shadowCont) Add(ctx context.Context, p *QueueShadowPartition, ctr uint) {
	sc.Lock()
	defer sc.Unlock()

	if ctr == 1 {
		if len(sc.continues) > consts.QueueShadowContinuationMaxPartitions {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": sc.shardName, "op": "max_capacity"}})
			return
		}

		if t, ok := sc.cooldown[p.PartitionID]; ok && t.After(time.Now()) {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": sc.shardName, "op": "cooldown"}})
			return
		}

		delete(sc.cooldown, p.PartitionID)
	}

	c, ok := sc.continues[p.PartitionID]
	if !ok || c.count < ctr {
		sc.continues[p.PartitionID] = shadowContinuation{shadowPart: p, count: ctr}
		metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": sc.shardName, "op": "added"}})
	}
}

func (sc *shadowCont) Remove(ctx context.Context, p *QueueShadowPartition, cooldown bool) {
	sc.Lock()
	defer sc.Unlock()

	delete(sc.continues, p.PartitionID)

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		sc.cooldown[p.PartitionID] = time.Now().Add(consts.QueueShadowContinuationCooldownPeriod)
		metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": sc.shardName, "op": "removed"}})
	}
}

func (sc *shadowCont) Continuations() iter.Seq[shadowContinuation] {
	cont := []shadowContinuation{}

	sc.Lock()
	for _, c := range sc.continues {
		cont = append(cont, c)
	}
	sc.Unlock()

	return func(yield func(shadowContinuation) bool) {
		for _, c := range cont {
			if !yield(c) {
				return
			}
		}
	}
}

func (sc *shadowCont) Has(p *QueueShadowPartition) bool {
	sc.Lock()
	defer sc.Unlock()

	_, ok := sc.continues[p.PartitionID]
	return ok
}

func (sc *shadowCont) IsWithinLimit(ctr uint) bool {
	return ctr >= sc.limit
}

// shadowContinuation is the equivalent of continuation for shadow partitions
type shadowContinuation struct {
	shadowPart *QueueShadowPartition
	count      uint
}
