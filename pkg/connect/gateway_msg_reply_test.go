package connect

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestHandleWorkerReplySavesResponseAndWritesAck(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	requestID := "test-worker-reply"

	err := wsproto.Write(context.Background(), res.ws, &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REPLY,
		Payload: marshalSDKResponse(t, res, requestID),
	})
	require.NoError(t, err)

	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_WORKER_REPLY_ACK, msg.Kind)

	ackData := &connectpb.WorkerReplyAckData{}
	err = proto.Unmarshal(msg.Payload, ackData)
	require.NoError(t, err)
	require.Equal(t, requestID, ackData.RequestId)

	savedResponse, err := res.stateManager.GetResponse(t.Context(), res.envID, requestID)
	require.NoError(t, err)
	require.Equal(t, requestID, savedResponse.RequestId)
}

func TestHandleWorkerReplyInvalidPayloadIsFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerReply(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REPLY,
		Payload: []byte("invalid protobuf"),
	})
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectInternal, serr.SysCode)
}

func TestHandleWorkerReplyAckWriteFailureAfterSaveResponseIsNonFatal(t *testing.T) {
	res := createTestingGateway(t, testingParameters{silent: true})
	handshake(t, res)

	requestID := "test-reply-ack-write-failure"
	responseBytes := marshalSDKResponse(t, res, requestID)

	err := res.ws.CloseNow()
	require.NoError(t, err)

	ch := &connectionHandler{
		svc: res.svc,
		conn: &state.Connection{
			EnvID: res.envID,
			Data: &connectpb.WorkerConnectRequestData{
				InstanceId: "test-worker",
			},
		},
		ws:  res.ws,
		log: res.svc.logger,
	}

	serr := ch.handleWorkerReply(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REPLY,
		Payload: responseBytes,
	})
	require.Nil(t, serr, "reply ack write failure after SaveResponse should not close with connect_internal_error")

	savedResponse, err := res.stateManager.GetResponse(t.Context(), res.envID, requestID)
	require.NoError(t, err)
	require.Equal(t, requestID, savedResponse.RequestId)
}
