package connect

import (
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestHandleWorkerRequestExtendLeaseExtendsLease(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	requestID := "test-extend-lease"
	leaseID, err := res.svc.stateManager.LeaseRequest(t.Context(), res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)

	sendWorkerExtendLeaseMessage(t, res, &connectpb.WorkerRequestExtendLeaseData{
		RequestId:      requestID,
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		FunctionSlug:   res.fnSlug,
		StepId:         ptr.String("step"),
		RunId:          res.runID.String(),
		LeaseId:        leaseID.String(),
		SystemTraceCtx: []byte("system"),
		UserTraceCtx:   []byte("user"),
	})

	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK, msg.Kind)

	ackPayload := connectpb.WorkerRequestExtendLeaseAckData{}
	err = proto.Unmarshal(msg.Payload, &ackPayload)
	require.NoError(t, err)
	require.Equal(t, requestID, ackPayload.RequestId)
	require.NotNil(t, ackPayload.NewLeaseId)

	parsed, err := ulid.Parse(*ackPayload.NewLeaseId)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(consts.ConnectWorkerRequestLeaseDuration), ulid.Time(parsed.Time()), 2*time.Second)
}

func TestHandleWorkerRequestExtendLeaseDeletedLeaseNacks(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	requestID := "test-extend-deleted-lease"
	leaseID, err := res.svc.stateManager.LeaseRequest(t.Context(), res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)
	require.NoError(t, res.svc.stateManager.DeleteLease(t.Context(), res.envID, requestID))

	sendWorkerExtendLeaseMessage(t, res, &connectpb.WorkerRequestExtendLeaseData{
		RequestId:    requestID,
		AccountId:    res.accountID.String(),
		EnvId:        res.envID.String(),
		AppId:        res.appID.String(),
		FunctionSlug: res.fnSlug,
		RunId:        res.runID.String(),
		LeaseId:      leaseID.String(),
	})

	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK, msg.Kind)

	nackPayload := connectpb.WorkerRequestExtendLeaseAckData{}
	err = proto.Unmarshal(msg.Payload, &nackPayload)
	require.NoError(t, err)
	require.Equal(t, requestID, nackPayload.RequestId)
	require.Nil(t, nackPayload.NewLeaseId)
}

func TestHandleWorkerRequestExtendLeaseInvalidPayloadIsFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerRequestExtendLease(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
		Payload: []byte("invalid protobuf"),
	})
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload, serr.SysCode)
}

func TestHandleWorkerRequestExtendLeaseInvalidLeaseIDIsFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	payload, err := proto.Marshal(&connectpb.WorkerRequestExtendLeaseData{
		RequestId: "req-1",
		LeaseId:   "invalid-ulid",
	})
	require.NoError(t, err)

	serr := ch.handleWorkerRequestExtendLease(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
		Payload: payload,
	})
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload, serr.SysCode)
}
