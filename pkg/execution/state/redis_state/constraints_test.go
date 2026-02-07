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
	"github.com/stretchr/testify/assert"
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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
		res2, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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
		res2, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
		require.NoError(t, err)

		require.NotNil(t, res.CapacityLease)
		require.True(t, res.SkipConstraintChecks)

		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("successful acquire has no limiting constraint", func(t *testing.T) {
		reset()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res1, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, multiConstraints, &qi1, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, res1.CapacityLease)
		require.True(t, res1.SkipConstraintChecks)

		backlog2 := osqueue.ItemBacklog(ctx, qi2)
		res2, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog2, multiConstraints, &qi2, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, res2.CapacityLease)
		require.True(t, res2.SkipConstraintChecks)

		// The second acquire should show function concurrency is exhausted
		require.Equal(t, 2, len(cmLifecycles.AcquireCalls))
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].ExhaustedConstraints, 1)

		// Third acquire should fail due to function concurrency exhaustion
		backlog3 := osqueue.ItemBacklog(ctx, qi3)
		res3, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog3, multiConstraints, &qi3, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

		res, err := shard.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.ItemCapacityLeases)
		require.False(t, res.SkipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing identifiers
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("skip constraintapi but require checks without capacity manager", func(t *testing.T) {
		reset()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
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

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
		res, err := shard.BacklogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, opIdempotencyKey, clock.Now())
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

func TestConstraintConfigFromConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints osqueue.PartitionConstraintConfig
		expected    constraintapi.ConstraintConfig
	}{
		{
			name:        "empty constraints",
			constraints: osqueue.PartitionConstraintConfig{},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 0,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     0,
					FunctionConcurrency:    0,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
				},
			},
		},
		{
			name: "basic concurrency limits",
			constraints: osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:     100,
					FunctionConcurrency:    10,
					AccountRunConcurrency:  50,
					FunctionRunConcurrency: 5,
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     100,
					FunctionConcurrency:    10,
					AccountRunConcurrency:  50,
					FunctionRunConcurrency: 5,
				},
			},
		},
		{
			name: "with custom concurrency keys",
			constraints: osqueue.PartitionConstraintConfig{
				FunctionVersion: 2,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  100,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               5,
							HashedKeyExpression: "key1-hash",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							Limit:               3,
							HashedKeyExpression: "key2-hash",
						},
					},
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 2,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  100,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             5,
							KeyExpressionHash: "key1-hash",
						},
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							Limit:             3,
							KeyExpressionHash: "key2-hash",
						},
					},
				},
			},
		},
		{
			name: "with throttle",
			constraints: osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Throttle: &osqueue.PartitionThrottle{
					Limit:                     10,
					Burst:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     0,
					FunctionConcurrency:    0,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:             10,
						Burst:             5,
						Period:            60,
						KeyExpressionHash: "throttle-hash",
					},
				},
			},
		},
		{
			name: "complete configuration",
			constraints: osqueue.PartitionConstraintConfig{
				FunctionVersion: 3,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:     200,
					FunctionConcurrency:    20,
					AccountRunConcurrency:  100,
					FunctionRunConcurrency: 10,
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               15,
							HashedKeyExpression: "custom-key-hash",
						},
					},
				},
				Throttle: &osqueue.PartitionThrottle{
					Limit:                     20,
					Burst:                     10,
					Period:                    30,
					ThrottleKeyExpressionHash: "complete-throttle-hash",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 3,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     200,
					FunctionConcurrency:    20,
					AccountRunConcurrency:  100,
					FunctionRunConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             15,
							KeyExpressionHash: "custom-key-hash",
						},
					},
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:             20,
						Burst:             10,
						Period:            30,
						KeyExpressionHash: "complete-throttle-hash",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := osqueue.ConstraintConfigFromConstraints(tt.constraints)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintItemsFromBacklog(t *testing.T) {
	accountID, fnID := uuid.New(), uuid.New()
	tests := []struct {
		name     string
		backlog  *osqueue.QueueBacklog
		sp       *osqueue.QueueShadowPartition
		expected []constraintapi.ConstraintItem
	}{
		{
			name: "minimal backlog",
			backlog: &osqueue.QueueBacklog{
				ShadowPartitionID: fnID.String(),
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
		},
		{
			name: "with throttle",
			backlog: &osqueue.QueueBacklog{
				Throttle: &osqueue.BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-expr-hash",
					ThrottleKey:               "throttle-key-value",
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						KeyExpressionHash: "throttle-expr-hash",
						EvaluatedKeyHash:  "throttle-key-value",
					},
				},
			},
		},
		{
			name: "with custom concurrency keys",
			backlog: &osqueue.QueueBacklog{
				ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "custom-key-1-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "custom-key-1-hash",
						HashedValue:         "custom-key-1-value",
					},
					{
						CanonicalKeyID:      fmt.Sprintf("f:%s:%s", fnID, "custom-key-2-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeFn,
						EntityID:            fnID,
						HashedKeyExpression: "custom-key-2-hash",
						HashedValue:         "custom-key-2-value",
					},
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "custom-key-1-hash",
						EvaluatedKeyHash:  "custom-key-1-value",
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "custom-key-2-hash",
						EvaluatedKeyHash:  "custom-key-2-value",
					},
				},
			},
		},
		{
			name: "complete backlog with throttle and concurrency keys",
			backlog: &osqueue.QueueBacklog{
				Throttle: &osqueue.BacklogThrottle{
					ThrottleKeyExpressionHash: "complete-throttle-hash",
					ThrottleKey:               "complete-throttle-value",
				},
				ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("e:%s:%s", fnID, "complete-key-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeEnv,
						EntityID:            fnID,
						HashedKeyExpression: "complete-key-hash",
						HashedValue:         "complete-key-value",
					},
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						KeyExpressionHash: "complete-throttle-hash",
						EvaluatedKeyHash:  "complete-throttle-value",
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeEnv,
						KeyExpressionHash: "complete-key-hash",
						EvaluatedKeyHash:  "complete-key-value",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constraintItemsFromBacklog(tt.sp, tt.backlog, queueKeyGenerator{queueDefaultKey: "q:v1"})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLimitingConstraint(t *testing.T) {
	tests := []struct {
		name                string
		constraints         osqueue.PartitionConstraintConfig
		limitingConstraints []constraintapi.ConstraintItem
		expected            enums.QueueConstraint
	}{
		{
			name:                "no limiting constraints",
			constraints:         osqueue.PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{},
			expected:            enums.QueueConstraintNotLimited,
		},
		{
			name:        "account concurrency constraint",
			constraints: osqueue.PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "",
					},
				},
			},
			expected: enums.QueueConstraintAccountConcurrency,
		},
		{
			name:        "function concurrency constraint",
			constraints: osqueue.PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "",
					},
				},
			},
			expected: enums.QueueConstraintFunctionConcurrency,
		},
		{
			name: "custom concurrency key 1",
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "custom-key-1",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "custom-key-1",
					},
				},
			},
			expected: enums.QueueConstraintCustomConcurrencyKey1,
		},
		{
			name: "custom concurrency key 2",
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "custom-key-1",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							HashedKeyExpression: "custom-key-2",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "custom-key-2",
					},
				},
			},
			expected: enums.QueueConstraintCustomConcurrencyKey2,
		},
		{
			name:        "throttle constraint",
			constraints: osqueue.PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindThrottle,
				},
			},
			expected: enums.QueueConstraintThrottle,
		},
		{
			name:        "multiple constraints - last one wins",
			constraints: osqueue.PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "",
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
				},
			},
			expected: enums.QueueConstraintThrottle,
		},
		{
			name:        "unknown constraint kind",
			constraints: osqueue.PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: "unknown-kind",
				},
			},
			expected: enums.QueueConstraintNotLimited,
		},
		{
			name: "custom concurrency key without matching configuration",
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "different-key",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "non-matching-key",
					},
				},
			},
			expected: enums.QueueConstraintNotLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := osqueue.ConvertLimitingConstraint(tt.constraints, tt.limitingConstraints)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintItemsBacklogToLimitingConstraintRoundTrip(t *testing.T) {
	accountID, fnID := uuid.New(), uuid.New()
	tests := []struct {
		name                    string
		backlog                 *osqueue.QueueBacklog
		sp                      *osqueue.QueueShadowPartition
		constraints             osqueue.PartitionConstraintConfig
		expectedQueueConstraint enums.QueueConstraint
		description             string
	}{
		{
			name:    "account concurrency constraint round trip",
			backlog: &osqueue.QueueBacklog{},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency: 10,
				},
			},
			expectedQueueConstraint: enums.QueueConstraintAccountConcurrency,
			description:             "Account concurrency constraint items should map back to account concurrency queue constraint",
		},
		{
			name:    "function concurrency constraint round trip",
			backlog: &osqueue.QueueBacklog{},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					FunctionConcurrency: 5,
				},
			},
			expectedQueueConstraint: enums.QueueConstraintFunctionConcurrency,
			description:             "Function concurrency constraint items should map back to function concurrency queue constraint",
		},
		{
			name: "throttle constraint round trip",
			backlog: &osqueue.QueueBacklog{
				Throttle: &osqueue.BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-hash",
					ThrottleKey:               "throttle-value",
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Throttle: &osqueue.PartitionThrottle{
					Limit:                     10,
					Burst:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expectedQueueConstraint: enums.QueueConstraintThrottle,
			description:             "Throttle constraint items should map back to throttle queue constraint",
		},
		{
			name: "custom concurrency key 1 round trip",
			backlog: &osqueue.QueueBacklog{
				ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "custom-key-1-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "custom-key-1-hash",
						HashedValue:         "custom-key-1-value",
					},
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               3,
							HashedKeyExpression: "custom-key-1-hash",
						},
					},
				},
			},
			expectedQueueConstraint: enums.QueueConstraintCustomConcurrencyKey1,
			description:             "First custom concurrency key constraint items should map back to custom key 1 queue constraint",
		},
		{
			name: "custom concurrency key 2 round trip",
			backlog: &osqueue.QueueBacklog{
				ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "key-1-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "key-1-hash",
						HashedValue:         "key-1-value",
					},
					{
						CanonicalKeyID:      fmt.Sprintf("f:%s:%s", fnID, "custom-key-2-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeFn,
						EntityID:            fnID,
						HashedKeyExpression: "custom-key-2-hash",
						HashedValue:         "custom-key-2-value",
					},
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               5,
							HashedKeyExpression: "key-1-hash",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							Limit:               2,
							HashedKeyExpression: "custom-key-2-hash",
						},
					},
				},
			},
			expectedQueueConstraint: enums.QueueConstraintCustomConcurrencyKey2,
			description:             "Second custom concurrency key constraint items should map back to custom key 2 queue constraint",
		},
		{
			name: "multiple constraints with throttle taking precedence",
			backlog: &osqueue.QueueBacklog{
				Throttle: &osqueue.BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-hash",
					ThrottleKey:               "throttle-value",
				},
				ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "custom-key-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "custom-key-hash",
						HashedValue:         "custom-key-value",
					},
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency: 100,
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               3,
							HashedKeyExpression: "custom-key-hash",
						},
					},
				},
				Throttle: &osqueue.PartitionThrottle{
					Limit:                     15,
					Burst:                     3,
					Period:                    30,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expectedQueueConstraint: enums.QueueConstraintThrottle,
			description:             "When multiple constraints exist, throttle should take precedence (last one wins)",
		},
		{
			name: "non-matching custom concurrency key should not limit",
			backlog: &osqueue.QueueBacklog{
				ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "different-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "different-hash",
						HashedValue:         "different-value",
					},
				},
			},
			sp: &osqueue.QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               3,
							HashedKeyExpression: "non-matching-hash",
						},
					},
				},
			},
			expectedQueueConstraint: enums.QueueConstraintNotLimited,
			description:             "Custom concurrency keys that don't match configuration should not create limiting constraints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Generate constraint items from the backlog
			constraintItems := constraintItemsFromBacklog(tt.sp, tt.backlog, queueKeyGenerator{queueDefaultKey: "q:v1"})

			// Step 2: Filter the constraint items to find the ones that would be limiting
			// We simulate what the constraint API would return as limiting constraints
			var simulatedLimitingConstraints []constraintapi.ConstraintItem

			// Determine which constraint type we expect to be limiting based on the test case
			switch tt.expectedQueueConstraint {
			case enums.QueueConstraintAccountConcurrency:
				// Only account concurrency would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.Scope == enums.ConcurrencyScopeAccount && item.Concurrency.KeyExpressionHash == "" {
						simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
						break
					}
				}
			case enums.QueueConstraintFunctionConcurrency:
				// Only function concurrency would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.Scope == enums.ConcurrencyScopeFn && item.Concurrency.KeyExpressionHash == "" {
						simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
						break
					}
				}
			case enums.QueueConstraintThrottle:
				// Only throttle would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindThrottle && item.Throttle != nil {
						simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
						break
					}
				}
			case enums.QueueConstraintCustomConcurrencyKey1:
				// Only the first custom concurrency key would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.KeyExpressionHash != "" && item.Concurrency.EvaluatedKeyHash != "" {
						// Check if this matches the first custom concurrency key in the configuration
						if len(tt.constraints.Concurrency.CustomConcurrencyKeys) > 0 {
							expectedKey := tt.constraints.Concurrency.CustomConcurrencyKeys[0]
							if item.Concurrency.Mode == expectedKey.Mode &&
								item.Concurrency.Scope == expectedKey.Scope &&
								item.Concurrency.KeyExpressionHash == expectedKey.HashedKeyExpression {
								simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
								break
							}
						}
					}
				}
			case enums.QueueConstraintCustomConcurrencyKey2:
				// Only the second custom concurrency key would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.KeyExpressionHash != "" && item.Concurrency.EvaluatedKeyHash != "" {
						// Check if this matches the second custom concurrency key in the configuration
						if len(tt.constraints.Concurrency.CustomConcurrencyKeys) > 1 {
							expectedKey := tt.constraints.Concurrency.CustomConcurrencyKeys[1]
							if item.Concurrency.Mode == expectedKey.Mode &&
								item.Concurrency.Scope == expectedKey.Scope &&
								item.Concurrency.KeyExpressionHash == expectedKey.HashedKeyExpression {
								simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
								break
							}
						}
					}
				}
			case enums.QueueConstraintNotLimited:
				// No constraints would be limiting - leave the slice empty
			}

			// Step 3: Convert the limiting constraints back to a queue constraint
			queueConstraint := osqueue.ConvertLimitingConstraint(tt.constraints, simulatedLimitingConstraints)

			// Step 4: Verify the round trip matches expectations
			assert.Equal(t, tt.expectedQueueConstraint, queueConstraint, tt.description)

			// Additional verification: ensure the constraint items contain the expected types
			if tt.expectedQueueConstraint != enums.QueueConstraintNotLimited {
				assert.NotEmpty(t, simulatedLimitingConstraints, "Should have found limiting constraints for non-NotLimited queue constraint")
			}

			// Verify that basic account and function concurrency constraints are always present
			hasAccountConcurrency := false
			hasFunctionConcurrency := false
			for _, item := range constraintItems {
				if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil {
					if item.Concurrency.Scope == enums.ConcurrencyScopeAccount && item.Concurrency.KeyExpressionHash == "" {
						hasAccountConcurrency = true
					}
					if item.Concurrency.Scope == enums.ConcurrencyScopeFn && item.Concurrency.KeyExpressionHash == "" {
						hasFunctionConcurrency = true
					}
				}
			}
			assert.True(t, hasAccountConcurrency, "Should always include account concurrency constraint")
			assert.True(t, hasFunctionConcurrency, "Should always include function concurrency constraint")
		})
	}
}
