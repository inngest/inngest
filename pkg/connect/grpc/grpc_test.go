package grpc

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupRedisGatewayManager(t *testing.T) (state.GatewayManager, func()) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	gatewayManager := state.NewRedisConnectionStateManager(rc)

	cleanup := func() {
		rc.Close()
		r.Close()
	}

	return gatewayManager, cleanup
}

func setupTestEnvironment(t *testing.T) (context.Context, *mockConnectGatewayServer, func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error), state.GatewayManager, func()) {
	ctx := context.Background()

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	mockServer := &mockConnectGatewayServer{}
	connectpb.RegisterConnectGatewayServer(server, mockServer)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	bufDialer := func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		dialerOpt := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		})
		return grpc.NewClient("passthrough:///bufnet", append(opts, dialerOpt)...)
	}

	gatewayManager, gmCleanup := setupRedisGatewayManager(t)

	cleanup := func() {
		server.Stop()
		lis.Close()
		gmCleanup()
	}

	return ctx, mockServer, bufDialer, gatewayManager, cleanup
}

func newGatewayGRPCManagerWithDialer(ctx context.Context, stateManager state.GatewayManager, dialer GRPCDialer) GatewayGRPCManager {
	return newGatewayGRPCManager(ctx, stateManager, WithGatewayDialer(dialer))
}

type mockConnectGatewayServer struct {
	connectpb.UnimplementedConnectGatewayServer
	pingCount    int
	forwardCount int
}

func (m *mockConnectGatewayServer) Ping(ctx context.Context, req *connectpb.PingRequest) (*connectpb.PingResponse, error) {
	m.pingCount++
	return &connectpb.PingResponse{Message: "ok"}, nil
}

func (m *mockConnectGatewayServer) Forward(ctx context.Context, req *connectpb.ForwardRequest) (*connectpb.ForwardResponse, error) {
	m.forwardCount++
	return &connectpb.ForwardResponse{}, nil
}

func (m *mockConnectGatewayServer) Ack(ctx context.Context, req *connectpb.AckMessage) (*connectpb.AckResponse, error) {
	return &connectpb.AckResponse{Success: true}, nil
}

func (m *mockConnectGatewayServer) getPingCount() int {
	return m.pingCount
}

func (m *mockConnectGatewayServer) getForwardCount() int {
	return m.forwardCount
}

func (m *mockConnectGatewayServer) reset() {
	m.pingCount = 0
	m.forwardCount = 0
}

func TestConnectToGateways(t *testing.T) {
	ctx, mockServer, bufDialer, gatewayManager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	gatewayID := ulid.MustNew(ulid.Now(), rand.Reader)
	gateway := &state.Gateway{
		Id:                gatewayID,
		Status:            state.GatewayStatusActive,
		IPAddress:         net.ParseIP("127.0.0.1"),
		Hostname:          "test-gateway",
		LastHeartbeatAtMS: time.Now().UnixMilli(),
	}
	err := gatewayManager.UpsertGateway(ctx, gateway)
	require.NoError(t, err)

	forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)

	t.Run("connects to single gateway", func(t *testing.T) {
		mockServer.reset()

		err := forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		require.Equal(t, 1, mockServer.getPingCount(), "Ping should be called once during ConnectToGateways")
		require.Equal(t, 0, mockServer.getForwardCount(), "Forward should not be called during ConnectToGateways")
	})

	t.Run("connects to multiple gateways", func(t *testing.T) {
		mockServer.reset()

		forwarder = newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)

		gatewayID2 := ulid.MustNew(ulid.Now(), nil)
		gateway2 := &state.Gateway{
			Id:                gatewayID2,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.2"),
			Hostname:          "test-gateway-2",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}
		err := gatewayManager.UpsertGateway(ctx, gateway2)
		require.NoError(t, err)

		err = gatewayManager.UpsertGateway(ctx, gateway)
		require.NoError(t, err)

		err = forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		require.Equal(t, 2, mockServer.getPingCount(), "Ping should be called twice for two gateways")
	})
}

