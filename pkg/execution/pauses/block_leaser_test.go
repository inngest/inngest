package pauses

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedisBlockLeaser(t *testing.T) (*miniredis.Miniredis, redisBlockLeaser) {
	t.Helper()

	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{s.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	leaser := redisBlockLeaser{
		rc:       client,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	return s, leaser
}

func TestRedisBlockLeaser_Lease(t *testing.T) {
	s, leaser := setupRedisBlockLeaser(t)
	defer s.Close()

	ctx := context.Background()
	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "event1",
	}

	// Test successful lease
	leaseID1, err := leaser.Lease(ctx, index)
	require.NoError(t, err)
	assert.NotEmpty(t, leaseID1)

	// Test that we can't lease again while a lease is active
	_, err = leaser.Lease(ctx, index)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already leased")

	// Test that we can lease after the lease expires
	s.FastForward(6 * time.Second) // Move past the lease duration

	leaseID2, err := leaser.Lease(ctx, index)
	require.NoError(t, err)
	assert.NotEmpty(t, leaseID2)
	assert.NotEqual(t, leaseID1, leaseID2)
}

func TestRedisBlockLeaser_Renew(t *testing.T) {
	s, leaser := setupRedisBlockLeaser(t)
	defer s.Close()

	ctx := context.Background()
	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "event1",
	}

	// Get initial lease
	leaseID1, err := leaser.Lease(ctx, index)
	require.NoError(t, err)

	// Test successful renewal
	leaseID2, err := leaser.Renew(ctx, index, leaseID1)
	require.NoError(t, err)
	assert.NotEqual(t, leaseID1, leaseID2)

	// Test renewal with incorrect lease ID
	wrongID := ulid.MustNew(1, nil)
	_, err = leaser.Renew(ctx, index, wrongID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to renew lease")

	// Test renewal after expiry
	s.FastForward(7 * time.Second)

	_, err = leaser.Renew(ctx, index, leaseID2)
	require.Error(t, err, s.Dump())
	assert.Contains(t, err.Error(), "unable to renew lease")
}

func TestRedisBlockLeaser_Revoke(t *testing.T) {
	s, leaser := setupRedisBlockLeaser(t)
	defer s.Close()

	ctx := context.Background()
	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "event1",
	}

	// Get initial lease
	leaseID1, err := leaser.Lease(ctx, index)
	require.NoError(t, err)

	// Test successful revocation
	err = leaser.Revoke(ctx, index, leaseID1)
	require.NoError(t, err)

	// Verify we can lease again after revocation
	leaseID2, err := leaser.Lease(ctx, index)
	require.NoError(t, err)
	assert.NotEqual(t, leaseID1, leaseID2)

	// Test revocation of non-existent lease
	err = leaser.Revoke(ctx, index, leaseID1)
	require.NoError(t, err) // Should not error as Del is idempotent
}
