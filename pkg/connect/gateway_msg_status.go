package connect

import (
	"time"

	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
)

func (c *connectionHandler) handleWorkerStatus(msg *connectpb.ConnectMessage) *connecterrors.SocketError {
	if time.Since(c.getLastStatus()) < 2*time.Second {
		c.log.Trace("ignoring WORKER_STATUS, rate limited")
		return nil
	}
	c.setLastStatus(time.Now())

	var data connectpb.WorkerStatusData
	if err := proto.Unmarshal(msg.Payload, &data); err != nil {
		c.log.Warn("invalid WORKER_STATUS payload", "err", err)
		return nil
	}

	c.log.Debug("worker status received",
		"in_flight_request_count", len(data.InFlightRequestIds),
		"in_flight_request_ids", data.InFlightRequestIds,
		"shutdown_requested", data.ShutdownRequested,
	)
	return nil
}