func TestForward(t *testing.T) {
	ctx, mockServer, bufDialer, gatewayManager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	gatewayID := ulid.MustNew(ulid.Now(), rand.Reader)
	gateway := &state.Gateway{
		Id:                gatewayID,
		Status:            state.GatewayStatusActive,
		IPAddress:         net.ParseIP("127.0.0.1"),
		Hostname:          "test-gateway",
		LastHeartbeatAtMS: time.Now().UnixMilli(),
	}
	err := gatewayManager.UpsertGateway(ctx, gateway)
	require.NoError(t, err)

	forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)

	t.Run("forwards to existing gateway", func(t *testing.T) {
		mockServer.reset()

		err := forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		initialPingCount := mockServer.getPingCount()

		connectionID := ulid.MustNew(ulid.Now(), nil)
		data := &connectpb.GatewayExecutorRequestData{
			RequestId: uuid.New().String(),
		}

		err = forwarder.Forward(ctx, gatewayID, connectionID, data)
		require.NoError(t, err)

		require.Equal(t, initialPingCount, mockServer.getPingCount(), "No additional pings should be made during Forward")
		require.Equal(t, 1, mockServer.getForwardCount(), "Forward should be called once")
	})

	t.Run("fails for non-existent gateway", func(t *testing.T) {
		mockServer.reset()

		nonExistentGatewayID := ulid.MustNew(ulid.Now(), nil)
		connectionID := ulid.MustNew(ulid.Now(), nil)
		data := &connectpb.GatewayExecutorRequestData{
			RequestId: uuid.New().String(),
		}

		err := forwarder.Forward(ctx, nonExistentGatewayID, connectionID, data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not find gateway")

		require.Equal(t, 0, mockServer.getPingCount(), "Ping should not be called for non-existent gateway")
		require.Equal(t, 0, mockServer.getForwardCount(), "Forward should not be called for non-existent gateway")
	})

	t.Run("creates just-in-time connection", func(t *testing.T) {
		mockServer.reset()

		newGatewayID := ulid.MustNew(ulid.Now(), nil)
		newGateway := &state.Gateway{
			Id:                newGatewayID,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.3"),
			Hostname:          "test-gateway-jit",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}
		err := gatewayManager.UpsertGateway(ctx, newGateway)
		require.NoError(t, err)

		connectionID := ulid.MustNew(ulid.Now(), nil)
		data := &connectpb.GatewayExecutorRequestData{
			RequestId: uuid.New().String(),
		}

		err = forwarder.Forward(ctx, newGatewayID, connectionID, data)
		require.NoError(t, err)

		require.Equal(t, 1, mockServer.getPingCount(), "Ping should be called once during just-in-time connection")
		require.Equal(t, 1, mockServer.getForwardCount(), "Forward should be called once")
	})

	t.Run("forwards to multiple gateways", func(t *testing.T) {
		mockServer.reset()

		newGatewayManager, newCleanup := setupRedisGatewayManager(t)
		defer newCleanup()
		newForwarder := newGatewayGRPCManagerWithDialer(ctx, newGatewayManager, bufDialer)

		gatewayID1 := ulid.MustNew(ulid.Now(), rand.Reader)
		gateway1 := &state.Gateway{
			Id:                gatewayID1,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.1"),
			Hostname:          "test-gateway-1",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}

		gatewayID2 := ulid.MustNew(ulid.Now(), nil)
		gateway2 := &state.Gateway{
			Id:                gatewayID2,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.2"),
			Hostname:          "test-gateway-2",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}

		err := newGatewayManager.UpsertGateway(ctx, gateway1)
		require.NoError(t, err)
		err = newGatewayManager.UpsertGateway(ctx, gateway2)
		require.NoError(t, err)

		err = newForwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		require.Equal(t, 2, mockServer.getPingCount(), "Ping should be called twice for two gateways")

		connectionID := ulid.MustNew(ulid.Now(), nil)
		data := &connectpb.GatewayExecutorRequestData{
			RequestId: uuid.New().String(),
		}

		err = newForwarder.Forward(ctx, gatewayID1, connectionID, data)
		require.NoError(t, err)

		err = newForwarder.Forward(ctx, gatewayID2, connectionID, data)
		require.NoError(t, err)

		require.Equal(t, 2, mockServer.getPingCount(), "No additional pings should be made during Forward")
		require.Equal(t, 2, mockServer.getForwardCount(), "Forward should be called twice")
	})
}

