package connect

import (
	"testing"

	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestHandleIncomingWebSocketMessageUnknownKindIsIgnored(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{})
	require.Nil(t, serr)
}
