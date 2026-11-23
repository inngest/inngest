package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestKeyQueueRefillPause(t *testing.T) {
	for _, kind := range []string{KindStart, KindEdge} {
		t.Run(kind, func(t *testing.T) {
			ctx := t.Context()
			clock := clockwork.NewFakeClock()
			accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
			sp := &QueueShadowPartition{PartitionID: fnID.String(), AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}
			item := QueueItem{ID: "item", FunctionID: fnID, WorkspaceID: envID, Data: Item{
				Kind: kind, Identifier: state.Identifier{AccountID: accountID, WorkspaceID: envID, WorkflowID: fnID, RunID: ulid.Make()},
			}}
			backlog := ItemBacklog(ctx, item)
			shard := &refillPauseTestShard{mockShardForIterator: &mockShardForIterator{name: "test"}, backlog: &backlog, item: &item}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			manager := &refillPauseCapacityManager{}
			paused := false
			q, err := New(ctx, "test", registry, WithClock(clock),
				WithRunMode(QueueRunMode{ShadowContinuations: true}),
				WithCapacityManager(manager), WithAcquireCapacityLeaseOnBacklogRefill(true),
				WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
				WithAllowKeyQueueRefill(func(_ context.Context, acct, env, fn uuid.UUID) bool {
					require.Equal(t, accountID, acct)
					require.Equal(t, envID, env)
					require.Equal(t, fnID, fn)
					return !paused
				}),
			)
			require.NoError(t, err)

			for _, wantPaused := range []bool{false, true, false} {
				paused = wantPaused
				acquired, refilled := manager.acquires, shard.refills
				q.AddShadowContinue(ctx, sp, 1)
				require.NoError(t, q.ProcessShadowPartition(ctx, sp, 1))
				if paused {
					require.Equal(t, acquired, manager.acquires, "pause must precede capacity acquisition")
					require.Equal(t, refilled, shard.refills)
					require.Equal(t, clock.Now().Add(BacklogDefaultRequeueExtension), *shard.shadowPartitionRequeueAt)
					require.Empty(t, q.GetShadowContinuations())
				} else {
					require.Equal(t, acquired+1, manager.acquires)
					require.Equal(t, refilled+1, shard.refills)
				}
			}

			// Pausing refill does not gate ready-item admission or dispatch.
			paused = true
			dispatched := false
			res, err := q.LeaseItem(ctx, LeaseItemRequest{Item: &item, StaticTime: clock.Now()}, func(context.Context, ProcessItem) (DispatchedItem, error) {
				dispatched = true
				return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
			})
			require.NoError(t, err)
			require.Equal(t, LeaseItemStatusDispatched, res.Status)
			require.True(t, dispatched)

			// Discovering outdated backlogs must continue during the pause.
			backlog.Throttle = &BacklogThrottle{ThrottleKey: "old-throttle"}
			resBacklog, _, err := q.ProcessShadowPartitionBacklog(ctx, sp, &backlog, clock.Now(), PartitionConstraintConfig{})
			require.NoError(t, err)
			require.Nil(t, resBacklog)
			require.Equal(t, 1, shard.normalizations)
		})
	}
}

func TestKeyQueueRefillAllowed(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	system := "system"
	for _, tt := range []struct {
		name string
		sp   QueueShadowPartition
		opts []QueueOpt
		want bool
	}{
		{name: "allowed by default", sp: QueueShadowPartition{AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}, want: true},
		{name: "nil callback", sp: QueueShadowPartition{AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}, opts: []QueueOpt{WithAllowKeyQueueRefill(nil)}, want: true},
		{name: "paused", sp: QueueShadowPartition{AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}},
		{name: "system queue bypasses callback", sp: QueueShadowPartition{SystemQueueName: &system, AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}, want: true},
		{name: "missing account", sp: QueueShadowPartition{EnvID: &envID, FunctionID: &fnID}, want: true},
		{name: "missing environment", sp: QueueShadowPartition{AccountID: &accountID, FunctionID: &fnID}, want: true},
		{name: "missing function", sp: QueueShadowPartition{AccountID: &accountID, EnvID: &envID}, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := []QueueOpt{}
			if tt.name != "allowed by default" {
				opts = append(opts, WithAllowKeyQueueRefill(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return false }))
			}
			opts = append(opts, tt.opts...)
			require.Equal(t, tt.want, tt.sp.KeyQueueRefillAllowed(t.Context(), NewQueueOptions(opts...)))
		})
	}
}

type refillPauseTestShard struct {
	*mockShardForIterator
	backlog        *QueueBacklog
	item           *QueueItem
	refills        int
	normalizations int
}

func (s *refillPauseTestShard) ShadowPartitionPeek(context.Context, *QueueShadowPartition, bool, time.Time, int64, ...PeekOpt) ([]*QueueBacklog, int, error) {
	return []*QueueBacklog{s.backlog}, 1, nil
}

func (s *refillPauseTestShard) BacklogPeek(context.Context, *QueueBacklog, time.Time, time.Time, int64, ...PeekOpt) (*BacklogPeekResult, error) {
	return &BacklogPeekResult{Items: []*QueueItem{s.item}, TotalCount: 1}, nil
}

func (s *refillPauseTestShard) BacklogRefill(_ context.Context, _ *QueueBacklog, _ *QueueShadowPartition, _ time.Time, items []string, _ ...BacklogRefillOptionFn) (*BacklogRefillResult, error) {
	s.refills++
	return &BacklogRefillResult{TotalBacklogCount: 1, RefilledItems: items}, nil
}

func (s *refillPauseTestShard) BacklogPrepareNormalize(context.Context, *QueueBacklog, *QueueShadowPartition) error {
	s.normalizations++
	return nil
}

type refillPauseCapacityManager struct {
	constraintapi.CapacityManager
	acquires int
}

func (m *refillPauseCapacityManager) Acquire(_ context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	m.acquires++
	keys := req.LeaseIdempotencyKeys
	if len(keys) == 0 {
		keys = []string{req.IdempotencyKey}
	}
	res := &constraintapi.CapacityAcquireResponse{}
	for _, key := range keys {
		res.Leases = append(res.Leases, constraintapi.CapacityLease{
			LeaseID: ulid.MustNew(ulid.Timestamp(req.CurrentTime.Add(req.Duration)), nil), IdempotencyKey: key,
		})
	}
	return res, nil
}

var _ QueueShard = (*refillPauseTestShard)(nil)
var _ constraintapi.CapacityManager = (*refillPauseCapacityManager)(nil)
