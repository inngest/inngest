package state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestLeaseRequest(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()

	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	var requestStateManager RequestStateManager = connManager

	envID := uuid.New()
	instanceID := "instance-1"
	isWorkerCapacityUnlimited := true
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	executorIP := net.IPv4(1, 1, 1, 1)

	var existingLeaseID ulid.ULID

	t.Run("deleting a missing lease should be a no-op", func(t *testing.T) {
		err = requestStateManager.DeleteLease(ctx, envID, requestID)
		require.NoError(t, err)
	})

	t.Run("should not report missing lease as leased", func(t *testing.T) {
		leased, err := requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.False(t, leased)
	})

	t.Run("extending missing lease should not work", func(t *testing.T) {
		otherLeaseID := ulid.MustNew(ulid.Now(), rand.Reader)
		leaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, instanceID, requestID, otherLeaseID, consts.ConnectWorkerRequestLeaseDuration, isWorkerCapacityUnlimited)
		require.Nil(t, leaseID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequestLeaseNotFound)
	})

	t.Run("leasing request should work", func(t *testing.T) {
		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, executorIP)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		existingLeaseID = *leaseID

		ip, err := requestStateManager.GetExecutorIP(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, executorIP, ip)
	})

	t.Run("should report active lease as leased", func(t *testing.T) {
		leased, err := requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.True(t, leased)
	})

	t.Run("leasing again should not work", func(t *testing.T) {
		ip, err := requestStateManager.GetExecutorIP(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, executorIP, ip)

		// Simulate a new executor
		newIP := net.IPv4(1, 2, 3, 4)

		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, newIP)
		require.Nil(t, leaseID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequestLeased)

		// Expect the IP to have been updated. This is useful to allow gRPC responses in case the
		// original executor terminated while processing the request.
		ip, err = requestStateManager.GetExecutorIP(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, newIP, ip)
	})

	t.Run("extending somebody else's lease should not work", func(t *testing.T) {
		otherLeaseID := ulid.MustNew(ulid.Now(), rand.Reader)
		leaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, instanceID, requestID, otherLeaseID, consts.ConnectWorkerRequestLeaseDuration, isWorkerCapacityUnlimited)
		require.Nil(t, leaseID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequestLeased)
	})

	t.Run("extending own lease should work", func(t *testing.T) {
		leaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, instanceID, requestID, existingLeaseID, consts.ConnectWorkerRequestLeaseDuration, isWorkerCapacityUnlimited)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotEqual(t, existingLeaseID, leaseID)

		existingLeaseID = *leaseID
	})

	t.Run("should not report expired lease as leased", func(t *testing.T) {
		advancePastExpiry := consts.ConnectWorkerRequestLeaseDuration + 1*time.Second
		r.FastForward(advancePastExpiry)
		fakeClock.Advance(advancePastExpiry)

		leased, err := requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.False(t, leased)
	})

	t.Run("leasing expired item should work", func(t *testing.T) {
		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, executorIP)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		existingLeaseID = *leaseID
	})

	t.Run("dropping lease should work", func(t *testing.T) {
		leased, err := requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.True(t, leased)

		newLeaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, instanceID, requestID, existingLeaseID, 0, isWorkerCapacityUnlimited)
		require.NoError(t, err)
		require.Nil(t, newLeaseID)

		leased, err = requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.False(t, leased)

		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, executorIP)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		existingLeaseID = *leaseID

		leased, err = requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.True(t, leased)

		err = requestStateManager.DeleteLease(ctx, envID, requestID)
		require.NoError(t, err)

		leased, err = requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.False(t, leased)
	})
}

func TestExtendRequestLeaseIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	instanceID := "instance-1"
	otherInstanceID := "instance-2"
	duration := consts.ConnectWorkerRequestLeaseDuration
	executorIP := net.IPv4(1, 1, 1, 1)
	leaseKey := connManager.keyRequestLease(envID, requestID)

	initialLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, executorIP)
	require.NoError(t, err)

	keysBeforeExtension := r.Keys()
	extendedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, extendedLeaseID)
	require.NotEqual(t, *initialLeaseID, *extendedLeaseID)

	serializedLeaseAfterExtension, err := r.Get(leaseKey)
	require.NoError(t, err)

	var storedLease Lease
	require.NoError(t, json.Unmarshal([]byte(serializedLeaseAfterExtension), &storedLease))
	require.Equal(t, *extendedLeaseID, storedLease.LeaseID)
	require.NotNil(t, storedLease.PreviousLeaseID)
	require.Equal(t, *initialLeaseID, *storedLease.PreviousLeaseID)
	require.Equal(t, instanceID, storedLease.WorkerInstanceID)

	// Idempotency metadata stays inside the existing lease value. It must not
	// add a replay/history key for each connection or renewal.
	require.ElementsMatch(t, keysBeforeExtension, r.Keys())

	ttlAfterExtension := r.TTL(leaseKey)
	require.Equal(t, consts.MaxFunctionTimeout+duration, ttlAfterExtension)
	advance := 10 * time.Second
	r.FastForward(advance)
	fakeClock.Advance(advance)

	replayedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.Equal(t, *extendedLeaseID, *replayedLeaseID)
	serializedLeaseAfterReplay, err := r.Get(leaseKey)
	require.NoError(t, err)
	require.Equal(t, serializedLeaseAfterExtension, serializedLeaseAfterReplay)
	require.Equal(t, ttlAfterExtension-advance, r.TTL(leaseKey))
	require.ElementsMatch(t, keysBeforeExtension, r.Keys())

	// Knowing a stale lease ID is insufficient without the worker instance
	// that performed the original extension.
	replayedLeaseID, err = connManager.ExtendRequestLease(
		ctx,
		envID,
		otherInstanceID,
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.ErrorIs(t, err, ErrRequestLeased)
	require.Nil(t, replayedLeaseID)

	// Keep replay history bounded to one predecessor. Rotating again replaces
	// the predecessor rather than growing the Redis value over time.
	nextLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*extendedLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, nextLeaseID)

	staleReplayLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.ErrorIs(t, err, ErrRequestLeased)
	require.Nil(t, staleReplayLeaseID)

	replayedNextLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*extendedLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.Equal(t, *nextLeaseID, *replayedNextLeaseID)
	require.ElementsMatch(t, keysBeforeExtension, r.Keys())

	// Repeated connection rotations keep only one predecessor and do not
	// increase key cardinality or grow the serialized lease value.
	currentLeaseID := *nextLeaseID
	for range 100 {
		rotatedLeaseID, err := connManager.ExtendRequestLease(
			ctx,
			envID,
			instanceID,
			requestID,
			currentLeaseID,
			duration,
			true,
		)
		require.NoError(t, err)
		require.NotNil(t, rotatedLeaseID)
		currentLeaseID = *rotatedLeaseID
	}

	serializedLeaseAfterRotations, err := r.Get(leaseKey)
	require.NoError(t, err)
	require.LessOrEqual(t, len(serializedLeaseAfterRotations), len(serializedLeaseAfterExtension))
	require.ElementsMatch(t, keysBeforeExtension, r.Keys())
}