func TestGarbageCollectClients(t *testing.T) {
	ctx, mockServer, bufDialer, gatewayManager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("collects single client", func(t *testing.T) {
		gatewayID := ulid.MustNew(ulid.Now(), rand.Reader)
		gateway := &state.Gateway{
			Id:                gatewayID,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.1"),
			Hostname:          "test-gateway",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}
		err := gatewayManager.UpsertGateway(ctx, gateway)
		require.NoError(t, err)

		forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)

		err = forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		err = gatewayManager.DeleteGateway(ctx, gatewayID)
		require.NoError(t, err)

		forwarderImpl := forwarder.(*gatewayGRPCManager)
		deletedCount, err := forwarderImpl.GarbageCollectClients()
		require.NoError(t, err)
		require.Equal(t, 1, deletedCount, "Should have deleted exactly 1 client")

		deletedCount, err = forwarderImpl.GarbageCollectClients()
		require.NoError(t, err)
		require.Equal(t, 0, deletedCount, "Second garbage collection should delete nothing")

		connectionID := ulid.MustNew(ulid.Now(), nil)
		data := &connectpb.GatewayExecutorRequestData{
			RequestId: uuid.New().String(),
		}

		err = forwarder.Forward(ctx, gatewayID, connectionID, data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not find gateway")
	})

	t.Run("collects multiple clients", func(t *testing.T) {
		mockServer.reset()

		newGatewayManager, newCleanup := setupRedisGatewayManager(t)
		defer newCleanup()
		newForwarder := newGatewayGRPCManagerWithDialer(ctx, newGatewayManager, bufDialer)

		gatewayID2 := ulid.MustNew(ulid.Now(), rand.Reader)
		time.Sleep(1 * time.Millisecond)
		gatewayID3 := ulid.MustNew(ulid.Now(), rand.Reader)

		gateway2 := &state.Gateway{
			Id:                gatewayID2,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.4"),
			Hostname:          "test-gateway-gc-2",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}
		gateway3 := &state.Gateway{
			Id:                gatewayID3,
			Status:            state.GatewayStatusActive,
			IPAddress:         net.ParseIP("127.0.0.5"),
			Hostname:          "test-gateway-gc-3",
			LastHeartbeatAtMS: time.Now().UnixMilli(),
		}

		err := newGatewayManager.UpsertGateway(ctx, gateway2)
		require.NoError(t, err)
		err = newGatewayManager.UpsertGateway(ctx, gateway3)
		require.NoError(t, err)

		allGateways, err := newGatewayManager.GetAllGateways(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(allGateways), "Should have 2 gateways in Redis")

		err = newForwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		require.Equal(t, 2, mockServer.getPingCount(), "Should have pinged 2 gateways")

		err = newGatewayManager.DeleteGateway(ctx, gatewayID2)
		require.NoError(t, err)
		err = newGatewayManager.DeleteGateway(ctx, gatewayID3)
		require.NoError(t, err)

		newForwarderImpl := newForwarder.(*gatewayGRPCManager)
		deletedCount, err := newForwarderImpl.GarbageCollectClients()
		require.NoError(t, err)
		require.Equal(t, 2, deletedCount, "Should have deleted exactly 2 clients")

		deletedCount, err = newForwarderImpl.GarbageCollectClients()
		require.NoError(t, err)
		require.Equal(t, 0, deletedCount, "Second garbage collection should delete nothing")
	})

	t.Run("handles errors", func(t *testing.T) {
		gatewayManager, cleanup := setupRedisGatewayManager(t)
		defer cleanup()

		forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, nil)
		forwarderImpl := forwarder.(*gatewayGRPCManager)

		cleanup()

		deletedCount, err := forwarderImpl.GarbageCollectClients()
		require.Error(t, err)
		require.Equal(t, 0, deletedCount, "Should return 0 count when error occurs")
	})
}

