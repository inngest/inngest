package connect

import (
	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/syscode"
)

const maxConsecutiveConnStatusUpdateFailures = 3

func (c *connectionHandler) handleConnStatusUpdateResult(err error, reportMsg string) *connecterrors.SocketError {
	if err == nil {
		c.consecutiveConnStatusUpdateFailures.Store(0)
		return nil
	}

	failures := c.consecutiveConnStatusUpdateFailures.Add(1)
	c.log.With(
		"consecutive_failures", failures,
		"max_consecutive_failures", maxConsecutiveConnStatusUpdateFailures,
	).ReportError(err, reportMsg)

	if failures < maxConsecutiveConnStatusUpdateFailures {
		return nil
	}

	return &connecterrors.SocketError{
		SysCode:    syscode.CodeConnectInternal,
		StatusCode: websocket.StatusInternalError,
		Msg:        "could not update connection status",
	}
}
