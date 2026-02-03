package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func BenchmarkAcquire(b *testing.B) {
	r := miniredis.RunT(b)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(b, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithRateLimitClient(rc),
		WithQueueShards(map[string]rueidis.Client{
			"test": rc,
		}),
		WithClock(clock),
		WithNumScavengerShards(1),
		WithQueueStateKeyPrefix("q:v1"),
		WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(b, err)
	require.NotNil(b, cm)

	// The following tests are essential functionality. We also have detailed test for each method,
	// to cover edge cases.

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:  20,
			FunctionConcurrency: 5,
		},
	}

	acctConcurrency := fmt.Sprintf("{%s}:concurrency:account:%s", cm.queueStateKeyPrefix, accountID)
	fnConcurrency := fmt.Sprintf("{%s}:concurrency:p:%s", cm.queueStateKeyPrefix, fnID)

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeAccount,
				InProgressItemKey: acctConcurrency,
			},
		},
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				InProgressItemKey: fnConcurrency,
			},
		},
	}

	// Reset timer after setup to only measure Acquire calls
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Stop timer to clear state without counting it
		b.StopTimer()
		r.FlushAll()
		b.StartTimer()

		// Use unique idempotency key per iteration to avoid idempotency hits
		leaseIdempotencyKey := fmt.Sprintf("event-%d", i)

		acquireReq := &CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			FunctionID:           fnID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{leaseIdempotencyKey},
			IdempotencyKey:       leaseIdempotencyKey,
			LeaseRunIDs:          nil,
			Duration:             5 * time.Second,
			Source: LeaseSource{
				Service:           ServiceExecutor,
				Location:          CallerLocationSchedule,
				RunProcessingMode: RunProcessingModeBackground,
			},
			Configuration:   config,
			Constraints:     constraints,
			CurrentTime:     clock.Now(),
			MaximumLifetime: time.Minute,
			Migration: MigrationIdentifier{
				QueueShard: "test",
			},
		}

		resp, err := cm.Acquire(ctx, acquireReq)
		if err != nil {
			b.Fatalf("Acquire failed: %v", err)
		}
		if resp == nil {
			b.Fatal("Response is nil")
		}
	}
}
