package constraintapi

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestExtendWithoutLease(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithClient(rc),
		WithShardName("test"),
		WithClock(clock),
	)
	require.NoError(t, err)
	require.NotNil(t, cm)

	accountID := uuid.New()

	leaseID := ulid.MustNew(ulid.Timestamp(clock.Now().Add(3*time.Second)), rand.Reader)

	// Calling ExtendLease without an existing lease should not return an error but simply an empty response without a new lease
	res, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
		AccountID: accountID,
		Source: LeaseSource{
			Service:  ServiceUnknown,
			Location: CallerLocationUnknown,
		},
		IdempotencyKey: "test",
		LeaseID:        leaseID,
		Duration:       5 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Nil(t, res.LeaseID)

	require.Equal(t, 2, res.internalDebugState.Status)
}
