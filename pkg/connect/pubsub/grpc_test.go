package pubsub

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
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

	forwarder := NewGatewayGRPCForwarderWithDialer(ctx, gatewayManager, bufDialer)

	t.Run("connects to single gateway", func(t *testing.T) {
		mockServer.reset()

		err := forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		require.Equal(t, 1, mockServer.getPingCount(), "Ping should be called once during ConnectToGateways")
		require.Equal(t, 0, mockServer.getForwardCount(), "Forward should not be called during ConnectToGateways")
	})

	t.Run("connects to multiple gateways", func(t *testing.T) {
		mockServer.reset()

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

	forwarder := NewGatewayGRPCForwarderWithDialer(ctx, gatewayManager, bufDialer)

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
		newForwarder := NewGatewayGRPCForwarderWithDialer(ctx, newGatewayManager, bufDialer)

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

		forwarder := NewGatewayGRPCForwarderWithDialer(ctx, gatewayManager, bufDialer)

		err = forwarder.ConnectToGateways(ctx)
		require.NoError(t, err)

		err = gatewayManager.DeleteGateway(ctx, gatewayID)
		require.NoError(t, err)

		forwarderImpl := forwarder.(*gatewayGRPCForwarder)
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
		newForwarder := NewGatewayGRPCForwarderWithDialer(ctx, newGatewayManager, bufDialer)

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

		newForwarderImpl := newForwarder.(*gatewayGRPCForwarder)
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

		forwarder := NewGatewayGRPCForwarderWithDialer(ctx, gatewayManager, nil)
		forwarderImpl := forwarder.(*gatewayGRPCForwarder)

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

	forwarder := NewGatewayGRPCForwarderWithDialer(ctx, gatewayManager, failingDialer)

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
