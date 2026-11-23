package cluster

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// oneSlotSem is a manual release semaphore.  a lease that is released does
// not give its slot back, only ReleaseSemaphore or the sweeper does.
const oneSlotSem = "fn:one-slot"

// oneSlot is a cluster of three nodes with one shard at RF 3, and one
// account whose semaphore has a single slot.
type oneSlot struct {
	c       *testCluster
	cl      *Client
	ctx     context.Context
	account uuid.UUID
	env     uuid.UUID
	fn      uuid.UUID
	// old is the primary at the start, and mgr is its manager.
	old string
	mgr *memory.Manager
}

func newOneSlot(t *testing.T) *oneSlot {
	t.Helper()
	c := newNodeCluster(t, 3, 1, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	s := &oneSlot{
		c:       c,
		cl:      newTestClient(t, c),
		ctx:     ctx,
		account: accountOn(t, c, 0),
		env:     uuid.New(),
		fn:      uuid.New(),
		old:     c.Table().Shards[0].Primary,
	}
	set, err := s.cl.SetCapacity(ctx, s.account, oneSlotSem, "one-slot-capacity", 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, set.Capacity)

	mgr, err := c.node(s.old).Primary(0)
	require.NoError(t, err)
	s.mgr = mgr
	return s
}

// acquire builds a request for the one slot at level all.
func (s *oneSlot) acquire(key string, d time.Duration) *constraintapi.CapacityAcquireRequest {
	sem := constraintapi.SemaphoreConstraint{
		ID:      oneSlotSem,
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseManual,
	}
	return &constraintapi.CapacityAcquireRequest{
		IdempotencyKey: key,
		AccountID:      s.account,
		EnvID:          s.env,
		FunctionID:     s.fn,
		Configuration: constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Semaphores: []constraintapi.Semaphore{
				{
					ID:      sem.ID,
					Weight:  sem.Weight,
					Release: sem.Release,
				},
			},
		},
		Constraints: []constraintapi.ConstraintItem{
			{
				Kind:      constraintapi.ConstraintKindSemaphore,
				Semaphore: &sem,
			},
		},
		Amount:               1,
		LeaseIdempotencyKeys: []string{key + "-lease"},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          time.Now(),
		Duration:             d,
		MaximumLifetime:      time.Hour,
		Source:               ledgerSource(),
		CommitLevel:          constraintapi.CommitLevelAll,
	}
}

func (s *oneSlot) extend(key string, lease ulid.ULID) *constraintapi.CapacityExtendLeaseRequest {
	return &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: key,
		AccountID:      s.account,
		LeaseID:        lease,
		Duration:       30 * time.Second,
		Source:         ledgerSource(),
		CommitLevel:    constraintapi.CommitLevelAll,
	}
}

// usage is the semaphore usage the current primary reports.
func (s *oneSlot) usage(t *testing.T) int64 {
	t.Helper()
	capacity, usage, err := s.cl.GetCapacity(s.ctx, s.account, oneSlotSem, "")
	require.NoError(t, err)
	require.EqualValues(t, 1, capacity)
	return usage
}

// killOld kills the first primary with no drain.  the next call through
// the client is answered by whichever node takes the seat.
func (s *oneSlot) killOld(t *testing.T) {
	t.Helper()
	s.c.Kill(s.old)
}

// TestRetriedAcquireAcrossAFailoverReplaysTheLease takes the only slot of a
// manual release semaphore on a primary that then dies before the caller
// uses the answer.  the retry with the same keys on the new primary returns
// the first lease.  without the replicated idempotency record the retry is
// denied, because the first lease has the slot and nobody knows its ID.
func TestRetriedAcquireAcrossAFailoverReplaysTheLease(t *testing.T) {
	s := newOneSlot(t)
	req := s.acquire("retried-acquire", 30*time.Second)

	first, ierr := s.mgr.Acquire(s.ctx, req)
	require.NoError(t, ierr)
	require.Len(t, first.Leases, 1)
	require.False(t, first.OperationIdempotencyHit)

	s.killOld(t)
	again, err := s.cl.Acquire(s.ctx, req)
	require.NoError(t, err)
	require.NotEqual(t, s.old, s.c.Table().Shards[0].Primary)
	require.True(t, again.OperationIdempotencyHit, "the new primary replayed the grant, table %s", tableString(s.c))
	require.Equal(t, first.Leases, again.Leases)

	// the slot is counted once and the replayed lease is the live one
	require.EqualValues(t, 1, s.usage(t))
	ext, err := s.cl.ExtendLease(s.ctx, s.extend("extend-after-replay", again.Leases[0].LeaseID))
	require.NoError(t, err)
	require.NotNil(t, ext.LeaseID, "the replayed lease did not extend")
}

