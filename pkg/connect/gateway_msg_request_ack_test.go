package connect

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestHandleWorkerRequestAckInvalidPayloadIsFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerRequestAck(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_ACK,
		Payload: []byte("invalid protobuf"),
	})
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectWorkerRequestAckInvalidPayload, serr.SysCode)
}

func TestHandleWorkerRequestAckMissingExecutorLeaseIsNonFatalAfterPendingAck(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	requestID := "test-missing-executor-lease"
	ackCh := make(chan struct{})
	ch.pendingAcks.Store(requestID, ackCh)

	serr := ch.handleWorkerRequestAck(workerRequestAckMessage(t, res, requestID))
	require.Nil(t, serr)

	select {
	case <-ackCh:
	case <-time.After(time.Second):
		t.Fatal("pending ack channel should be closed before executor notification")
	}
}

func workerRequestAckMessage(t *testing.T, res testingResources, requestID string) *connectpb.ConnectMessage {
	t.Helper()

	payload, err := proto.Marshal(&connectpb.WorkerRequestAckData{
		RequestId: requestID,
		AccountId: res.accountID.String(),
		EnvId:     res.envID.String(),
		AppId:     res.appID.String(),
		RunId:     res.runID.String(),
	})
	require.NoError(t, err)

	return &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_ACK,
		Payload: payload,
	}
}
