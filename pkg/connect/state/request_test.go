package state

import (
	"context"
	"crypto/rand"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
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
	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()

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
		leaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, requestID, otherLeaseID, consts.ConnectWorkerRequestLeaseDuration)
		require.Nil(t, leaseID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequestLeaseNotFound)
	})

	t.Run("leasing request should work", func(t *testing.T) {
		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		existingLeaseID = *leaseID
	})

	t.Run("should report active lease as leased", func(t *testing.T) {
		leased, err := requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.True(t, leased)
	})

	t.Run("leasing again should not work", func(t *testing.T) {
		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration)
		require.Nil(t, leaseID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequestLeased)
	})

	t.Run("extending somebody else's lease should not work", func(t *testing.T) {
		otherLeaseID := ulid.MustNew(ulid.Now(), rand.Reader)
		leaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, requestID, otherLeaseID, consts.ConnectWorkerRequestLeaseDuration)
		require.Nil(t, leaseID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequestLeased)
	})

	t.Run("extending own lease should work", func(t *testing.T) {
		leaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, requestID, existingLeaseID, consts.ConnectWorkerRequestLeaseDuration)
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
		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		existingLeaseID = *leaseID
	})

	t.Run("dropping lease should work", func(t *testing.T) {
		leased, err := requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.True(t, leased)

		newLeaseID, err := requestStateManager.ExtendRequestLease(ctx, envID, requestID, existingLeaseID, 0)
		require.NoError(t, err)
		require.Nil(t, newLeaseID)

		leased, err = requestStateManager.IsRequestLeased(ctx, envID, requestID)
		require.NoError(t, err)
		require.False(t, leased)

		leaseID, err := requestStateManager.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration)
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
