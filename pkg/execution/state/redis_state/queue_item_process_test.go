package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueItemProcessWithConstraintChecks(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName(consts.DefaultQueueShardName),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("without constraint api", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I:             qi,
			P:             p,
			CapacityLease: nil,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))
	})

	t.Run("with constraint api but no valid lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I:             qi,
			P:             p,
			CapacityLease: nil,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			<-time.After(3 * time.Second)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		// No extend calls should be fired
		require.Equal(t, 1, int(counter))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
	})

	t.Run("with constraint api and valid lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.AcquireCalls, 1)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I: qi,
			P: p,
			CapacityLease: &osqueue.CapacityLease{
				LeaseID:    resp.Leases[0].LeaseID,
				IssuedAtMS: clock.Now().UnixMilli(),
			},
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						// Ensure we tick the extend at least once
						clock.Advance(time.Second)
					}
				}
			}()

			<-time.After(3 * time.Second)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))

		service.Wait()

		// Expect at least 1 extend call
		require.Greater(t, len(cmLifecycles.ExtendCalls), 0)

		// Expect exactly 1 release call
		require.Equal(t, len(cmLifecycles.ReleaseCalls), 1)
	})

	t.Run("with matching lease extension intervals", func(t *testing.T) {
		reset()

		// Use the production default interval (QueueLeaseDuration / 2) for both tickers
		// This reproduces the bug where both tickers fire simultaneously
		leaseExtendInterval := osqueue.QueueLeaseDuration / 2

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// Use matching interval (same as queue item lease ticker)
			osqueue.WithCapacityLeaseExtendInterval(leaseExtendInterval),
			osqueue.WithLogger(l),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		// Acquire a lease (same pattern as existing test)
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        time.Minute, // Longer duration to allow multiple extensions
			MaximumLifetime: 5 * time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.AcquireCalls, 1)

		// Lease the queue item (required when queue item lease ticker fires)
		leaseID, err := shard.Lease(ctx, qi, osqueue.QueueLeaseDuration, clock.Now())
		require.NoError(t, err)
		qi.LeaseID = leaseID

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I: qi,
			P: p,
			CapacityLease: &osqueue.CapacityLease{
				LeaseID:    resp.Leases[0].LeaseID,
				IssuedAtMS: clock.Now().UnixMilli(),
			},
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			go func() {
				// Advance clock in intervals matching the lease extension interval
				// This ensures both tickers fire at the same time
				for i := 0; i < 3; i++ {
					select {
					case <-ctx.Done():
						return
					case <-time.After(100 * time.Millisecond):
						// Advance by the full lease extension interval to trigger both tickers simultaneously
						clock.Advance(leaseExtendInterval)
					}
				}
			}()

			// Wait for clock advances to complete
			<-time.After(500 * time.Millisecond)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))

		service.Wait()

		// With matching intervals, both tickers fire simultaneously.
		// Before the fix (single select loop), capacity lease extensions might not run.
		// After the fix (separate goroutines), capacity lease extensions should always run.
		// Expect at least 2 extend calls (we advanced clock 3 times by the interval)
		require.GreaterOrEqual(t, len(cmLifecycles.ExtendCalls), 2,
			"capacity lease extensions should run even when both tickers fire simultaneously")

		// Expect exactly 1 release call
		require.Equal(t, 1, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("with constraint api and valid lease and early release", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.AcquireCalls, 1)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I: qi,
			P: p,
			CapacityLease: &osqueue.CapacityLease{
				LeaseID:    resp.Leases[0].LeaseID,
				IssuedAtMS: clock.Now().UnixMilli(),
			},
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			released := make(chan struct{})
			var advancerDone sync.WaitGroup
			advancerDone.Go(func() {
				for {
					select {
					case <-released:
						return
					case <-time.After(time.Second):
						// Wait until both the queue-item lease ticker and the
						// capacity-lease extend ticker are blocked on the clock
						// before advancing. This ensures the extend goroutine has
						// finished processing and is ready for the next tick.
						clock.BlockUntil(2)
						clock.Advance(time.Second)
					}
				}
			})

			<-time.After(3 * time.Second)

			// Release the capacity early
			require.NotNil(t, ri.CapacityLease)

			// Stop clock advances and wait for the advancer goroutine to exit
			// before releasing the lease. This prevents advancing after the
			// lease is released, which would cause the extend goroutine to see
			// a stale/released lease. Note that a tick fired by the final
			// advance may still be buffered in the extend ticker's channel;
			// ProcessItem handles that race by discarding extend failures once
			// the lease has been released.
			close(released)
			advancerDone.Wait()

			err := ri.CapacityLease.Release()
			require.NoError(t, err)

			// And do some more processing before returning
			<-time.After(500 * time.Millisecond)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))

		service.Wait()

		// Expect at least 1 extend call
		require.Greater(t, len(cmLifecycles.ExtendCalls), 0)

		// Expect exactly 2 release calls
		require.Equal(t, 2, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("with constraint api and early release racing extend tick", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I: qi,
			P: p,
			CapacityLease: &osqueue.CapacityLease{
				LeaseID:    resp.Leases[0].LeaseID,
				IssuedAtMS: clock.Now().UnixMilli(),
			},
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			require.NotNil(t, ri.CapacityLease)

			// Fire the capacity-lease extend ticker, then release the lease
			// immediately, before the extend goroutine has necessarily
			// consumed the tick. The tick stays buffered in the ticker
			// channel, so the extend goroutine can observe it after Release()
			// cancelled the extend context and released the lease.
			// ProcessItem must treat the resulting stale tick (or in-flight
			// extension failure) as benign instead of requeueing the item and
			// aborting the handler.
			clock.BlockUntil(2)
			clock.Advance(time.Second)

			err := ri.CapacityLease.Release()
			require.NoError(t, err)

			// And do some more processing before returning
			<-time.After(100 * time.Millisecond)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))

		service.Wait()
	})
}

