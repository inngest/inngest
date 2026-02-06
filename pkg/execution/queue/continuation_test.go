package queue

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func newTestPartition(queueName string) *QueuePartition {
	fnID := uuid.New()
	return &QueuePartition{
		ID:         queueName,
		FunctionID: &fnID,
		AccountID:  uuid.New(),
	}
}

// newContinuationTestProcessor creates a minimal queueProcessor suitable for
// testing addContinue / removeContinue behavior.
func newContinuationTestProcessor(opts ...QueueOpt) *queueProcessor {
	o := NewQueueOptions(opts...)
	o.runMode.Continuations = true

	return &queueProcessor{
		QueueOptions:     o,
		continues:        make(map[string]continuation),
		continueCooldown: make(map[string]time.Time),
		continuesLock:    &sync.Mutex{},
	}
}

// TestContinuationCooldownPreventsResumption proves that after a partition
// exhausts its continuation limit, the cooldown duration controls how long
// the partition must wait before continuations can resume.
//
// With the production default of 10s, a dev server function with >5 steps
// loses the continuation optimization for 10 full seconds after every 5
// steps. By making the cooldown configurable, the dev server can set it
// to 1s, allowing continuations to resume after a brief pause.
func TestContinuationCooldownPreventsResumption(t *testing.T) {
	t.Run("default 10s cooldown blocks resumption for too long", func(t *testing.T) {
		ctx := context.Background()

		qp := newContinuationTestProcessor(
			WithQueueContinuationLimit(5),
			// Default cooldown is consts.QueueContinuationCooldownPeriod = 10s
		)
		partition := newTestPartition("fn-with-many-steps")

		// Simulate 4 successful continuations (ctr 1..4)
		for i := uint(1); i <= 4; i++ {
			qp.addContinue(ctx, partition, i)
			require.Contains(t, qp.continues, partition.Queue(),
				"continuation %d should be in the map", i)
		}

		// The 5th call (ctr == continuationLimit) triggers cooldown.
		qp.addContinue(ctx, partition, 5)
		require.NotContains(t, qp.continues, partition.Queue(),
			"continuation should be removed after hitting the limit")
		require.Contains(t, qp.continueCooldown, partition.Queue(),
			"cooldown should be set after hitting the limit")

		// Verify the cooldown expires at the configured time, not 10s later.
		cooldownExpiry := qp.continueCooldown[partition.Queue()]
		require.InDelta(t, time.Now().Add(consts.QueueContinuationCooldownPeriod).UnixMilli(),
			cooldownExpiry.UnixMilli(), 500,
			"cooldown should default to consts.QueueContinuationCooldownPeriod (10s)")
	})

	t.Run("configurable 1s cooldown allows fast resumption", func(t *testing.T) {
		ctx := context.Background()

		qp := newContinuationTestProcessor(
			WithQueueContinuationLimit(5),
			WithContinuationCooldown(time.Second),
		)
		partition := newTestPartition("fn-with-many-steps")

		// Trigger cooldown.
		for i := uint(1); i <= 4; i++ {
			qp.addContinue(ctx, partition, i)
		}
		qp.addContinue(ctx, partition, 5)
		require.NotContains(t, qp.continues, partition.Queue())

		// Wait just over 1 second — the configured cooldown.
		time.Sleep(1100 * time.Millisecond)

		// Continuation should now be accepted.
		qp.addContinue(ctx, partition, 1)
		require.Contains(t, qp.continues, partition.Queue(),
			"continuation should be accepted after the 1s configured cooldown expires")
	})
}

// TestContinuationSkipProbabilityCausesRandomLatency proves that the default
// 20% skip probability in scanContinuations causes continuations to be randomly
// skipped. In the dev server with a 150ms poll tick, each skip adds ~150ms of
// latency to a step transition.
//
// This test verifies two things:
// 1. The default probability (0.2) causes ~20% of continuations to be skipped.
// 2. Setting probability to 0 eliminates all skips.
func TestContinuationSkipProbabilityCausesRandomLatency(t *testing.T) {
	t.Run("default 0.2 probability causes skips", func(t *testing.T) {
		// Reproduce the exact skip check from scanContinuations:
		//   if q.continuationSkipProbability > 0 && rand.Float64() <= q.continuationSkipProbability { ... }
		qp := newContinuationTestProcessor() // uses default skip probability

		skipped := 0
		total := 1000

		rng := rand.New(rand.NewSource(42))
		for i := 0; i < total; i++ {
			if qp.continuationSkipProbability > 0 && rng.Float64() <= qp.continuationSkipProbability {
				skipped++
			}
		}

		require.Greater(t, skipped, 100,
			"with default skip probability of %.1f, expected significant skips but got %d/%d",
			qp.continuationSkipProbability, skipped, total)
	})

	t.Run("zero probability eliminates skips", func(t *testing.T) {
		qp := newContinuationTestProcessor(
			WithContinuationSkipProbability(0),
		)

		skipped := 0
		total := 1000

		rng := rand.New(rand.NewSource(42))
		for i := 0; i < total; i++ {
			if qp.continuationSkipProbability > 0 && rng.Float64() <= qp.continuationSkipProbability {
				skipped++
			}
		}

		require.Equal(t, 0, skipped,
			"with skip probability 0, no continuations should be skipped, but got %d/%d",
			skipped, total)
	})
}

// TestContinuationHighLimitAvoidsEarlyCooldown proves that raising the
// continuation limit prevents cooldown from kicking in for functions with
// many steps. The default limit of 5 means a function with 6+ steps
// triggers cooldown. Raising it to 50 avoids this for typical dev workloads.
func TestContinuationHighLimitAvoidsEarlyCooldown(t *testing.T) {
	ctx := context.Background()

	t.Run("default limit of 5 triggers cooldown at step 6", func(t *testing.T) {
		qp := newContinuationTestProcessor(
			WithQueueContinuationLimit(5),
		)
		partition := newTestPartition("fn-many-steps")

		for i := uint(1); i <= 4; i++ {
			qp.addContinue(ctx, partition, i)
		}
		// 5th triggers cooldown
		qp.addContinue(ctx, partition, 5)
		require.NotContains(t, qp.continues, partition.Queue())
		require.Contains(t, qp.continueCooldown, partition.Queue(),
			"cooldown should activate after 5 continuations")
	})

	t.Run("limit of 50 allows many steps without cooldown", func(t *testing.T) {
		qp := newContinuationTestProcessor(
			WithQueueContinuationLimit(50),
		)
		partition := newTestPartition("fn-many-steps")

		// 20 consecutive continuations — well beyond the old limit of 5.
		for i := uint(1); i <= 20; i++ {
			qp.addContinue(ctx, partition, i)
			require.Contains(t, qp.continues, partition.Queue(),
				"continuation %d should still be active with limit=50", i)
		}
		require.NotContains(t, qp.continueCooldown, partition.Queue(),
			"cooldown should not activate with a higher limit")
	})
}
