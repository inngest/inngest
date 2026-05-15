package connect

import (
	"context"
	"time"

	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectionHandler) handleWorkerHeartbeat() *connecterrors.SocketError {
	status := connectpb.ConnectionStatus_READY
	if c.svc.isDraining.Load() || c.draining.Load() {
		status = connectpb.ConnectionStatus_DRAINING

		c.log.Warn("worker heartbeat received during draining sequence",
			"conn_draining", c.draining.Load(),
			"svc_draining", c.svc.isDraining.Load(),
		)
	}

	err := c.updateConnStatus(status, "worker heartbeat",
		"conn_draining", c.draining.Load(),
		"svc_draining", c.svc.isDraining.Load(),
	)
	if serr := c.handleConnStatusUpdateResult(err, "failed to update connection status after heartbeat"); serr != nil {
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
		go l.OnHeartbeat(context.Background(), c.conn)
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer writeCancel()
	if err := wsproto.Write(writeCtx, c.ws, &connectpb.ConnectMessage{
		Kind: connectpb.GatewayMessageType_GATEWAY_HEARTBEAT,
	}); err != nil {
		// The connection will fail to read and be closed in the read loop.
		return nil
	}

	c.setLastHeartbeat(time.Now())
	c.log.Trace("worker heartbeat processed", "status", status.String())

	return nil
}