func TestExtendRequestLeaseLegacyValueCompatibility(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	instanceID := "instance-1"
	duration := consts.ConnectWorkerRequestLeaseDuration
	leaseKey := connManager.keyRequestLease(envID, requestID)
	legacyLeaseID := ulid.MustNew(ulid.Timestamp(fakeClock.Now().Add(duration)), rand.Reader)

	// Model a lease written by a gateway from before replay metadata was added.
	legacyLease := struct {
		LeaseID    ulid.ULID `json:"leaseID"`
		ExecutorIP net.IP    `json:"executorIP"`
	}{
		LeaseID:    legacyLeaseID,
		ExecutorIP: net.IPv4(1, 1, 1, 1),
	}
	serializedLegacyLease, err := json.Marshal(legacyLease)
	require.NoError(t, err)
	require.NotContains(t, string(serializedLegacyLease), `"prevLeaseID"`)
	require.NotContains(t, string(serializedLegacyLease), `"workerID"`)
	require.NoError(t, r.Set(leaseKey, string(serializedLegacyLease)))
	r.SetTTL(leaseKey, consts.MaxFunctionTimeout+duration)

	extendedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		legacyLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, extendedLeaseID)

	serializedExtendedLease, err := r.Get(leaseKey)
	require.NoError(t, err)
	var storedLease Lease
	require.NoError(t, json.Unmarshal([]byte(serializedExtendedLease), &storedLease))
	require.Equal(t, *extendedLeaseID, storedLease.LeaseID)
	require.Equal(t, legacyLeaseID, *storedLease.PreviousLeaseID)
	require.Equal(t, instanceID, storedLease.WorkerInstanceID)
	require.Equal(t, legacyLease.ExecutorIP, storedLease.ExecutorIP)

	unrelatedLeaseID := ulid.MustNew(ulid.Timestamp(fakeClock.Now().Add(duration)), rand.Reader)
	rejectedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		unrelatedLeaseID,
		duration,
		true,
	)
	require.ErrorIs(t, err, ErrRequestLeased)
	require.Nil(t, rejectedLeaseID)
}

func TestLeaseRequestPreservesExtensionReplayMetadata(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	instanceID := "instance-1"
	duration := consts.ConnectWorkerRequestLeaseDuration
	leaseKey := connManager.keyRequestLease(envID, requestID)

	initialLeaseID, err := connManager.LeaseRequest(
		ctx,
		envID,
		requestID,
		duration,
		net.IPv4(1, 1, 1, 1),
	)
	require.NoError(t, err)
	extendedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)

	ttlBeforeLeaseAttempt := r.TTL(leaseKey)

	newExecutorIP := net.IPv4(2, 2, 2, 2)
	rejectedLeaseID, err := connManager.LeaseRequest(
		ctx,
		envID,
		requestID,
		duration,
		newExecutorIP,
	)
	require.ErrorIs(t, err, ErrRequestLeased)
	require.Nil(t, rejectedLeaseID)
	require.Equal(t, ttlBeforeLeaseAttempt, r.TTL(leaseKey))

	serializedLease, err := r.Get(leaseKey)
	require.NoError(t, err)

	var storedLease Lease
	require.NoError(t, json.Unmarshal([]byte(serializedLease), &storedLease))
	require.Equal(t, *extendedLeaseID, storedLease.LeaseID)
	require.Equal(t, *initialLeaseID, *storedLease.PreviousLeaseID)
	require.Equal(t, instanceID, storedLease.WorkerInstanceID)
	require.Equal(t, newExecutorIP, storedLease.ExecutorIP)

	// The predecessor remains usable after the rejected lease attempt.
	replayedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.Equal(t, *extendedLeaseID, *replayedLeaseID)
}

func TestExtendRequestLeaseReplayOwnershipClearedOnReassignment(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	duration := consts.ConnectWorkerRequestLeaseDuration
	executorIP := net.IPv4(1, 1, 1, 1)
	leaseKey := connManager.keyRequestLease(envID, requestID)

	initialLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, executorIP)
	require.NoError(t, err)
	ownedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		"old-instance",
		requestID,
		*initialLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)

	advance := duration + time.Second
	r.FastForward(advance)
	fakeClock.Advance(advance)

	reassignedLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, executorIP)
	require.NoError(t, err)

	serializedReassignedLease, err := r.Get(leaseKey)
	require.NoError(t, err)
	var reassignedLease Lease
	require.NoError(t, json.Unmarshal([]byte(serializedReassignedLease), &reassignedLease))
	require.Equal(t, *reassignedLeaseID, reassignedLease.LeaseID)
	require.Nil(t, reassignedLease.PreviousLeaseID)
	require.Empty(t, reassignedLease.WorkerInstanceID)

	// The old owner cannot use its former current lease to adopt a lease that
	// was legitimately reassigned after expiry.
	replayedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		"old-instance",
		requestID,
		*ownedLeaseID,
		duration,
		true,
	)
	require.ErrorIs(t, err, ErrRequestLeased)
	require.Nil(t, replayedLeaseID)

	newOwnerLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		"new-instance",
		requestID,
		*reassignedLeaseID,
		duration,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, newOwnerLeaseID)
}