func TestGatewayGRPCForwarderWithFailingServer(t *testing.T) {
	ctx := context.Background()

	failingDialer := func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, errors.New("connection failed")
	}

	gatewayManager, cleanup := setupRedisGatewayManager(t)
	defer cleanup()

	gatewayID := ulid.MustNew(ulid.Now(), nil)
	gateway := &state.Gateway{
		Id:                gatewayID,
		Status:            state.GatewayStatusActive,
		IPAddress:         net.ParseIP("127.0.0.1"),
		Hostname:          "test-gateway",
		LastHeartbeatAtMS: time.Now().UnixMilli(),
	}
	err := gatewayManager.UpsertGateway(ctx, gateway)
	require.NoError(t, err)

	forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, failingDialer)

	t.Run("ConnectToGateways should ignore connection failures", func(t *testing.T) {
		err := forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)
	})

	t.Run("Forward should fail when no clients available", func(t *testing.T) {
		connectionID := ulid.MustNew(ulid.Now(), nil)
		data := &connectpb.GatewayExecutorRequestData{
			RequestId: uuid.New().String(),
		}

		err := forwarder.Forward(ctx, gatewayID, connectionID, data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "connection failed")
	})
}

func TestReply(t *testing.T) {
	ctx, _, bufDialer, gatewayManager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)
	forwarderImpl := forwarder.(*gatewayGRPCManager)

	t.Run("delivers reply to subscribed channel", func(t *testing.T) {
		requestID := "test-request-123"

		// Subscribe to receive responses
		responseCh := forwarder.Subscribe(ctx, requestID)

		// Create a reply request
		replyReq := &connectpb.ReplyRequest{
			Data: &connectpb.SDKResponse{
				RequestId:      requestID,
				AccountId:      "acc-123",
				EnvId:          "env-123",
				AppId:          "app-123",
				Status:         connectpb.SDKResponseStatus_DONE,
				Body:           []byte("test response"),
				SdkVersion:     "test-version",
				RequestVersion: 1,
			},
		}

		go func() {
			select {
			// Verify the response was delivered to the channel
			case receivedResp := <-responseCh:
				require.Equal(t, requestID, receivedResp.RequestId)
				require.Equal(t, connectpb.SDKResponseStatus_DONE, receivedResp.Status)
				require.Equal(t, []byte("test response"), receivedResp.Body)
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected response to be delivered to subscription channel")
			}
		}()

		time.Sleep(20 * time.Millisecond)

		resp, err := forwarderImpl.Reply(ctx, replyReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.Success)

		forwarder.Unsubscribe(ctx, requestID)
	})

	t.Run("returns false when no subscription exists", func(t *testing.T) {
		requestID := "non-existent-request"

		replyReq := &connectpb.ReplyRequest{
			Data: &connectpb.SDKResponse{
				RequestId: requestID,
				Status:    connectpb.SDKResponseStatus_DONE,
			},
		}

		resp, err := forwarderImpl.Reply(ctx, replyReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.Success)
	})

	t.Run("handles unsubscribed channel gracefully", func(t *testing.T) {
		requestID := "test-request-full"

		forwarder.Subscribe(ctx, requestID)

		replyReq := &connectpb.ReplyRequest{
			Data: &connectpb.SDKResponse{
				RequestId: requestID,
				Status:    connectpb.SDKResponseStatus_DONE,
			},
		}

		forwarder.Unsubscribe(ctx, requestID)

		resp, err := forwarderImpl.Reply(ctx, replyReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.Success)
	})
}

