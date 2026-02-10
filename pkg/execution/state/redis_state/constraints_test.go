package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestItemLeaseConstraintCheck(t *testing.T) {
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
		constraintapi.WithShardName("default"),
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

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	start := clock.Now()

	t.Run("waive checks for system queues", func(t *testing.T) {
		reset()

		qn := "example-system-queue"
		item := osqueue.QueueItem{
			Data: osqueue.Item{
				Payload:    json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{},
				QueueName:  &qn,
			},
			QueueName: &qn,
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.CapacityLease)
		require.True(t, res.SkipConstraintChecks)

		// Do not expect a call for the system queue
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("skip constraintapi but require checks when missing identifier", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.Error(t, err)
		require.ErrorContains(t, err, "missing accountID")
		require.ErrorContains(t, err, "missing envID")

		// No lease acquired
		require.Nil(t, res.CapacityLease)
		require.False(t, res.SkipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing identifiers
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("skip constraintapi but require checks when capacity manager not configured", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.CapacityLease)
		require.False(t, res.SkipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing capacity manager
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("skip constraintapi but require checks when feature flag disabled", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return false // disable flag
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.CapacityLease)
		require.False(t, res.SkipConstraintChecks) // Require checks

		// Do not expect a ConstraintAPI call for disabled feature flag
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	// Tests that valid leases (>= 2s remaining) are NOT released and are reused as-is.
	// The 2-second buffer is defined in constraints.go: hasValidLease := expiry.After(now.Add(2 * time.Second))
	t.Run("should not acquire lease with valid existing item lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Simulate valid lease (10 seconds in future, well above the 2-second threshold)
		capacityLeaseID := ulid.MustNew(ulid.Timestamp(clock.Now().Add(10*time.Second)), rand.Reader)

		qi.CapacityLease = &osqueue.CapacityLease{
			LeaseID: capacityLeaseID,
		}

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// Wait for any async goroutines to complete (for consistency, though none expected here)
		service.Wait()

		require.NotNil(t, res.CapacityLease)
		require.True(t, res.SkipConstraintChecks)

		// We do not expect any calls to the Constraint API - the valid lease is reused as-is
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("should acquire lease with expired existing item lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		// First, acquire a real lease (this creates it in Redis)
		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, res.CapacityLease)
		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))

		originalLeaseID := res.CapacityLease.LeaseID
		originalLeaseExpiry := originalLeaseID.Timestamp()

		// Advance time to make the lease appear expired
		// The lease has 30s duration (QueueLeaseDuration), and there's a 2s validity buffer
		// So we need to advance past the lease expiry time
		timeToAdvance := originalLeaseExpiry.Sub(clock.Now()) + 5*time.Second
		clock.Advance(timeToAdvance)
		r.SetTime(clock.Now())

		// Set the expired lease on the item
		qi.CapacityLease = res.CapacityLease

		// Call again - should detect expired lease, release it, and acquire a new one
		res2, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// Wait for async goroutines to complete (release happens via service.Go())
		service.Wait()

		require.NotNil(t, res2.CapacityLease)
		require.True(t, res2.SkipConstraintChecks)

		// Expect 2 acquire calls total (initial + after expiry), and 1 release call for the expired lease
		require.Equal(t, 2, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 1, len(cmLifecycles.ReleaseCalls))

		// Verify the released lease ID matches the original (now expired) lease
		require.Equal(t, originalLeaseID, cmLifecycles.ReleaseCalls[0].LeaseID)
	})

	// Tests that near-expiring leases (valid for < 2s) are also released and a new lease is acquired.
	// This covers the edge case where the lease technically hasn't expired but won't be valid long enough.
	// The 2-second buffer is defined in constraints.go: hasValidLease := expiry.After(now.Add(2 * time.Second))
	t.Run("should release near-expiring lease (TTL < 2s) and acquire new one", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		// First, acquire a real lease (this creates it in Redis)
		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, res.CapacityLease)
		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))

		originalLeaseID := res.CapacityLease.LeaseID
		originalLeaseExpiry := originalLeaseID.Timestamp()

		// Advance time so the lease has less than 2 seconds remaining
		// (e.g., advance to 1 second before the lease ULID timestamp)
		// Since ULID timestamps are based on creation time, we advance to 1 second before
		// the original lease expiry to make it appear near-expiring
		timeToAdvance := originalLeaseExpiry.Sub(clock.Now()) - 1*time.Second
		clock.Advance(timeToAdvance)
		r.SetTime(clock.Now())

		// Verify the lease now has less than 2 seconds remaining
		ttlRemaining := originalLeaseExpiry.Sub(clock.Now())
		require.Less(t, ttlRemaining, 2*time.Second, "Lease should have less than 2 seconds remaining")
		require.Greater(t, ttlRemaining, time.Duration(0), "Lease should still be technically valid (positive TTL)")

		// Set the near-expiring lease on the item
		qi.CapacityLease = res.CapacityLease

		// Call again - should detect near-expiring lease, release it, and acquire a new one
		res2, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// Wait for async goroutines to complete (release happens via service.Go())
		service.Wait()

		require.NotNil(t, res2.CapacityLease)
		require.True(t, res2.SkipConstraintChecks)

		// Expect 2 acquire calls total (initial + after near-expiry), and 1 release call
		require.Equal(t, 2, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 1, len(cmLifecycles.ReleaseCalls))

		// Verify the released lease ID matches the near-expiring lease
		require.Equal(t, originalLeaseID, cmLifecycles.ReleaseCalls[0].LeaseID)
	})

	t.Run("acquire lease from constraint api", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		require.NotNil(t, res.CapacityLease)
		require.True(t, res.SkipConstraintChecks)

		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("successful acquire has no limiting constraint", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		// Leases were granted
		require.NotNil(t, res.CapacityLease)
		require.True(t, res.SkipConstraintChecks)

		// CRITICAL: No limiting constraint should be set when leases are granted
		require.Equal(t, enums.QueueConstraintNotLimited, res.LimitingConstraint)

		// Verify lifecycle data
		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)

		// ExhaustedConstraints should be empty since capacity was available
		require.Empty(t, cmLifecycles.AcquireCalls[0].ExhaustedConstraints)
	})

	t.Run("multiple constraints with partial exhaustion", func(t *testing.T) {
		reset()

		// Set up multiple constraints: account concurrency=10, function concurrency=2
		// We'll exhaust function concurrency but leave account concurrency available
		multiConstraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 1,
			Concurrency: osqueue.PartitionConcurrency{
				AccountConcurrency:  10, // Not exhausted
				FunctionConcurrency: 2,  // Will be exhausted
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return multiConstraints
			}),
		)

		// Create three items
		item1 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload1\"}"),
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
				},
			},
		}
		item2 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload2\"}"),
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
				},
			},
		}
		item3 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload3\"}"),
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
				},
			},
		}

		qi1, err := shard.EnqueueItem(ctx, item1, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		qi2, err := shard.EnqueueItem(ctx, item2, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		qi3, err := shard.EnqueueItem(ctx, item3, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi1)
		backlog := osqueue.ItemBacklog(ctx, qi1)

		// First two acquires should succeed (function concurrency limit=2)
		res1, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, multiConstraints, &qi1, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, res1.CapacityLease)
		require.True(t, res1.SkipConstraintChecks)

		backlog2 := osqueue.ItemBacklog(ctx, qi2)
		res2, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog2, multiConstraints, &qi2, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, res2.CapacityLease)
		require.True(t, res2.SkipConstraintChecks)

		// The second acquire should show function concurrency is exhausted
		require.Equal(t, 2, len(cmLifecycles.AcquireCalls))
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].ExhaustedConstraints, 1)

		// Third acquire should fail due to function concurrency exhaustion
		backlog3 := osqueue.ItemBacklog(ctx, qi3)
		res3, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog3, multiConstraints, &qi3, clock.Now())
		require.NoError(t, err)
		require.Nil(t, res3.CapacityLease)
		require.False(t, res3.SkipConstraintChecks)
		require.Equal(t, enums.QueueConstraintFunctionConcurrency, res3.LimitingConstraint)

		// Verify lifecycle captures exhausted constraint
		require.Equal(t, 3, len(cmLifecycles.AcquireCalls))
		require.Len(t, cmLifecycles.AcquireCalls[2].GrantedLeases, 0)
		require.Len(t, cmLifecycles.AcquireCalls[2].ExhaustedConstraints, 1)

		// The exhausted constraint should be function concurrency
		exhausted := cmLifecycles.AcquireCalls[2].ExhaustedConstraints[0]
		require.Equal(t, constraintapi.ConstraintKindConcurrency, exhausted.Kind)
		require.Equal(t, enums.ConcurrencyScopeFn, exhausted.Concurrency.Scope)
	})

	t.Run("lacking constraint capacity", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		// Simulate in progress leases
		keyInProgressLeases := constraintapi.ConcurrencyConstraint{
			Scope: enums.ConcurrencyScopeAccount,
		}.InProgressLeasesKey(accountID, envID, fnID)
		for i := range 10 {
			_, err := r.ZAdd(
				keyInProgressLeases,
				float64(clock.Now().Add(5*time.Second).UnixMilli()),
				fmt.Sprintf("i%d", i),
			)
			require.NoError(t, err)
		}

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		require.Equal(t, enums.QueueConstraintAccountConcurrency, res.LimitingConstraint)
		require.Nil(t, res.CapacityLease)
		require.False(t, res.SkipConstraintChecks)

		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))

		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 0)
		require.Len(t, cmLifecycles.AcquireCalls[0].LimitingConstraints, 1)
		require.Equal(t, constraintapi.ConstraintKindConcurrency, cmLifecycles.AcquireCalls[0].LimitingConstraints[0].Kind)

		// Verify ExhaustedConstraints is populated
		require.Len(t, cmLifecycles.AcquireCalls[0].ExhaustedConstraints, 1)
		require.Equal(t, constraintapi.ConstraintKindConcurrency,
			cmLifecycles.AcquireCalls[0].ExhaustedConstraints[0].Kind)
		require.Equal(t, enums.ConcurrencyScopeAccount,
			cmLifecycles.AcquireCalls[0].ExhaustedConstraints[0].Concurrency.Scope)

		// Verify that the exhausted constraint matches the limiting constraint
		require.Equal(t,
			cmLifecycles.AcquireCalls[0].ExhaustedConstraints[0].Kind,
			cmLifecycles.AcquireCalls[0].LimitingConstraints[0].Kind)
	})
}