func TestQueueItemProcessCleanupUsesRenewedLease(t *testing.T) {
	tests := []struct {
		name    string
		runErr  error
		requeue bool
	}{
		{
			name: "dequeue on success",
		},
		{
			name:    "requeue on retryable error",
			runErr:  errors.New("retryable"),
			requeue: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			clock := clockwork.NewFakeClock()

			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			q, shard := newQueue(
				t,
				rc,
				osqueue.WithClock(clock),
				osqueue.WithBackoffFunc(func(int) time.Time {
					return clock.Now().Add(time.Minute)
				}),
			)

			accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)
			jobID := "lease-cleanup-" + tc.name
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				ID:          jobID,
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: osqueue.Item{
					JobID:       &jobID,
					WorkspaceID: envID,
					Kind:        osqueue.KindStart,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  fnID,
						RunID:       runID,
					},
				},
			}, clock.Now(), osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, osqueue.QueueLeaseDuration, clock.Now())
			require.NoError(t, err)
			item.LeaseID = leaseID
			initialLease := *leaseID
			initialGeneration := item.GenerationID

			started := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- q.ProcessItem(ctx, osqueue.ProcessItem{
					P: osqueue.ItemPartition(ctx, item),
					I: item,
				}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
					close(started)
					deadline := time.After(time.Second)
					ticker := time.NewTicker(10 * time.Millisecond)
					defer ticker.Stop()

					for {
						loaded, err := shard.LoadQueueItem(ctx, item.ID)
						if err == nil && loaded.LeaseID != nil && *loaded.LeaseID != initialLease {
							return osqueue.RunResult{}, tc.runErr
						}

						select {
						case <-deadline:
							return osqueue.RunResult{}, errors.New("timed out waiting for lease renewal")
						case <-ticker.C:
						case <-ctx.Done():
							return osqueue.RunResult{}, ctx.Err()
						}
					}
				})
			}()

			require.Eventually(t, func() bool {
				select {
				case <-started:
					return true
				default:
					return false
				}
			}, time.Second, 10*time.Millisecond)

			clock.Advance(osqueue.QueueLeaseDuration / 2)
			r.SetTime(clock.Now())

			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for ProcessItem to finish")
			}

			loaded, err := shard.LoadQueueItem(ctx, item.ID)
			if !tc.requeue {
				require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
				require.Nil(t, loaded)
				return
			}

			require.NoError(t, err)
			require.Nil(t, loaded.LeaseID)
			require.Equal(t, initialGeneration+1, loaded.GenerationID)
			require.Equal(t, 1, loaded.Data.Attempt)
		})
	}
}

