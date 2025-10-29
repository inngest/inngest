package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// TestIncrWorkerRequestsLuaScript tests the incr_worker_requests.lua script directly
func TestIncrWorkerRequestsLuaScript(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance"

	// Helper to get Redis keys
	capacityKey := fmt.Sprintf("{%s}:worker-capacity:%s", envID.String(), instanceID)
	workerLeasesKey := fmt.Sprintf("{%s}:worker-leases-set:%s", envID.String(), instanceID)
	leaseWorkerKey := func(requestID string) string {
		return fmt.Sprintf("{%s}:lease-worker:%s", envID.String(), requestID)
	}

	t.Run("returns 0 when no capacity limit set", func(t *testing.T) {
		// No capacity key exists
		requestID := "req-1"

		counterTTL := consts.ConnectWorkerInformationDuration
		now := time.Now()
		expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64(counterTTL.Seconds())),
			instanceID,
			requestID,
			fmt.Sprintf("%d", expirationTime),
			fmt.Sprintf("%d", now.UnixMilli()),
		}

		result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(0), result, "should return 0 when no capacity set")

		// Verify no set was created
		require.False(t, r.Exists(workerLeasesKey), "set should not be created when no capacity limit")
		require.False(t, r.Exists(leaseWorkerKey(requestID)), "lease mapping should not be created when no capacity limit")
	})

	t.Run("returns 0 when capacity is zero", func(t *testing.T) {
		// Set capacity to 0 (unlimited)
		r.Set(capacityKey, "0")
		requestID := "req-2"

		now := time.Now()
		expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			instanceID,
			requestID,
			fmt.Sprintf("%d", expirationTime),
			fmt.Sprintf("%d", now.UnixMilli()),
		}

		result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(0), result, "should return 0 when capacity is 0")

		// Clean up
		r.Del(capacityKey)
	})

	t.Run("successfully increments when under capacity", func(t *testing.T) {
		// Set capacity to 5
		r.Set(capacityKey, "5")
		requestID := "req-3"

		now := time.Now()
		expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			instanceID,
			requestID,
			fmt.Sprintf("%d", expirationTime),
			fmt.Sprintf("%d", now.UnixMilli()),
		}

		result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(0), result, "should successfully increment")

		// Verify request was added to set
		setMembers, err := r.ZMembers(workerLeasesKey)
		require.NoError(t, err)
		require.Len(t, setMembers, 1)
		require.Equal(t, requestID, setMembers[0])

		// Verify TTL is set on set
		require.True(t, r.Exists(workerLeasesKey))
		ttl := r.TTL(workerLeasesKey)
		require.Greater(t, ttl, time.Duration(0))

		// Verify lease-to-worker mapping was created
		mappedInstance, err := r.Get(leaseWorkerKey(requestID))
		require.NoError(t, err)
		require.Equal(t, instanceID, mappedInstance)

		// Verify TTL on mapping
		require.True(t, r.Exists(leaseWorkerKey(requestID)))
		mappingTTL := r.TTL(leaseWorkerKey(requestID))
		require.Greater(t, mappingTTL, time.Duration(0))

		// Clean up
		r.Del(capacityKey)
		r.Del(workerLeasesKey)
		r.Del(leaseWorkerKey(requestID))
	})

	t.Run("increments counter multiple times", func(t *testing.T) {
		// Set capacity to 5
		r.Set(capacityKey, "5")

		for i := 1; i <= 3; i++ {
			requestID := fmt.Sprintf("req-%d", i)
			keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
			now := time.Now()
			expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			args := []string{
				fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
				instanceID,
				requestID,
				fmt.Sprintf("%d", expirationTime),
				fmt.Sprintf("%d", now.UnixMilli()),
			}

			result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
			require.NoError(t, err)
			require.Equal(t, int64(0), result, "should successfully increment on iteration %d", i)

			// Verify request was added to set
			setMembers, err := r.ZMembers(workerLeasesKey)
			require.NoError(t, err)
			require.Len(t, setMembers, i, "set should have %d members", i)
		}

		// Clean up
		r.Del(capacityKey)
		r.Del(workerLeasesKey)
		for i := 1; i <= 3; i++ {
			r.Del(leaseWorkerKey(fmt.Sprintf("req-%d", i)))
		}
	})

	t.Run("returns 1 when at capacity", func(t *testing.T) {
		// Set capacity to 2 and fill it with 2 existing requests
		r.Set(capacityKey, "2")
		for i := 1; i <= 2; i++ {
			existingReqID := fmt.Sprintf("existing-req-%d", i)
			existingExpTime := time.Now().Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			r.ZAdd(workerLeasesKey, float64(existingExpTime), existingReqID)
		}

		requestID := "req-overflow"
		now := time.Now()
		expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			instanceID,
			requestID,
			fmt.Sprintf("%d", expirationTime),
			fmt.Sprintf("%d", now.UnixMilli()),
		}

		result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(1), result, "should return 1 when at capacity")

		// Verify set still has only 2 members
		setMembers, err := r.ZMembers(workerLeasesKey)
		require.NoError(t, err)
		require.Len(t, setMembers, 2, "set should still have 2 members")
		require.NotContains(t, setMembers, requestID, "overflow request should not be in set")

		// Verify no mapping was created
		require.False(t, r.Exists(leaseWorkerKey(requestID)))

		// Clean up
		r.Del(capacityKey)
		r.Del(workerLeasesKey)
	})
}

