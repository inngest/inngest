package connect

import (
	"testing"

	"github.com/inngest/inngest/pkg/connect/state"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestHandleWorkerHeartbeatMissingInstanceIDIsFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	ch.conn = &state.Connection{
		AccountID:    res.accountID,
		EnvID:        res.envID,
		ConnectionId: res.connID,
		GatewayId:    res.svc.gatewayId,
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId: "",
		},
	}

	serr := ch.handleWorkerHeartbeat()
	require.NotNil(t, serr)
	require.Contains(t, serr.Msg, "missing instanceId")
}
