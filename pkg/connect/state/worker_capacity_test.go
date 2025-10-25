package state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestSetWorkerTotalCapacity(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance-1"

	t.Run("sets capacity with positive value", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Verify capacity was set
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(10), capacity)

		// Verify TTL is set
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		ttl := r.TTL(capacityKey)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, 4*consts.ConnectWorkerRequestLeaseDuration)
	})

	t.Run("deletes capacity when set to zero", func(t *testing.T) {
		// First set a capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Now set to zero
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 0)
		require.NoError(t, err)

		// Verify capacity is gone
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity)

		// Verify key is deleted
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		require.False(t, r.Exists(capacityKey))
	})

	t.Run("deletes capacity when set to negative", func(t *testing.T) {
		// First set a capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Now set to negative
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, -1)
		require.NoError(t, err)

		// Verify capacity is gone
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity)
	})

	t.Run("updates existing capacity", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Update to different value
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 15)
		require.NoError(t, err)

		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(15), capacity)
	})
}

func TestGetWorkerTotalCapacity(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance-1"

	t.Run("returns zero when no capacity set", func(t *testing.T) {
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity)
	})

	t.Run("returns set capacity", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 25)
		require.NoError(t, err)

		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(25), capacity)
	})
}

func TestGetWorkerCapacities(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance-1"

	t.Run("returns ConnectNoWorkerCapacity when no limit set", func(t *testing.T) {
		available, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, consts.ConnectWorkerNoConcurrencyLimitForRequests, available)
	})

	t.Run("returns full capacity when no active leases", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		available, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(10), available)
	})

	t.Run("returns reduced capacity after assigning leases", func(t *testing.T) {
		instanceID := "test-instance-2"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Assign 3 leases
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		available, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(2), available)
	})

	t.Run("returns zero when at capacity", func(t *testing.T) {
		instanceID := "test-instance-3"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		available, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), available)
	})
}

func TestAssignRequestLeaseToWorker(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("succeeds when no capacity limit set", func(t *testing.T) {
		instanceID := "test-instance-no-limit"
		err := mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Should not create counter when no limit
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		require.False(t, r.Exists(counterKey))
	})

	t.Run("increments counter when capacity set", func(t *testing.T) {
		instanceID := "test-instance-1"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Check counter was incremented
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		require.True(t, r.Exists(counterKey))

		counterVal, err := r.Get(counterKey)
		require.NoError(t, err)
		require.Equal(t, "1", counterVal)
	})

	t.Run("sets TTL on counter", func(t *testing.T) {
		instanceID := "test-instance-ttl"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		ttl := r.TTL(counterKey)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, 4*consts.ConnectWorkerRequestLeaseDuration)
	})

	t.Run("rejects when at capacity", func(t *testing.T) {
		instanceID := "test-instance-full"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Fill capacity
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Should reject third
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)
	})

	t.Run("allows multiple workers with different capacities", func(t *testing.T) {
		instance1 := "worker-1"
		instance2 := "worker-2"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, instance1, 1)
		require.NoError(t, err)
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instance2, 10)
		require.NoError(t, err)

		// Worker 1 at capacity
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instance1, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instance1, "req-2")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)

		// Worker 2 still has capacity
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instance2, "req-1")
		require.NoError(t, err)
	})
}

func TestDeleteRequestLeaseFromWorker(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("no-op when no capacity set", func(t *testing.T) {
		instanceID := "test-instance-no-cap"
		err := mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
	})

	t.Run("decrements counter", func(t *testing.T) {
		instanceID := "test-instance-1"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Add some leases
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Remove one
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Check counter
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		counterVal, err := r.Get(counterKey)
		require.NoError(t, err)
		require.Equal(t, "1", counterVal)
	})

	t.Run("deletes counter when reaching zero", func(t *testing.T) {
		instanceID := "test-instance-2"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Counter should be deleted
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		require.False(t, r.Exists(counterKey))
	})

	t.Run("refreshes TTL when counter still positive", func(t *testing.T) {
		instanceID := "test-instance-3"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Fast forward time a bit in miniredis
		r.FastForward(30 * time.Second)

		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// TTL should be refreshed
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		ttl := r.TTL(counterKey)
		require.Greater(t, ttl, 70*time.Second) // Should be close to 80s
	})

	t.Run("allows assignment after deletion", func(t *testing.T) {
		instanceID := "test-instance-4"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Fill capacity
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Should reject
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)

		// Delete one
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Should now succeed
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)
	})
}

