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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestHandleWorkerReplyAcksWhenExecutorStalls(t *testing.T) {
	for _, method := range []string{connectpb.ConnectExecutor_Ping_FullMethodName, connectpb.ConnectExecutor_Reply_FullMethodName} {
		t.Run(method, func(t *testing.T) {
			res := createTestingGateway(t, testingParameters{silent: true})
			handshake(t, res)
			started := make(chan struct{})
			executor := startReplyFailingExecutor(t, grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
				if info.FullMethod == method {
					close(started)
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return handler(ctx, req)
			}))
			res.svc.grpcConfig.Executor.Port = executor.port

			requestID := "test-stalled-executor"
			_, err := res.stateManager.LeaseRequest(t.Context(), res.envID, requestID, time.Minute, testExecutorIP)
			require.NoError(t, err)
			err = wsproto.Write(t.Context(), res.ws, &connectpb.ConnectMessage{
				Kind:    connectpb.GatewayMessageType_WORKER_REPLY,
				Payload: marshalSDKResponse(t, res, requestID),
			})
			require.NoError(t, err)
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("executor RPC did not start")
			}

			msg := awaitNextMessage(t, res.ws, 2*time.Second)
			require.Equal(t, connectpb.GatewayMessageType_WORKER_REPLY_ACK, msg.Kind)
			ack := &connectpb.WorkerReplyAckData{}
			require.NoError(t, proto.Unmarshal(msg.Payload, ack))
			require.Equal(t, requestID, ack.RequestId)
			response, err := res.stateManager.GetResponse(t.Context(), res.envID, requestID)
			require.NoError(t, err)
			require.NotNil(t, response)
			require.Equal(t, "test response", string(response.Body))
			exchangeHeartbeat(t, res.ws, 2*time.Second)
		})
	}
}

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