func TestSubscribeUnsubscribe(t *testing.T) {
	ctx, _, bufDialer, gatewayManager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)
	forwarderImpl := forwarder.(*gatewayGRPCManager)

	t.Run("subscription lifecycle", func(t *testing.T) {
		requestID1 := "request-1"
		requestID2 := "request-2"

		// Subscribe to two different requests
		responseCh1 := forwarder.Subscribe(ctx, requestID1)
		responseCh2 := forwarder.Subscribe(ctx, requestID2)

		// Send replies to both
		reply1 := &connectpb.ReplyRequest{
			Data: &connectpb.SDKResponse{
				RequestId: requestID1,
				Status:    connectpb.SDKResponseStatus_DONE,
				Body:      []byte("response 1"),
			},
		}

		reply2 := &connectpb.ReplyRequest{
			Data: &connectpb.SDKResponse{
				RequestId: requestID2,
				Status:    connectpb.SDKResponseStatus_DONE,
				Body:      []byte("response 2"),
			},
		}

		// Start goroutines to receive responses before sending replies
		var receivedResp1, receivedResp2 *connectpb.SDKResponse
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			select {
			case receivedResp1 = <-responseCh1:
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected response 1 to be delivered")
			}
		}()

		go func() {
			defer wg.Done()
			select {
			case receivedResp2 = <-responseCh2:
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected response 2 to be delivered")
			}
		}()

		time.Sleep(10 * time.Millisecond)

		// Both should succeed
		resp1, err := forwarderImpl.Reply(ctx, reply1)
		require.NoError(t, err)
		require.True(t, resp1.Success)

		resp2, err := forwarderImpl.Reply(ctx, reply2)
		require.NoError(t, err)
		require.True(t, resp2.Success)

		// Wait for all responses to be received
		wg.Wait()

		// Verify both responses are delivered to correct channels
		require.Equal(t, requestID1, receivedResp1.RequestId)
		require.Equal(t, []byte("response 1"), receivedResp1.Body)
		require.Equal(t, requestID2, receivedResp2.RequestId)
		require.Equal(t, []byte("response 2"), receivedResp2.Body)

		// Unsubscribe from request1
		forwarder.Unsubscribe(ctx, requestID1)

		// Sending reply to request1 should now fail
		resp1Retry, err := forwarderImpl.Reply(ctx, reply1)
		require.NoError(t, err)
		require.False(t, resp1Retry.Success)

		// But request2 should still work - start a goroutine to consume the message
		var receivedResp2Retry *connectpb.SDKResponse
		var wg2 sync.WaitGroup
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			select {
			case receivedResp2Retry = <-responseCh2:
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected response 2 to be delivered")
			}
		}()

		time.Sleep(10 * time.Millisecond)

		resp2Retry, err := forwarderImpl.Reply(ctx, reply2)
		require.NoError(t, err)
		require.True(t, resp2Retry.Success)

		// Wait for the second message to be received
		wg2.Wait()
		require.NotNil(t, receivedResp2Retry)
		require.Equal(t, requestID2, receivedResp2Retry.RequestId)

		// Clean up
		forwarder.Unsubscribe(ctx, requestID2)
	})

	t.Run("unsubscribe non-existent subscription should not panic", func(t *testing.T) {
		forwarder.Unsubscribe(ctx, "non-existent-request")
	})
}