func TestBacklogRefillConstraintCheck(t *testing.T) {
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
		constraintapi.WithShardName("default"),
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

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	start := clock.Now()

	t.Run("skip constraintapi but require checks when missing identifier", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
		require.Error(t, err)
		require.ErrorContains(t, err, "missing accountID")
		require.ErrorContains(t, err, "missing envID")

		// No lease acquired
		require.Nil(t, res)

		// Do not expect a ConstraintAPI call for missing identifiers
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("skip constraintapi but require checks without capacity manager", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.ItemCapacityLeases)
		require.False(t, res.SkipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing capacity manager
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("skip constraintapi but require checks with disabled feature flag", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return false
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.ItemCapacityLeases)
		require.False(t, res.SkipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing capacity manager
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("acquire leases from constraintapi", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// Acquired lease and request to skip checks
		require.Len(t, res.ItemCapacityLeases, 1)
		require.Len(t, res.ItemsToRefill, 1)
		require.Equal(t, qi.ID, res.ItemsToRefill[0])
		require.True(t, res.SkipConstraintChecks)

		// Expect exactly one acquire request
		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))

		// Verify lifecycle data for successful acquire
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Empty(t, cmLifecycles.AcquireCalls[0].ExhaustedConstraints,
			"ExhaustedConstraints should be empty when leases are granted")
	})

	t.Run("lacking capacity returns 0 leases from constraintapi", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		// Simulate in progress leases
		keyInProgressLeases := constraintapi.ConcurrencyConstraint{
			Scope: enums.ConcurrencyScopeAccount,
		}.InProgressLeasesKey(accountID, envID, fnID)
		for i := range 10 {
			_, err := r.ZAdd(
				keyInProgressLeases,
				float64(clock.Now().Add(5*time.Second).UnixMilli()),
				fmt.Sprintf("i%d", i),
			)
			require.NoError(t, err)
		}

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := osqueue.ItemShadowPartition(ctx, qi)
		backlog := osqueue.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// Acquired lease and request to skip checks
		require.Len(t, res.ItemCapacityLeases, 0)
		require.False(t, res.SkipConstraintChecks)
		require.Equal(t, enums.QueueConstraintAccountConcurrency, res.LimitingConstraint)

		// Expect exactly one acquire request
		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))

		// Verify ExhaustedConstraints is populated
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 0)
		require.Len(t, cmLifecycles.AcquireCalls[0].ExhaustedConstraints, 1)
		require.Equal(t, constraintapi.ConstraintKindConcurrency,
			cmLifecycles.AcquireCalls[0].ExhaustedConstraints[0].Kind)
	})
}