func TestExtendRequestLeaseIdempotentReplayWithCapacityLimit(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	instanceID := "capacity-instance"
	duration := consts.ConnectWorkerRequestLeaseDuration

	require.NoError(t, connManager.SetWorkerTotalCapacity(ctx, envID, instanceID, 50))
	initialLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, net.IPv4(1, 1, 1, 1))
	require.NoError(t, err)
	require.NoError(t, connManager.AssignRequestToWorker(ctx, envID, instanceID, requestID))

	extendedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		false,
	)
	require.NoError(t, err)

	keysAfterExtension := r.Keys()
	serializedLeaseAfterExtension, err := r.Get(connManager.keyRequestLease(envID, requestID))
	require.NoError(t, err)
	var storedLease Lease
	require.NoError(t, json.Unmarshal([]byte(serializedLeaseAfterExtension), &storedLease))
	require.NotNil(t, storedLease.PreviousLeaseID)
	require.Empty(t, storedLease.WorkerInstanceID)

	mappingKey := connManager.requestWorkerKey(envID, requestID)
	mappingTTLAfterExtension := r.TTL(mappingKey)
	require.Equal(t, consts.ConnectWorkerRequestToWorkerMappingTTL, mappingTTLAfterExtension)

	advance := 10 * time.Second
	r.FastForward(advance)
	fakeClock.Advance(advance)

	replayedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, *extendedLeaseID, *replayedLeaseID)
	require.ElementsMatch(t, keysAfterExtension, r.Keys())
	require.Equal(t, mappingTTLAfterExtension-advance, r.TTL(mappingKey))

	workerRequests, err := r.ZMembers(connManager.workerRequestsKey(envID, instanceID))
	require.NoError(t, err)
	require.Contains(t, workerRequests, requestID)

	wrongWorkerLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		"other-instance",
		requestID,
		*initialLeaseID,
		duration,
		false,
	)
	require.ErrorIs(t, err, ErrRequestWorkerDoesNotExist)
	require.Nil(t, wrongWorkerLeaseID)
}

func TestExtendRequestLeaseReplayAfterCapacityBecomesUnlimited(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	instanceID := "capacity-instance"
	duration := consts.ConnectWorkerRequestLeaseDuration

	require.NoError(t, connManager.SetWorkerTotalCapacity(ctx, envID, instanceID, 50))
	initialLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, net.IPv4(1, 1, 1, 1))
	require.NoError(t, err)
	require.NoError(t, connManager.AssignRequestToWorker(ctx, envID, instanceID, requestID))

	extendedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, extendedLeaseID)

	serializedLease, err := r.Get(connManager.keyRequestLease(envID, requestID))
	require.NoError(t, err)
	var storedLease Lease
	require.NoError(t, json.Unmarshal([]byte(serializedLease), &storedLease))
	require.Empty(t, storedLease.WorkerInstanceID)

	mappingKey := connManager.requestWorkerKey(envID, requestID)
	mappingTTLBeforeReplay := r.TTL(mappingKey)
	require.Equal(t, consts.ConnectWorkerRequestToWorkerMappingTTL, mappingTTLBeforeReplay)

	// Losing the capacity key makes the gateway treat this worker as unlimited,
	// but the existing request-to-worker mapping still proves replay ownership.
	require.NoError(t, connManager.SetWorkerTotalCapacity(ctx, envID, instanceID, 0))
	workerCapacity, err := connManager.GetWorkerCapacities(ctx, envID, instanceID)
	require.NoError(t, err)
	require.True(t, workerCapacity.IsUnlimited())

	replayedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		requestID,
		*initialLeaseID,
		duration,
		workerCapacity.IsUnlimited(),
	)
	require.NoError(t, err)
	require.Equal(t, *extendedLeaseID, *replayedLeaseID)
	require.Equal(t, mappingTTLBeforeReplay, r.TTL(mappingKey))

	wrongWorkerLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		"other-instance",
		requestID,
		*initialLeaseID,
		duration,
		workerCapacity.IsUnlimited(),
	)
	require.ErrorIs(t, err, ErrRequestLeased)
	require.Nil(t, wrongWorkerLeaseID)
}

