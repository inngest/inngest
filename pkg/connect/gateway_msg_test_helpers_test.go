package connect

import (
	"testing"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/stretchr/testify/require"
)

func newTestConnectionHandler(t *testing.T, res testingResources) *connectionHandler {
	t.Helper()

	metadata, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
	require.NoError(t, err)
	require.NotNil(t, metadata)

	conn := &state.Connection{
		AccountID:    res.accountID,
		EnvID:        res.envID,
		ConnectionId: res.connID,
		Data:         res.reqData,
		Groups: map[string]*state.WorkerGroup{
			res.workerGroup.Hash: res.workerGroup,
		},
		GatewayId: res.svc.gatewayId,
	}

	return &connectionHandler{
		svc:            res.svc,
		conn:           conn,
		ws:             res.ws,
		log:            res.svc.logger,
		stopForwarding: make(chan struct{}),
	}
}
