package connect

import (
	"context"

	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectionHandler) handleWorkerReply(msg *connectpb.ConnectMessage) *connecterrors.SocketError {
	// Always handle SDK reply, even if gateway is draining.
	err := c.handleSdkReply(context.Background(), msg)
	if err != nil {
		c.log.ReportError(err, "could not handle sdk reply")
		// TODO Should we actually close the connection here?
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "could not handle SDK reply",
		}
	}

	return nil
}