func TestExtendRequestLeaseCapacityLimitedReassignment(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()
	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	oldInstanceID := "capacity-instance-old"
	newInstanceID := "capacity-instance-new"
	duration := consts.ConnectWorkerRequestLeaseDuration
	executorIP := net.IPv4(1, 1, 1, 1)

	require.NoError(t, connManager.SetWorkerTotalCapacity(ctx, envID, oldInstanceID, 50))
	require.NoError(t, connManager.SetWorkerTotalCapacity(ctx, envID, newInstanceID, 50))
	initialLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, executorIP)
	require.NoError(t, err)
	require.NoError(t, connManager.AssignRequestToWorker(ctx, envID, oldInstanceID, requestID))

	oldWorkerLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		oldInstanceID,
		requestID,
		*initialLeaseID,
		duration,
		false,
	)
	require.NoError(t, err)

	advance := duration + time.Second
	r.FastForward(advance)
	fakeClock.Advance(advance)

	reassignedLeaseID, err := connManager.LeaseRequest(ctx, envID, requestID, duration, executorIP)
	require.NoError(t, err)
	require.NoError(t, connManager.DeleteRequestFromWorker(ctx, envID, oldInstanceID, requestID))
	require.NoError(t, connManager.AssignRequestToWorker(ctx, envID, newInstanceID, requestID))

	// The old worker still knows a formerly valid lease ID, but ownership has
	// moved and its replay must not be adopted by the new worker.
	replayedLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		oldInstanceID,
		requestID,
		*oldWorkerLeaseID,
		duration,
		false,
	)
	require.ErrorIs(t, err, ErrRequestWorkerDoesNotExist)
	require.Nil(t, replayedLeaseID)

	newWorkerLeaseID, err := connManager.ExtendRequestLease(
		ctx,
		envID,
		newInstanceID,
		requestID,
		*reassignedLeaseID,
		duration,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, newWorkerLeaseID)

	mappingKey := connManager.requestWorkerKey(envID, requestID)
	mappedInstanceID, err := r.Get(mappingKey)
	require.NoError(t, err)
	require.Equal(t, newInstanceID, mappedInstanceID)
	require.Equal(t, consts.ConnectWorkerRequestToWorkerMappingTTL, r.TTL(mappingKey))

	require.False(t, r.Exists(connManager.workerRequestsKey(envID, oldInstanceID)))
	newWorkerRequests, err := r.ZMembers(connManager.workerRequestsKey(envID, newInstanceID))
	require.NoError(t, err)
	require.Contains(t, newWorkerRequests, requestID)
}

func TestBufferResponse(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	fakeClock := clockwork.NewFakeClock()

	connManager := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{
		Clock: fakeClock,
	})

	var requestStateManager RequestStateManager = connManager

	envID := uuid.New()
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	expectedResp := &connpb.SDKResponse{
		RequestId:      requestID,
		AccountId:      "test-account",
		EnvId:          envID.String(),
		Status:         connpb.SDKResponseStatus_DONE,
		Body:           []byte("hello world"),
		SdkVersion:     "v1.2.3",
		RequestVersion: 1,
		RunId:          "run-id-test",
	}

	resp, err := requestStateManager.GetResponse(ctx, envID, requestID)
	require.NoError(t, err)
	require.Nil(t, resp)

	err = requestStateManager.SaveResponse(ctx, envID, requestID, expectedResp)
	require.NoError(t, err)

	resp, err = requestStateManager.GetResponse(ctx, envID, requestID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, proto.Equal(expectedResp, resp))

	err = requestStateManager.DeleteResponse(ctx, envID, requestID)
	require.NoError(t, err)

	resp, err = requestStateManager.GetResponse(ctx, envID, requestID)
	require.NoError(t, err)
	require.Nil(t, resp)
}