// TestDecrWorkerRequestsLuaScript tests the decr_worker_requests.lua script directly
func TestDecrWorkerRequestsLuaScript(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance"

	// Helper to get Redis keys
	workerLeasesKey := fmt.Sprintf("{%s}:worker-leases-set:%s", envID.String(), instanceID)
	leaseWorkerKey := func(requestID string) string {
		return fmt.Sprintf("{%s}:lease-worker:%s", envID.String(), requestID)
	}

	t.Run("returns 2 when set doesn't exist", func(t *testing.T) {
		// No set exists
		requestID := "req-nonexistent"

		keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			requestID,
			instanceID,
		}

		result, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(2), result, "should return 2 when set doesn't exist")
	})

	t.Run("returns 0 and deletes set when removing last member", func(t *testing.T) {
		// Set up set with one member
		requestID := "req-last"
		expTime := time.Now().Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		r.ZAdd(workerLeasesKey, float64(expTime), requestID)
		r.Set(leaseWorkerKey(requestID), instanceID)

		keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			requestID,
			instanceID,
		}

		result, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(0), result, "should return 0 when set becomes empty")

		// Verify set was deleted
		require.False(t, r.Exists(workerLeasesKey), "set should be deleted when empty")

		// Verify lease-to-worker mapping was deleted
		require.False(t, r.Exists(leaseWorkerKey(requestID)), "lease mapping should be deleted")
	})

	t.Run("returns 1 and refreshes TTL when set still has members", func(t *testing.T) {
		// Set up set with multiple members
		for i := 1; i <= 3; i++ {
			reqID := fmt.Sprintf("req-%d", i)
			expTime := time.Now().Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			r.ZAdd(workerLeasesKey, float64(expTime), reqID)
			r.Set(leaseWorkerKey(reqID), instanceID)
		}

		// Fast forward time to simulate TTL decay
		r.FastForward(30 * time.Second)

		requestID := "req-1"
		keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			requestID,
			instanceID,
		}

		result, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(1), result, "should return 1 when set still has members")

		// Verify request was removed from set
		setMembers, err := r.ZMembers(workerLeasesKey)
		require.NoError(t, err)
		require.Len(t, setMembers, 2, "set should have 2 remaining members")
		require.NotContains(t, setMembers, requestID, "removed request should not be in set")

		// Verify TTL was refreshed
		ttl := r.TTL(workerLeasesKey)
		expectedTTL := consts.ConnectWorkerInformationDuration
		require.Greater(t, ttl, expectedTTL-5*time.Second)

		// Verify lease mapping was deleted
		require.False(t, r.Exists(leaseWorkerKey(requestID)))

		// Clean up
		r.Del(workerLeasesKey)
		for i := 2; i <= 3; i++ {
			r.Del(leaseWorkerKey(fmt.Sprintf("req-%d", i)))
		}
	})

	t.Run("removes specific request from set", func(t *testing.T) {
		// Set up set with multiple members
		requestIDs := []string{"req-a", "req-b", "req-c"}
		for _, reqID := range requestIDs {
			expTime := time.Now().Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			r.ZAdd(workerLeasesKey, float64(expTime), reqID)
			r.Set(leaseWorkerKey(reqID), instanceID)
		}

		// Remove middle request
		requestID := "req-b"
		keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			requestID,
			instanceID,
		}

		result, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(1), result)

		// Verify specific request was removed
		setMembers, err := r.ZMembers(workerLeasesKey)
		require.NoError(t, err)
		require.Len(t, setMembers, 2)
		require.Contains(t, setMembers, "req-a")
		require.Contains(t, setMembers, "req-c")
		require.NotContains(t, setMembers, "req-b")

		// Clean up
		r.Del(workerLeasesKey)
		for _, reqID := range requestIDs {
			r.Del(leaseWorkerKey(reqID))
		}
	})

	t.Run("returns 3 when instance ID doesn't match", func(t *testing.T) {
		// Set up set with one member
		requestID := "req-mismatch"
		expTime := time.Now().Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		r.ZAdd(workerLeasesKey, float64(expTime), requestID)
		r.Set(leaseWorkerKey(requestID), "other-instance")

		keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			requestID,
			instanceID, // Different from "other-instance"
		}

		result, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(3), result, "should return 3 when instance ID doesn't match")

		// Verify the lease wasn't removed
		setMembers, err := r.ZMembers(workerLeasesKey)
		require.NoError(t, err)
		require.Equal(t, []string{requestID}, setMembers, "lease should not be removed")

		// Verify mapping still exists
		require.True(t, r.Exists(leaseWorkerKey(requestID)))

		// Clean up
		r.Del(workerLeasesKey)
		r.Del(leaseWorkerKey(requestID))
	})
}

