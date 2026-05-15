package connect

import (
	"context"
	"errors"

	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/proto"
)

func (c *connectionHandler) handleWorkerRequestExtendLease(msg *connectpb.ConnectMessage) *connecterrors.SocketError {
	var data connectpb.WorkerRequestExtendLeaseData
	if err := proto.Unmarshal(msg.Payload, &data); err != nil {
		// This should never happen: Failing the ack means we will redeliver
		// the same request even though the worker already started processing it.
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "invalid payload in worker request extend lease",
		}
	}

	leaseID, err := ulid.Parse(data.LeaseId)
	if err != nil {
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "invalid lease ID in worker request extend lease payload",
		}
	}

	workerCap, err := c.svc.stateManager.GetWorkerCapacities(context.Background(), c.conn.EnvID, c.conn.Data.InstanceId)
	if err != nil {
		c.log.ReportError(err, "failed to get worker available capacity",
			logger.WithErrorReportTags(map[string]string{
				"instance_id": c.conn.Data.InstanceId,
				"env_id":      c.conn.EnvID.String(),
			}))
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "failed to get total worker capacity",
		}
	}
	c.log.Trace("worker capacity info before extending lease", "account_id", c.conn.AccountID, "env_id", c.conn.EnvID, "instance_id", c.conn.Data.InstanceId, "worker_total_capacity", workerCap.Total, "worker_available_capacity", workerCap.Available)

	newLeaseID, err := c.svc.stateManager.ExtendRequestLease(context.Background(), c.conn.EnvID, c.conn.Data.InstanceId,
		data.RequestId, leaseID, consts.ConnectWorkerRequestLeaseDuration, workerCap.IsUnlimited())
	if err != nil {
		switch {
		case errors.Is(err, state.ErrRequestLeaseExpired),
			errors.Is(err, state.ErrRequestLeased),
			errors.Is(err, state.ErrRequestLeaseNotFound),
			errors.Is(err, state.ErrRequestWorkerDoesNotExist):

			c.log.ReportError(err, "lease was claimed by other worker, expired, or worker does not exist",
				logger.WithErrorReportTags(map[string]string{
					"req_id":   data.RequestId,
					"lease_id": leaseID.String(),
				}))

			return c.writeWorkerRequestExtendLeaseAck(&data, nil, "failed to marshal nack payload")

		default:
			c.log.ReportError(err, "unexpected error extending lease",
				logger.WithErrorReportTags(map[string]string{
					"req_id":   data.RequestId,
					"lease_id": leaseID.String(),
				}))

			// This should never happen.
			return &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "unexpected error extending lease",
			}
		}
	}

	var newLeaseIDStr *string
	if newLeaseID != nil {
		newLeaseIDStr = proto.String(newLeaseID.String())
	}

	serr := c.writeWorkerRequestExtendLeaseAck(&data, newLeaseIDStr, "failed to marshal nack payload")
	if serr != nil {
		return serr
	}

	c.log.Debug("extended lease for long-running request", "req_id", data.RequestId)
	return nil
}

func (c *connectionHandler) writeWorkerRequestExtendLeaseAck(data *connectpb.WorkerRequestExtendLeaseData, newLeaseID *string, marshalErrMsg string) *connecterrors.SocketError {
	ackPayload, marshalErr := proto.Marshal(&connectpb.WorkerRequestExtendLeaseAckData{
		RequestId:    data.RequestId,
		AccountId:    data.AccountId,
		EnvId:        data.EnvId,
		AppId:        data.AppId,
		FunctionSlug: data.FunctionSlug,
		NewLeaseId:   newLeaseID,
	})
	if marshalErr != nil {
		// This should never happen.
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        marshalErrMsg,
		}
	}

	ackWriteCtx, ackWriteCancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer ackWriteCancel()
	if err := wsproto.Write(ackWriteCtx, c.ws, &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK,
		Payload: ackPayload,
	}); err != nil {
		// The connection will fail to read and be closed in the read loop.
		return nil
	}

	return nil
}
