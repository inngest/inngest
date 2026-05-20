package connect

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

func TestHandleWorkerRequestAckNotifiesExecutor(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	executor := startAckExecutor(t, &ackExecutor{
		ackReceived: make(chan *connectpb.AckMessage, 1),
	})
	res.svc.grpcConfig.Executor.Port = executor.port

	requestID := "test-worker-request-ack-success"
	_, err := res.svc.stateManager.LeaseRequest(t.Context(), res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerRequestAck(workerRequestAckMessage(t, res, requestID))
	require.Nil(t, serr)

	select {
	case ack := <-executor.ackReceived:
		require.Equal(t, requestID, ack.RequestId)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for executor ack")
	}
}

func TestHandleWorkerRequestAckRPCFailureIsNonFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	executor := startAckExecutor(t, &ackExecutor{
		ackReceived: make(chan *connectpb.AckMessage, 1),
		err:         status.Error(codes.Unavailable, "ack unavailable"),
	})
	res.svc.grpcConfig.Executor.Port = executor.port

	requestID := "test-worker-request-ack-rpc-failure"
	_, err := res.svc.stateManager.LeaseRequest(t.Context(), res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerRequestAck(workerRequestAckMessage(t, res, requestID))
	require.Nil(t, serr)
}

func TestHandleWorkerRequestAckExecutorDoneIsNonFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	executor := startAckExecutor(t, &ackExecutor{
		ackReceived: make(chan *connectpb.AckMessage, 1),
		success:     boolPtr(false),
	})
	res.svc.grpcConfig.Executor.Port = executor.port

	requestID := "test-worker-request-ack-executor-done"
	_, err := res.svc.stateManager.LeaseRequest(t.Context(), res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerRequestAck(workerRequestAckMessage(t, res, requestID))
	require.Nil(t, serr)
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

type ackExecutor struct {
	connectpb.UnimplementedConnectExecutorServer
	port        int
	ackReceived chan *connectpb.AckMessage
	success     *bool
	err         error
}

func startAckExecutor(t *testing.T, executor *ackExecutor) *ackExecutor {
	t.Helper()

	if executor.ackReceived == nil {
		executor.ackReceived = make(chan *connectpb.AckMessage, 1)
	}
	grpcServer := grpc.NewServer()
	connectpb.RegisterConnectExecutorServer(grpcServer, executor)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	executor.port = lis.Addr().(*net.TCPAddr).Port

	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	testConn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%d", executor.port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	require.NoError(t, testConn.Close())

	return executor
}

func (s *ackExecutor) Ack(_ context.Context, msg *connectpb.AckMessage) (*connectpb.AckResponse, error) {
	s.ackReceived <- msg
	if s.err != nil {
		return nil, s.err
	}
	success := true
	if s.success != nil {
		success = *s.success
	}
	return &connectpb.AckResponse{Success: success}, nil
}

func (s *ackExecutor) Ping(_ context.Context, _ *connectpb.PingRequest) (*connectpb.PingResponse, error) {
	return &connectpb.PingResponse{Message: "ok"}, nil
}

func boolPtr(v bool) *bool {
	return &v
}
