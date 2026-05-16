package connect

import (
	"context"

	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectionHandler) handleWorkerReady() *connecterrors.SocketError {
	// Do not allow marking worker as ready when gateway is draining.
	if c.svc.isDraining.Load() {
		c.log.Warn("ignoring worker ready as svc is draining",
			"instance_id", c.conn.Data.InstanceId,
			"env_id", c.conn.EnvID.String(),
			"account_id", c.conn.AccountID.String(),
			"gateway_id", c.conn.GatewayId.String(),
			"connection_id", c.conn.ConnectionId.String(),
			"conn_draining", c.draining.Load(),
		)

		return &ErrDraining
	}

	if c.draining.Load() {
		c.log.Warn("ignoring worker ready as connection is marked as draining")
		return nil
	}

	err := c.updateConnStatus(connectpb.ConnectionStatus_READY, "worker ready message")
	if serr := c.handleConnStatusUpdateResult(err, "failed to update connection status after worker ready"); serr != nil {
		return serr
	}

	if c.conn.Data.InstanceId == "" {
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "missing instanceId for connect message",
		}
	}

	if err := c.svc.stateManager.WorkerCapacityOnHeartbeat(context.Background(), c.conn.EnvID, c.conn.Data.InstanceId); err != nil {
		c.log.ReportError(err, "failed to refresh worker capacity TTL on heartbeat",
			logger.WithErrorReportTags(map[string]string{
				"instance_id":   util.SanitizeLogField(c.conn.Data.InstanceId),
				"env_id":        c.conn.EnvID.String(),
				"account_id":    c.conn.AccountID.String(),
				"gateway_id":    c.conn.GatewayId.String(),
				"connection_id": c.conn.ConnectionId.String(),
			}))
	}

	for _, l := range c.svc.lifecycles {
		go l.OnReady(context.Background(), c.conn)
	}

	c.log.Debug("marked worker as ready")

	return nil
}