// TestWorkerRequestsLuaScriptsIntegration tests both scripts working together
func TestWorkerRequestsLuaScriptsIntegration(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance"

	capacityKey := fmt.Sprintf("{%s}:worker-capacity:%s", envID.String(), instanceID)
	workerLeasesKey := fmt.Sprintf("{%s}:worker-leases-set:%s", envID.String(), instanceID)
	leaseWorkerKey := func(requestID string) string {
		return fmt.Sprintf("{%s}:lease-worker:%s", envID.String(), requestID)
	}

	t.Run("complete lifecycle: incr to capacity then decr back to zero", func(t *testing.T) {
		// Set capacity to 3
		r.Set(capacityKey, "3")

		// Increment to capacity
		for i := 1; i <= 3; i++ {
			requestID := fmt.Sprintf("req-%d", i)
			keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
			now := time.Now()
			expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			args := []string{
				fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
				instanceID,
				requestID,
				fmt.Sprintf("%d", expirationTime),
				fmt.Sprintf("%d", now.UnixMilli()),
			}

			result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
			require.NoError(t, err)
			require.Equal(t, int64(0), result, "increment %d should succeed", i)
		}

		// Verify at capacity
		setMembers, err := r.ZMembers(workerLeasesKey)
		require.NoError(t, err)
		require.Len(t, setMembers, 3, "set should have 3 members")

		// Try to exceed capacity
		now := time.Now()
		expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
		keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey("req-overflow")}
		args := []string{
			fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
			instanceID,
			"req-overflow",
			fmt.Sprintf("%d", expirationTime),
			fmt.Sprintf("%d", now.UnixMilli()),
		}

		result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
		require.NoError(t, err)
		require.Equal(t, int64(1), result, "should reject when at capacity")

		// Decrement all requests
		for i := 3; i >= 1; i-- {
			requestID := fmt.Sprintf("req-%d", i)
			keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
			args := []string{
				fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
				requestID,
				instanceID,
			}

			result, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
			require.NoError(t, err)

			if i == 1 {
				require.Equal(t, int64(0), result, "last decrement should return 0")
			} else {
				require.Equal(t, int64(1), result, "decrement should return 1")
			}
		}

		// Verify set is deleted
		require.False(t, r.Exists(workerLeasesKey))

		// Clean up
		r.Del(capacityKey)
	})

	t.Run("can increment again after full cycle", func(t *testing.T) {
		// Set capacity
		r.Set(capacityKey, "2")

		// First cycle: increment and decrement
		for i := 1; i <= 2; i++ {
			requestID := fmt.Sprintf("req-cycle1-%d", i)
			keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
			now := time.Now()
			expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			args := []string{
				fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
				instanceID,
				requestID,
				fmt.Sprintf("%d", expirationTime),
				fmt.Sprintf("%d", now.UnixMilli()),
			}
			_, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
			require.NoError(t, err)
		}

		// Decrement all
		for i := 2; i >= 1; i-- {
			requestID := fmt.Sprintf("req-cycle1-%d", i)
			keys := []string{workerLeasesKey, leaseWorkerKey(requestID)}
			args := []string{
				fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
				requestID,
				instanceID,
			}
			_, err := scripts["decr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
			require.NoError(t, err)
		}

		// Second cycle: should work again
		for i := 1; i <= 2; i++ {
			requestID := fmt.Sprintf("req-cycle2-%d", i)
			keys := []string{capacityKey, workerLeasesKey, leaseWorkerKey(requestID)}
			now := time.Now()
			expirationTime := now.Add(consts.ConnectWorkerRequestLeaseDuration).Unix()
			args := []string{
				fmt.Sprintf("%d", int64((consts.ConnectWorkerInformationDuration).Seconds())),
				instanceID,
				requestID,
				fmt.Sprintf("%d", expirationTime),
				fmt.Sprintf("%d", now.UnixMilli()),
			}
			result, err := scripts["incr_worker_requests"].Exec(ctx, rc, keys, args).AsInt64()
			require.NoError(t, err)
			require.Equal(t, int64(0), result, "second cycle increment %d should succeed", i)
		}

		// Clean up
		r.Del(capacityKey)
		r.Del(workerLeasesKey)
		for i := 1; i <= 2; i++ {
			r.Del(leaseWorkerKey(fmt.Sprintf("req-cycle2-%d", i)))
		}
	})
}
