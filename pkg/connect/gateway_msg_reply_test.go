package connect

import (
	"testing"

	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

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
