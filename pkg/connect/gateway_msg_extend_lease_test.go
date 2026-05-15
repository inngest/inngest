package connect

import (
	"testing"

	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

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
