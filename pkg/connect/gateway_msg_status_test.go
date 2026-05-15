package connect

import (
	"testing"

	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestHandleWorkerStatusAcceptsValidPayload(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	payload, err := proto.Marshal(&connectpb.WorkerStatusData{
		InFlightRequestIds: []string{"req-1", "req-2"},
		ShutdownRequested:  true,
	})
	require.NoError(t, err)

	serr := ch.handleWorkerStatus(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_STATUS,
		Payload: payload,
	})
	require.Nil(t, serr)
	require.False(t, ch.getLastStatus().IsZero())
}

func TestHandleWorkerStatusInvalidPayloadIsIgnored(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerStatus(&connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_STATUS,
		Payload: []byte("invalid protobuf"),
	})
	require.Nil(t, serr)
}
