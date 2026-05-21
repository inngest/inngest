package connect

import (
	"context"

	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectionHandler) handleWorkerPause() *connecterrors.SocketError {
	// NOTE: Unlike WORKER_READY, we intentionally do NOT reject WORKER_PAUSE
	// when the gateway is draining. The worker is signaling that it wants to
	// stop receiving new requests, so the router must stop selecting it.
	if c.svc.isDraining.Load() {
		c.log.Warn("worker pause signal received during draining sequence")
	}

	c.draining.Store(true)

	err := c.updateConnStatus(connectpb.ConnectionStatus_DRAINING, "worker pause message", "svc_draining", c.svc.isDraining.Load())
	if err != nil {
		c.log.Error("could not update connection status to DRAINING on WORKER_PAUSE",
			"err", err,
		)
	}

	c.svc.wsConnections.Delete(c.conn.ConnectionId.String())
	c.stopForwardingOnce.Do(func() { close(c.stopForwarding) })

	for _, l := range c.svc.lifecycles {
		go l.OnStartDraining(context.Background(), c.conn)
	}

	return nil
}