func TestSubscribeUnsubscribeWorkerAck(t *testing.T) {
	ctx, _, bufDialer, gatewayManager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	forwarder := newGatewayGRPCManagerWithDialer(ctx, gatewayManager, bufDialer)
	forwarderImpl := forwarder.(*gatewayGRPCManager)

	t.Run("worker ack subscription lifecycle", func(t *testing.T) {
		requestID1 := "worker-ack-request-1"
		requestID2 := "worker-ack-request-2"

		// Subscribe to two different worker ack requests
		ackCh1 := forwarder.SubscribeWorkerAck(ctx, requestID1)
		ackCh2 := forwarder.SubscribeWorkerAck(ctx, requestID2)

		// Create ack messages for both
		ack1 := &connectpb.AckMessage{
			RequestId: requestID1,
			Ts:        timestamppb.Now(),
		}

		ack2 := &connectpb.AckMessage{
			RequestId: requestID2,
			Ts:        timestamppb.Now(),
		}

		// Start goroutines to receive acks before sending them
		var receivedAck1, receivedAck2 *connectpb.AckMessage
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			select {
			case receivedAck1 = <-ackCh1:
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected ack 1 to be delivered")
			}
		}()

		go func() {
			defer wg.Done()
			select {
			case receivedAck2 = <-ackCh2:
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected ack 2 to be delivered")
			}
		}()

		time.Sleep(10 * time.Millisecond)

		// Both should succeed
		resp1, err := forwarderImpl.Ack(ctx, ack1)
		require.NoError(t, err)
		require.True(t, resp1.Success)

		resp2, err := forwarderImpl.Ack(ctx, ack2)
		require.NoError(t, err)
		require.True(t, resp2.Success)

		// Wait for all acks to be received
		wg.Wait()

		// Verify both acks are delivered to correct channels
		require.Equal(t, requestID1, receivedAck1.RequestId)
		require.Equal(t, requestID2, receivedAck2.RequestId)

		// Unsubscribe from request1
		forwarder.UnsubscribeWorkerAck(ctx, requestID1)

		// Sending ack to request1 should now fail
		resp1Retry, err := forwarderImpl.Ack(ctx, ack1)
		require.NoError(t, err)
		require.False(t, resp1Retry.Success)

		// But request2 should still work
		var receivedAck2Retry *connectpb.AckMessage
		var wg2 sync.WaitGroup
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			select {
			case receivedAck2Retry = <-ackCh2:
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected ack 2 to be delivered")
			}
		}()

		time.Sleep(10 * time.Millisecond)

		resp2Retry, err := forwarderImpl.Ack(ctx, ack2)
		require.NoError(t, err)
		require.True(t, resp2Retry.Success)

		// Wait for the second ack to be received
		wg2.Wait()
		require.NotNil(t, receivedAck2Retry)
		require.Equal(t, requestID2, receivedAck2Retry.RequestId)

		// Clean up
		forwarder.UnsubscribeWorkerAck(ctx, requestID2)
	})

	t.Run("delivers ack to subscribed channel", func(t *testing.T) {
		requestID := "test-worker-ack-123"

		// Subscribe to receive worker acks
		ackCh := forwarder.SubscribeWorkerAck(ctx, requestID)

		// Create an ack message
		ackMsg := &connectpb.AckMessage{
			RequestId: requestID,
			Ts:        timestamppb.Now(),
		}

		go func() {
			select {
			// Verify the ack was delivered to the channel
			case receivedAck := <-ackCh:
				require.Equal(t, requestID, receivedAck.RequestId)
				require.NotNil(t, receivedAck.Ts)
			case <-time.After(1 * time.Second):
				require.Fail(t, "expected ack to be delivered to subscription channel")
			}
		}()

		time.Sleep(20 * time.Millisecond)

		resp, err := forwarderImpl.Ack(ctx, ackMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.Success)

		forwarder.UnsubscribeWorkerAck(ctx, requestID)
	})

	t.Run("returns false when no worker ack subscription exists", func(t *testing.T) {
		requestID := "non-existent-worker-ack-request"

		ackMsg := &connectpb.AckMessage{
			RequestId: requestID,
			Ts:        timestamppb.Now(),
		}

		resp, err := forwarderImpl.Ack(ctx, ackMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.Success)
	})

	t.Run("handles unsubscribed worker ack channel gracefully", func(t *testing.T) {
		requestID := "test-worker-ack-unsubscribed"

		forwarder.SubscribeWorkerAck(ctx, requestID)

		ackMsg := &connectpb.AckMessage{
			RequestId: requestID,
			Ts:        timestamppb.Now(),
		}

		forwarder.UnsubscribeWorkerAck(ctx, requestID)

		resp, err := forwarderImpl.Ack(ctx, ackMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.Success)
	})

	t.Run("unsubscribe non-existent worker ack subscription should not panic", func(t *testing.T) {
		forwarder.UnsubscribeWorkerAck(ctx, "non-existent-worker-ack-request")
	})
}