func TestQueueProcessorPreLeaseWithConstraintAPI(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName(consts.DefaultQueueShardName),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("with constraint api", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  10,
						FunctionConcurrency: 5,
					},
				}
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		iter := osqueue.ProcessorIterator{
			Partition:            &p,
			Items:                []*osqueue.QueueItem{&qi},
			PartitionContinueCtr: 0,
			Queue:                q,
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		err = iter.Process(ctx, &qi)
		require.NoError(t, err)

		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("with constraint api and no active lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  10,
						FunctionConcurrency: 5,
					},
				}
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		iter := osqueue.ProcessorIterator{
			Partition:            &p,
			Items:                []*osqueue.QueueItem{&qi},
			PartitionContinueCtr: 0,
			Queue:                q,
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		err = iter.Process(ctx, &qi)
		require.NoError(t, err)

		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("with constraint api and active capacity lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.AcquireCalls, 1)

		cmLifecycles.Reset()

		// Set capacity lease ID
		qi.CapacityLease = &osqueue.CapacityLease{
			LeaseID:    resp.Leases[0].LeaseID,
			IssuedAtMS: clock.Now().UnixMilli(),
		}

		p := osqueue.ItemPartition(ctx, qi)

		iter := osqueue.ProcessorIterator{
			Partition:            &p,
			Items:                []*osqueue.QueueItem{&qi},
			PartitionContinueCtr: 0,
			Queue:                q,
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		err = iter.Process(ctx, &qi)
		require.NoError(t, err)

		// No further Constraint API calls should be made
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))

		// Expect item to be sent to worker with capacity lease + request to disable constraint updates
		item := <-q.Workers()
		require.Equal(t, qi, item.I)
		require.Equal(t, qi.CapacityLease, item.CapacityLease)
	})
}

