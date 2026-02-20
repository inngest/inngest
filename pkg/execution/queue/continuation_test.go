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

// TestContinuationDefaultOptions verifies that NewQueueOptions sets the expected
// production defaults for continuation-related fields.
func TestContinuationDefaultOptions(t *testing.T) {
	o := NewQueueOptions()

	require.Equal(t, uint(consts.DefaultQueueContinueLimit), o.continuationLimit,
		"default continuation limit should match consts.DefaultQueueContinueLimit")
	require.Equal(t, consts.QueueContinuationSkipProbability, o.continuationSkipProbability,
		"default skip probability should match consts.QueueContinuationSkipProbability")
}

// simulateMultiStepFunction simulates a function with stepCount steps running
// through the continuation system. Each step completes and calls addContinue,
// mimicking the real executor flow. When cooldown triggers or the limit is hit,
// the partition is processed via the regular peek path which resets the counter
// to 1 on the next attempt.
//
// Returns the number of steps that were accepted as continuations (i.e. present
// in the continues map after addContinue).
func simulateMultiStepFunction(qp *queueProcessor, partition *QueuePartition, stepCount int) int {
	ctx := context.Background()
	accepted := 0
	ctr := uint(1)

	for step := 0; step < stepCount; step++ {
		qp.addContinue(ctx, partition, ctr)

		if _, ok := qp.continues[partition.Queue()]; ok {
			accepted++
			ctr++
		} else {
			// Continuation was rejected (cooldown or limit). In production the
			// partition falls back to the regular peek path, which resets the
			// continuation counter to 1 for the next attempt.
			ctr = 1
		}
	}
	return accepted
}

// TestContinuationRuntimeProductionDefaults simulates a 20-step function under
// production defaults (limit=5, cooldown=10s, skip=0.2). It proves that:
//   - Only the first 4 steps use continuations (step 5 hits the limit).
//   - The remaining 15 steps are all rejected because the 10s cooldown is active.
//   - ~20% of scan ticks would be skipped by the skip probability.
func TestContinuationRuntimeProductionDefaults(t *testing.T) {
	t.Run("20-step function loses continuations after step 4", func(t *testing.T) {
		qp := newContinuationTestProcessor() // production defaults
		partition := newTestPartition("fn-20-steps")

		accepted := simulateMultiStepFunction(qp, partition, 20)

		// With limit=5: steps 1-4 accepted (ctr 1..4), step 5 (ctr=5) triggers
		// cooldown. Steps 6-20 all attempt ctr=1 but are rejected by the 10s cooldown.
		require.Equal(t, 4, accepted,
			"production defaults: only 4 of 20 steps should use continuations")

		// Cooldown is active for the partition.
		require.Contains(t, qp.continueCooldown, partition.Queue(),
			"production defaults: cooldown should be active")

		// The cooldown won't expire for ~10 seconds.
		cooldownExpiry := qp.continueCooldown[partition.Queue()]
		require.True(t, cooldownExpiry.After(time.Now().Add(9*time.Second)),
			"production defaults: cooldown should not expire for ~10s")
	})

	t.Run("skip probability causes scan-tick skips", func(t *testing.T) {
		qp := newContinuationTestProcessor() // production defaults

		// Simulate 1000 scan ticks using the same check as scanContinuations.
		skipped := 0
		rng := rand.New(rand.NewSource(99))
		for i := 0; i < 1000; i++ {
			if qp.continuationSkipProbability > 0 && rng.Float64() <= qp.continuationSkipProbability {
				skipped++
			}
		}

		// With 0.2 probability, expect ~200 skips. Each skip in the dev server
		// (150ms tick) adds a full tick of latency to a step transition.
		require.InDelta(t, 200, skipped, 50,
			"production defaults: ~20%% of scan ticks should be skipped")
	})
}

// TestContinuationRuntimeDevServerOptions simulates the same 20-step function
// under dev server options (limit=0, skip=0). It proves that:
//   - All 20 steps use continuations without interruption.
//   - No cooldown is ever triggered (limit=0 means unlimited).
//   - Zero scan ticks are skipped.
func TestContinuationRuntimeDevServerOptions(t *testing.T) {
	t.Run("20-step function uses continuations for every step", func(t *testing.T) {
		qp := newContinuationTestProcessor(
			WithQueueContinuationLimit(0), // unlimited
			WithContinuationSkipProbability(0),
		)
		partition := newTestPartition("fn-20-steps")

		accepted := simulateMultiStepFunction(qp, partition, 20)

		require.Equal(t, 20, accepted,
			"dev server: all 20 steps should use continuations")
		require.NotContains(t, qp.continueCooldown, partition.Queue(),
			"dev server: no cooldown should be triggered")
	})

	t.Run("zero skip probability means zero scan-tick skips", func(t *testing.T) {
		qp := newContinuationTestProcessor(
			WithContinuationSkipProbability(0),
		)

		skipped := 0
		rng := rand.New(rand.NewSource(99))
		for i := 0; i < 1000; i++ {
			if qp.continuationSkipProbability > 0 && rng.Float64() <= qp.continuationSkipProbability {
				skipped++
			}
		}

		require.Equal(t, 0, skipped,
			"dev server: no scan ticks should be skipped")
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

// TestContinuationLimitBehavior proves the limit controls when cooldown kicks in,
// and that limit=0 disables the limit entirely (unlimited continuations).
func TestContinuationLimitBehavior(t *testing.T) {
	ctx := context.Background()

	t.Run("default limit of 5 triggers cooldown at step 5", func(t *testing.T) {
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

	t.Run("limit of 0 means unlimited", func(t *testing.T) {
		qp := newContinuationTestProcessor(
			WithQueueContinuationLimit(0),
		)
		partition := newTestPartition("fn-many-steps")

		// 100 consecutive continuations — no limit ever triggers.
		for i := uint(1); i <= 100; i++ {
			qp.addContinue(ctx, partition, i)
			require.Contains(t, qp.continues, partition.Queue(),
				"continuation %d should still be active with limit=0 (unlimited)", i)
		}
		require.NotContains(t, qp.continueCooldown, partition.Queue(),
			"cooldown should never activate with limit=0")
	})
}
