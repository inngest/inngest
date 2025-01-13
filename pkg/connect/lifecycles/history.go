package lifecycles

import (
	"context"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"time"
)

type historyLifecycles struct {
	writer cqrs.ConnectionHistoryWriter
}

func (h *historyLifecycles) OnReady(ctx context.Context, conn *state.Connection) {
	err := h.upsertConnection(ctx, conn, connectpb.ConnectionStatus_READY, time.Now())
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
	}
}

func (h *historyLifecycles) OnStartDisconnecting(ctx context.Context, conn *state.Connection) {
	err := h.upsertConnection(ctx, conn, connectpb.ConnectionStatus_DISCONNECTING, time.Now())
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
	}
}

func (h *historyLifecycles) OnStartDraining(ctx context.Context, conn *state.Connection) {
	err := h.upsertConnection(ctx, conn, connectpb.ConnectionStatus_DRAINING, time.Now())
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
	}
}

func (h *historyLifecycles) OnHeartbeat(ctx context.Context, conn *state.Connection) {
	err := h.upsertConnection(ctx, conn, connectpb.ConnectionStatus_READY, time.Now())
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
	}
}

func (h *historyLifecycles) OnSynced(ctx context.Context, conn *state.Connection) {
	// No-op
}

func (h *historyLifecycles) OnConnected(ctx context.Context, conn *state.Connection) {
	err := h.upsertConnection(ctx, conn, connectpb.ConnectionStatus_CONNECTED, time.Now())
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
	}
}

func (h *historyLifecycles) OnDisconnected(ctx context.Context, conn *state.Connection, closeReason string) {
	var disconnectReason *string
	if closeReason != "" {
		disconnectReason = ptr.String(closeReason)
	}

	// Persist history in history store
	err := h.writer.InsertWorkerConnection(ctx, &cqrs.WorkerConnection{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.EnvID,
		AppID:       conn.Group.AppID,

		Id:         conn.ConnectionId,
		GatewayId:  conn.GatewayId,
		InstanceId: conn.Session.SessionId.GetInstanceId(),
		Status:     connectpb.ConnectionStatus_DISCONNECTED,

		ConnectedAt:     ulid.Time(conn.ConnectionId.Time()),
		LastHeartbeatAt: ptr.Time(time.Now()),
		DisconnectedAt:  ptr.Time(time.Now()),
		RecordedAt:      time.Now(),

		DisconnectReason: disconnectReason,

		GroupHash:     conn.Group.Hash,
		SDKLang:       conn.Group.SDKLang,
		SDKVersion:    conn.Group.SDKVersion,
		SDKPlatform:   conn.Group.SDKPlatform,
		SyncID:        conn.Group.SyncID,
		BuildId:       conn.Session.SessionId.BuildId,
		FunctionCount: len(conn.Group.FunctionSlugs),

		CpuCores: conn.Data.SystemAttributes.CpuCores,
		MemBytes: conn.Data.SystemAttributes.MemBytes,
		Os:       conn.Data.SystemAttributes.Os,
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
	}
}

func NewHistoryLifecycle(writer cqrs.ConnectionHistoryWriter) connect.ConnectGatewayLifecycleListener {
	return &historyLifecycles{
		writer: writer,
	}
}

func (h *historyLifecycles) upsertConnection(ctx context.Context, conn *state.Connection, status connectpb.ConnectionStatus, lastHeartbeatAt time.Time) error {
	// Persist history in history store
	// TODO Should the implementation use a messaging system like NATS for batching internally?
	err := h.writer.InsertWorkerConnection(ctx, &cqrs.WorkerConnection{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.EnvID,
		AppID:       conn.Group.AppID,

		Id:         conn.ConnectionId,
		GatewayId:  conn.GatewayId,
		InstanceId: conn.Session.SessionId.GetInstanceId(),
		Status:     status,

		ConnectedAt:     ulid.Time(conn.ConnectionId.Time()),
		LastHeartbeatAt: ptr.Time(lastHeartbeatAt),
		DisconnectedAt:  nil,
		RecordedAt:      time.Now(),

		DisconnectReason: nil,

		GroupHash:     conn.Group.Hash,
		SDKLang:       conn.Group.SDKLang,
		SDKVersion:    conn.Group.SDKVersion,
		SDKPlatform:   conn.Group.SDKPlatform,
		BuildId:       conn.Session.SessionId.BuildId,
		FunctionCount: len(conn.Group.FunctionSlugs),
		SyncID:        conn.Group.SyncID,

		CpuCores: conn.Data.SystemAttributes.CpuCores,
		MemBytes: conn.Data.SystemAttributes.MemBytes,
		Os:       conn.Data.SystemAttributes.Os,
	})
	if err != nil {
		return fmt.Errorf("could not persist connection history: %w", err)
	}

	return nil
}