// TestRetriedExtendAcrossAFailoverReplaysTheLease extends a lease on a
// primary that then dies before the caller uses the answer.  the retry on
// the new primary returns the same new lease ID.  without the replicated
// record the retry finds the old lease ended and returns no lease.
func TestRetriedExtendAcrossAFailoverReplaysTheLease(t *testing.T) {
	s := newOneSlot(t)

	first, ierr := s.mgr.Acquire(s.ctx, s.acquire("extended-lease", 30*time.Second))
	require.NoError(t, ierr)
	require.Len(t, first.Leases, 1)

	req := s.extend("retried-extend", first.Leases[0].LeaseID)
	extended, ierr := s.mgr.ExtendLease(s.ctx, req)
	require.NoError(t, ierr)
	require.NotNil(t, extended.LeaseID)

	s.killOld(t)
	again, err := s.cl.ExtendLease(s.ctx, req)
	require.NoError(t, err)
	require.NotEqual(t, s.old, s.c.Table().Shards[0].Primary)
	require.True(t, again.OperationIdempotencyHit, "the new primary replayed the extend, table %s", tableString(s.c))
	require.NotNil(t, again.LeaseID)
	require.Equal(t, *extended.LeaseID, *again.LeaseID)
	require.Equal(t, s.fn, again.FunctionID)

	// the replayed lease is the live one
	next, err := s.cl.ExtendLease(s.ctx, s.extend("extend-after-replay", *again.LeaseID))
	require.NoError(t, err)
	require.NotNil(t, next.LeaseID, "the replayed lease did not extend")
	require.EqualValues(t, 1, s.usage(t))
}

// TestOrphanLeaseOnManualSemaphoreFreesItsSlot takes the only slot on a
// primary that then dies, and nobody retries.  no caller knows the lease, so
// nobody extends or releases it.  the sweeper of the promoted primary
// reclaims it when it expires and gives the slot back, also for a manual
// release semaphore, and a different request is then granted.
func TestOrphanLeaseOnManualSemaphoreFreesItsSlot(t *testing.T) {
	s := newOneSlot(t)
	const orphanDuration = 4 * time.Second

	grantedAt := time.Now()
	orphan, ierr := s.mgr.Acquire(s.ctx, s.acquire("orphan", orphanDuration))
	require.NoError(t, ierr)
	require.Len(t, orphan.Leases, 1)

	s.killOld(t)
	other := s.acquire("other-request", 30*time.Second)
	denied, err := s.cl.Acquire(s.ctx, other)
	require.NoError(t, err)
	require.NotEqual(t, s.old, s.c.Table().Shards[0].Primary)
	require.Empty(t, denied.Leases, "the orphan lease has the only slot, %s after its grant", time.Since(grantedAt))
	require.EqualValues(t, 1, s.usage(t))

	var granted *constraintapi.CapacityAcquireResponse
	eventually(t, orphanDuration+10*time.Second, func() bool {
		resp, err := s.cl.Acquire(s.ctx, other)
		if err != nil || len(resp.Leases) == 0 {
			time.Sleep(20 * time.Millisecond)
			return false
		}
		granted = resp
		return true
	}, "the slot never came back, table %s", tableString(s.c))

	require.GreaterOrEqual(t, time.Since(grantedAt), orphanDuration, "the slot came back before the orphan lease expired")
	require.NotEqual(t, orphan.Leases[0].LeaseID, granted.Leases[0].LeaseID)
	require.EqualValues(t, 1, s.usage(t))
}

// TestRejoinedFollowerWaitsForOldIdempotencyRecords restarts a follower
// after the primary answered an acquire.  the state transfer gives the
// follower the lease and not the idempotency record, so the follower could
// not replay a retry.  it stays out of the in-sync set, where nobody can
// elect it, until the record has expired on the primary.  an extend made
// after the transfer reaches it through the log, and it replays that one
// when it takes the seat.
func TestRejoinedFollowerWaitsForOldIdempotencyRecords(t *testing.T) {
	s := newOneSlot(t)
	followers := followersOf(s.c.Table().Shards[0])
	rejoined, other := followers[0], followers[1]

	s.c.Kill(rejoined)
	eventually(t, 10*time.Second, func() bool {
		return !slices.Contains(s.c.Table().Shards[0].InSync, rejoined)
	}, "%s did not leave the in-sync set, table %s", rejoined, tableString(s.c))

	before := time.Now()
	first, ierr := s.mgr.Acquire(s.ctx, s.acquire("before-the-transfer", 30*time.Second))
	require.NoError(t, ierr)
	require.Len(t, first.Leases, 1)

	require.NoError(t, s.c.Restart(rejoined))
	eventually(t, 10*time.Second, func() bool {
		return slices.Contains(s.c.Table().Shards[0].InSync, rejoined)
	}, "%s did not enter the in-sync set, table %s", rejoined, tableString(s.c))
	// the record expires a whole TTL after a time the primary read in
	// milliseconds, which is at most one millisecond before the test's
	require.GreaterOrEqual(t, time.Since(before), testIdempotencyTTL-time.Millisecond, "%s entered the in-sync set while the primary still had a record it lacks", rejoined)
	require.Equal(t, s.old, s.c.Table().Shards[0].Primary)

	// only the rejoined follower is left to take the seat
	s.c.Kill(other)
	eventually(t, 10*time.Second, func() bool {
		return !slices.Contains(s.c.Table().Shards[0].InSync, other)
	}, "%s did not leave the in-sync set, table %s", other, tableString(s.c))

	req := s.extend("after-the-transfer", first.Leases[0].LeaseID)
	extended, ierr := s.mgr.ExtendLease(s.ctx, req)
	require.NoError(t, ierr)
	require.NotNil(t, extended.LeaseID)

	s.killOld(t)
	again, err := s.cl.ExtendLease(s.ctx, req)
	require.NoError(t, err)
	require.Equal(t, rejoined, s.c.Table().Shards[0].Primary)
	require.True(t, again.OperationIdempotencyHit, "the new primary replayed the extend, table %s", tableString(s.c))
	require.NotNil(t, again.LeaseID)
	require.Equal(t, *extended.LeaseID, *again.LeaseID)
	require.EqualValues(t, 1, s.usage(t))
}