func TestWorkerCapacitiesHeartbeat(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("no-op when no capacity set", func(t *testing.T) {
		instanceID := "test-instance-no-cap"
		err := mgr.WorkerCapacitiesHeartbeat(ctx, envID, instanceID)
		require.NoError(t, err)
	})

	t.Run("refreshes TTL on capacity key", func(t *testing.T) {
		instanceID := "test-instance-1"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Fast forward time
		r.FastForward(40 * time.Second)

		// Refresh TTL
		err = mgr.WorkerCapacitiesHeartbeat(ctx, envID, instanceID)
		require.NoError(t, err)

		// Check TTL is reset
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		ttl := r.TTL(capacityKey)
		require.Greater(t, ttl, 70*time.Second) // Should be close to 80s
	})

	t.Run("refreshes TTL on both capacity and counter keys", func(t *testing.T) {
		instanceID := "test-instance-2"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Assign a lease to create the counter key
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Fast forward time
		r.FastForward(40 * time.Second)

		// Refresh TTL
		err = mgr.WorkerCapacitiesHeartbeat(ctx, envID, instanceID)
		require.NoError(t, err)

		// Check both TTLs are reset
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)

		capacityTTL := r.TTL(capacityKey)
		require.Greater(t, capacityTTL, 70*time.Second) // Should be close to 80s

		counterTTL := r.TTL(counterKey)
		require.Greater(t, counterTTL, 70*time.Second) // Should be close to 80s
	})
}

func TestWorkerCapacityEndToEnd(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-worker"

	t.Run("complete lifecycle", func(t *testing.T) {
		// Worker connects with capacity 3
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 3)
		require.NoError(t, err)

		// Check available capacity
		available, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(3), available)

		// Assign 3 requests
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		// At capacity
		available, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), available)

		// Reject new request
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-4")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)

		// Complete one request
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Now has capacity again
		available, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(1), available)

		// Can assign new request
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-4")
		require.NoError(t, err)

		// Complete all requests
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-4")
		require.NoError(t, err)

		// Back to full capacity
		available, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(3), available)

		// Counter should be deleted
		counterKey := mgr.workerLeasesCounterKey(envID, instanceID)
		require.False(t, r.Exists(counterKey))
	})

	t.Run("worker reconnects with different capacity", func(t *testing.T) {
		instanceID := "test-worker-2"

		// Initial capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Assign some leases
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Worker reconnects with lower capacity
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Capacity updated
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(2), capacity)
	})

	t.Run("worker removes capacity limit", func(t *testing.T) {
		instanceID := "test-worker-3"

		// Set capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 3)
		require.NoError(t, err)

		// Assign lease
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Remove capacity limit
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 0)
		require.NoError(t, err)

		// Should return unlimited
		available, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, consts.ConnectWorkerNoConcurrencyLimitForRequests, available)

		// Can assign without limit
		for i := 0; i < 100; i++ {
			err = mgr.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req")
			require.NoError(t, err)
		}
	})
}

func TestGetLeaseWorkerInstanceID(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("returns empty when no mapping exists", func(t *testing.T) {
		instanceID, err := mgr.GetLeaseWorkerInstanceID(ctx, envID, "non-existent-request")
		require.NoError(t, err)
		require.Equal(t, "", instanceID)
	})

	t.Run("returns worker instance ID after assignment", func(t *testing.T) {
		workerInstance := "test-worker-1"
		requestID := "test-request-1"

		// Set capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, workerInstance, 5)
		require.NoError(t, err)

		// Assign request
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Get worker instance ID
		retrievedInstance, err := mgr.GetLeaseWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, workerInstance, retrievedInstance)
	})

	t.Run("mapping is deleted after request completion", func(t *testing.T) {
		workerInstance := "test-worker-2"
		requestID := "test-request-2"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, workerInstance, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Verify mapping exists
		retrievedInstance, err := mgr.GetLeaseWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, workerInstance, retrievedInstance)

		// Delete lease
		err = mgr.DeleteRequestLeaseFromWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Mapping should be deleted
		retrievedInstance, err = mgr.GetLeaseWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, "", retrievedInstance)
	})

	t.Run("different requests map to different workers", func(t *testing.T) {
		worker1 := "test-worker-3"
		worker2 := "test-worker-4"
		request1 := "test-request-3"
		request2 := "test-request-4"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, worker1, 5)
		require.NoError(t, err)
		err = mgr.SetWorkerTotalCapacity(ctx, envID, worker2, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, worker1, request1)
		require.NoError(t, err)
		err = mgr.AssignRequestLeaseToWorker(ctx, envID, worker2, request2)
		require.NoError(t, err)

		// Check mappings
		retrieved1, err := mgr.GetLeaseWorkerInstanceID(ctx, envID, request1)
		require.NoError(t, err)
		require.Equal(t, worker1, retrieved1)

		retrieved2, err := mgr.GetLeaseWorkerInstanceID(ctx, envID, request2)
		require.NoError(t, err)
		require.Equal(t, worker2, retrieved2)
	})

	t.Run("mapping has TTL set", func(t *testing.T) {
		workerInstance := "test-worker-5"
		requestID := "test-request-5"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, workerInstance, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestLeaseToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Check TTL is set
		leaseWorkerKey := mgr.leaseWorkerKey(envID, requestID)
		ttl := r.TTL(leaseWorkerKey)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, 4*consts.ConnectWorkerRequestLeaseDuration)
	})

	t.Run("no mapping created when no capacity limit", func(t *testing.T) {
		workerInstance := "test-worker-no-limit"
		requestID := "test-request-no-limit"

		// Don't set capacity - worker is unlimited

		err := mgr.AssignRequestLeaseToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// No mapping should exist
		retrievedInstance, err := mgr.GetLeaseWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, "", retrievedInstance)
	})
}