func TestPartitionProcessRequeueAfterLimitedWithConstraintAPI(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName(consts.DefaultQueueShardName),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("without constraintapi and no leases", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		items := []*osqueue.QueueItem{}

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		items = append(items, &qi)

		for range amount - 1 {
			qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			items = append(items, &qi)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		iter := osqueue.ProcessorIterator{
			Partition: &p,
			// Pass in all items
			Items:                items,
			PartitionContinueCtr: 0,
			Queue:                q,
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		require.False(t, iter.IsRequeuable())

		// Iterate over all items
		err = iter.Iterate(ctx)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, int32(2), iter.CtrSuccess.Load())

		// third item was concurrency limited, we stopped
		require.Equal(t, int32(1), iter.CtrConcurrency.Load(), r.Dump())

		// we should requeue the item
		require.True(t, iter.IsRequeuable())

		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))

		// the 2 items are in progress
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // item should also be added to scavenger index
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 3)
	})

	t.Run("without constraintapi and no leases using processPartition", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		for range amount - 1 {
			_, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		// score in global set is at earliest item
		require.Equal(t, start.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))

		err = q.ProcessPartition(ctx, &p, 0, false)
		require.NoError(t, err)

		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))

		// first two items were successfully leased
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // item should also be added to scavenger index

		// remaining items are still in partition
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect no calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 3)

		// partition was requeued
		require.Equal(t, start.Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))
	})

	t.Run("with constraintapi and no valid leases", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		items := []*osqueue.QueueItem{}

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		items = append(items, &qi)

		for range amount - 1 {
			qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			items = append(items, &qi)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		iter := osqueue.ProcessorIterator{
			Partition: &p,
			// Pass in all items
			Items:                items,
			PartitionContinueCtr: 0,
			Queue:                q,
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		require.False(t, iter.IsRequeuable())

		// Iterate over all items
		err = iter.Iterate(ctx)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, int32(2), iter.CtrSuccess.Load())

		// third item was concurrency limited, we stopped
		require.Equal(t, int32(1), iter.CtrConcurrency.Load(), r.Dump())

		// we should requeue the item
		require.True(t, iter.IsRequeuable())

		// the 2 items are in progress
		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))   // since we used constraint API, item should not be added to in progress items set
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // but instead to partition scavenger index
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect 2 successful and 1 failed calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 3)
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[2].GrantedLeases, 0)
		require.Equal(t, cmLifecycles.AcquireCalls[2].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)
	})

	t.Run("with constraintapi and no leases using processPartition", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		for range amount - 1 {
			_, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		// score in global set is at earliest item
		require.Equal(t, start.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))

		err = q.ProcessPartition(ctx, &p, 0, false)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))   // items should not be in old concurrency zset
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // but in scavenger set

		// remaining items are still in partition
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect 2 successful and 1 failed calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 3)
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[2].GrantedLeases, 0)
		require.Equal(t, cmLifecycles.AcquireCalls[2].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)

		// partition was requeued
		require.Equal(t, start.Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))
	})

	t.Run("with constraintapi and valid leases", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithLogger(l),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		amount := 10

		/*
		* - Acquire lease for first item
		* - Enqueue item with lease details
		* - Enqueue following items (with later timestamps)
		* - Process partition should peek all items
		* - First item with active lease should be allowed
		* - Second item should be Acquire-checked
		* - Second item should be limited and stop processing
		* - Partition should be requeued
		 */

		var qi osqueue.QueueItem
		{
			var err error
			firstItemID := util.XXHash("item0")

			// Acquire a lease
			resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
				AccountID:            accountID,
				EnvID:                envID,
				IdempotencyKey:       firstItemID,
				FunctionID:           fnID,
				LeaseIdempotencyKeys: []string{firstItemID},
				Amount:               1,
				Configuration: constraintapi.ConstraintConfig{
					FunctionVersion: 1,
					Concurrency: constraintapi.ConcurrencyConfig{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				},
				Constraints: []constraintapi.ConstraintItem{
					{
						Kind: constraintapi.ConstraintKindConcurrency,
						Concurrency: &constraintapi.ConcurrencyConstraint{
							Scope: enums.ConcurrencyScopeAccount,
						},
					},
					{
						Kind: constraintapi.ConstraintKindConcurrency,
						Concurrency: &constraintapi.ConcurrencyConstraint{
							Scope: enums.ConcurrencyScopeFn,
						},
					},
				},
				CurrentTime:     clock.Now(),
				Duration:        10 * time.Second,
				MaximumLifetime: time.Minute,
				Source: constraintapi.LeaseSource{
					Service:           constraintapi.ServiceExecutor,
					Location:          constraintapi.CallerLocationItemLease,
					RunProcessingMode: constraintapi.RunProcessingModeBackground,
				},
			})
			require.NoError(t, err)
			require.Len(t, resp.Leases, 1)

			require.Len(t, cmLifecycles.AcquireCalls, 1)

			cmLifecycles.Reset()

			// Set capacity lease ID on first item
			item.CapacityLease = &osqueue.CapacityLease{
				LeaseID:    resp.Leases[0].LeaseID,
				IssuedAtMS: clock.Now().UnixMilli(),
			}
			// Manually set ID for first item
			item.ID = util.XXHash("item0")

			qi, err = shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{
				PassthroughJobId: true,
			})
			require.NoError(t, err)

			// reset for following items
			item.CapacityLease = nil
			item.ID = ""
		}

		for i := range amount - 1 {
			_, err := shard.EnqueueItem(ctx, item, start.Add(time.Millisecond*time.Duration(i+1)), osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		// score in global set is at earliest item
		require.Equal(t, start.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))

		err = q.ProcessPartition(logger.WithStdlib(ctx, l), &p, 0, false)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))   // items should not be in old concurrency zset
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // but in scavenger set

		// remaining items are still in partition
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect 1 successful and 1 failed calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 2)
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 0)
		require.Equal(t, cmLifecycles.AcquireCalls[1].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)

		// partition was requeued
		require.Equal(t, start.Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))
	})
}
