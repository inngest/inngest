package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Exercise both queue admission paths against the real capacity manager and Lua.
func TestRetryThrottlePerAttempt(t *testing.T) {
	for _, kq := range []bool{false, true} {
		t.Run(fmt.Sprintf("key_queues_%v", kq), func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
			cm, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(rc), constraintapi.WithClock(clock), constraintapi.WithShardName("test"))
			require.NoError(t, err)
			reg, err := NewSingleShardRegistry(&mockShardForIterator{name: "local"})
			require.NoError(t, err)
			q, err := New(ctx, "local", reg, WithCapacityManager(cm), WithAcquireCapacityLeaseOnBacklogRefill(true), WithClock(clock))
			require.NoError(t, err)
			a, e, f, app := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			part := &QueueShadowPartition{PartitionID: f.String(), AccountID: &a, EnvID: &e, FunctionID: &f}
			backlog := &QueueBacklog{BacklogID: "backlog", ShadowPartitionID: f.String(), Throttle: &BacklogThrottle{ThrottleKey: "provider", ThrottleKeyExpressionHash: "provider-expression"}}
			cfg := PartitionConstraintConfig{FunctionVersion: 1, Concurrency: PartitionConcurrency{AccountConcurrency: NoConcurrencyLimit, FunctionConcurrency: NoConcurrencyLimit}, Throttle: &PartitionThrottle{ThrottleKeyExpressionHash: "provider-expression", Limit: 1, Burst: 0, Period: 3600}}
			item := &QueueItem{ID: "same-start-item", Data: Item{Kind: KindStart, Identifier: state.Identifier{RunID: ulid.Make(), AppID: app}}}
			acquire := func() *CapacityLease {
				if kq {
					res, err := q.BacklogRefillConstraintCheck(ctx, part, backlog, cfg, []*QueueItem{item}, fmt.Sprintf("backlog-%d", clock.Now().UnixMilli()), clock.Now(), true)
					require.NoError(t, err)
					if len(res.ItemCapacityLeases) == 0 {
						return nil
					}
					require.Equal(t, []string{item.ID}, res.ItemsToRefill)
					return &res.ItemCapacityLeases[0]
				}
				res, err := q.ItemLeaseConstraintCheck(ctx, part, backlog, cfg, item, clock.Now())
				require.NoError(t, err)
				return res.CapacityLease
			}
			release := func(l *CapacityLease) {
				_, err := cm.Release(ctx, &constraintapi.CapacityReleaseRequest{AccountID: a, LeaseID: l.LeaseID, IdempotencyKey: l.LeaseID.String(), LeaseIssuedAt: time.UnixMilli(l.IssuedAtMS), Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease}})
				require.NoError(t, err)
			}
			first := acquire()
			require.NotNil(t, first)
			// Retrying the same admission RPC must return the existing lease.
			repeated := acquire()
			require.NotNil(t, repeated)
			require.Equal(t, first.LeaseID, repeated.LeaseID)
			release(first)
			for _, gap := range []time.Duration{15 * time.Second, 30 * time.Second, time.Minute, 2 * time.Minute} {
				clock.Advance(gap)
				r.FastForward(gap)
				item.Data.Attempt++
				require.Nil(t, acquire(), "a new execution attempt must check the exhausted throttle")
			}
			// A different item is the positive control: it must be throttled in either mode.
			item.ID = "another-start-item"
			clock.Advance(time.Second)
			r.FastForward(time.Second)
			require.Nil(t, acquire())
		})
	}
}

func TestRetryThrottleRefillDispatch(t *testing.T) {
	for _, attempt := range []int{0, 1} {
		t.Run(fmt.Sprintf("attempt_%d", attempt), func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
			cm, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(rc), constraintapi.WithClock(clock), constraintapi.WithShardName("test"))
			require.NoError(t, err)
			reg, err := NewSingleShardRegistry(&mockShardForIterator{name: "local"})
			require.NoError(t, err)
			q, err := New(ctx, "local", reg, WithCapacityManager(cm), WithAcquireCapacityLeaseOnBacklogRefill(true), WithClock(clock))
			require.NoError(t, err)
			accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
			part := &QueueShadowPartition{PartitionID: fnID.String(), AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}
			backlog := &QueueBacklog{BacklogID: "backlog", ShadowPartitionID: fnID.String(), Start: true, Throttle: &BacklogThrottle{ThrottleKey: "provider", ThrottleKeyExpressionHash: "provider-expression"}}
			cfg := PartitionConstraintConfig{FunctionVersion: 1, Concurrency: PartitionConcurrency{AccountConcurrency: NoConcurrencyLimit, FunctionConcurrency: NoConcurrencyLimit}, Throttle: &PartitionThrottle{ThrottleKeyExpressionHash: "provider-expression", Limit: 1, Burst: 0, Period: 3600}}
			item := &QueueItem{ID: "original-queue-id", Data: Item{Kind: KindStart, Attempt: attempt, Identifier: state.Identifier{RunID: ulid.Make(), AppID: uuid.New()}}}
			refill, err := q.BacklogRefillConstraintCheck(ctx, part, backlog, cfg, []*QueueItem{item}, "refill-operation", clock.Now(), true)
			require.NoError(t, err)
			require.Equal(t, []string{item.ID}, refill.ItemsToRefill, "queue IDs must not include the attempt suffix")
			require.Len(t, refill.ItemCapacityLeases, 1)
			item.CapacityLease = &refill.ItemCapacityLeases[0]
			item.RefilledAt = clock.Now().UnixMilli()
			dispatch, err := q.ItemLeaseConstraintCheck(ctx, part, backlog, cfg, item, clock.Now())
			require.NoError(t, err)
			require.Equal(t, item.CapacityLease, dispatch.CapacityLease)

			// Losing the refill lease before execution must not charge the same
			// attempt again. Both paths need the same attempt-specific marker.
			gap := QueueLeaseDuration + time.Second
			clock.Advance(gap)
			r.FastForward(gap)
			dispatch, err = q.ItemLeaseConstraintCheck(ctx, part, backlog, cfg, item, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, dispatch.CapacityLease)
			require.NotEqual(t, item.CapacityLease.LeaseID, dispatch.CapacityLease.LeaseID)
			oldLeaseID := item.CapacityLease.LeaseID
			require.Eventually(t, func() bool {
				return !r.Exists(fmt.Sprintf("{cs}:a:%s:ld:%s", accountID, oldLeaseID))
			}, time.Second, time.Millisecond, "background cleanup must finish before closing the client")
			_, err = cm.Release(ctx, &constraintapi.CapacityReleaseRequest{AccountID: accountID, LeaseID: dispatch.CapacityLease.LeaseID, IdempotencyKey: dispatch.CapacityLease.LeaseID.String(), LeaseIssuedAt: time.UnixMilli(dispatch.CapacityLease.IssuedAtMS), Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease}})
			require.NoError(t, err)
			item.CapacityLease = nil
			item.Data.Attempt++
			clock.Advance(15 * time.Second)
			r.FastForward(15 * time.Second)
			dispatch, err = q.ItemLeaseConstraintCheck(ctx, part, backlog, cfg, item, clock.Now())
			require.NoError(t, err)
			require.Nil(t, dispatch.CapacityLease, "a new execution attempt must not reuse the refill marker")
		})
	}
}
