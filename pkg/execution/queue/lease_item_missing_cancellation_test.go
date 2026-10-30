package queue

import (
	"context"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type canceledReleaseManager struct {
	constraintapi.CapacityManager
	calls    int
	canceled bool
	fail     bool
}

func (m *canceledReleaseManager) Release(ctx context.Context, _ *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	m.calls++
	m.canceled = ctx.Err() != nil
	if m.fail {
		return nil, errs.Wrap(500, true, "injected missing-item release failure")
	}
	return &constraintapi.CapacityReleaseResponse{}, nil
}
func TestMissingItemReleaseSurvivesCanceledCaller(t *testing.T) {
	for _, fail := range []bool{false, true} {
		cm := &canceledReleaseManager{fail: fail}
		reg, e := NewSingleShardRegistry(goneItemShard{&mockShardForIterator{name: "cd-cancel"}})
		require.NoError(t, e)
		p, e := New(context.Background(), "cd", reg, WithCapacityManager(cm), WithReleaseCapacityOnItemGone(true))
		require.NoError(t, e)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		id := uuid.New()
		input := ProcessItem{I: QueueItem{ID: "missing-" + id.String(), FunctionID: id, Data: Item{Identifier: state.Identifier{AccountID: uuid.New()}}}, CapacityLease: &CapacityLease{LeaseID: ulid.Make(), IssuedAtMS: time.Now().UnixMilli()}}
		// Direct helper test isolates cleanup cancellation; no new Acquire can run on canceled ctx.
		p.releaseCapacityForMissingItem(ctx, &input.I, ItemLeaseConstraintCheckResult{CapacityLease: input.CapacityLease, CapacityLeaseAcquired: true})
		require.Equal(t, 1, cm.calls)
		require.False(t, cm.canceled, "cleanup must outlive canceled caller")
		// Missing-item cleanup intentionally logs an unsuccessful release; it has no inline retry loop.
	}
}
