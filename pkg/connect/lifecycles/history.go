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

	for groupHash, group := range conn.Groups {
		// Persist history in history store
		err := h.writer.InsertWorkerConnection(ctx, &cqrs.WorkerConnection{
			AccountID:   conn.AccountID,
			WorkspaceID: conn.EnvID,

			AppName: group.AppName,
			AppID:   group.AppID,

			Id:         conn.ConnectionId,
			GatewayId:  conn.GatewayId,
			InstanceId: conn.Data.InstanceId,
			Status:     connectpb.ConnectionStatus_DISCONNECTED,
			WorkerIP:   conn.WorkerIP,

			ConnectedAt:     ulid.Time(conn.ConnectionId.Time()),
			LastHeartbeatAt: ptr.Time(time.Now()),
			DisconnectedAt:  ptr.Time(time.Now()),
			RecordedAt:      time.Now(),

			DisconnectReason: disconnectReason,

			GroupHash:     groupHash,
			SDKLang:       conn.Data.SdkLanguage,
			SDKVersion:    conn.Data.SdkVersion,
			SDKPlatform:   conn.Data.GetPlatform(),
			SyncID:        group.SyncID,
			AppVersion:    group.AppVersion,
			FunctionCount: len(group.FunctionSlugs),

			CpuCores: conn.Data.SystemAttributes.CpuCores,
			MemBytes: conn.Data.SystemAttributes.MemBytes,
			Os:       conn.Data.SystemAttributes.Os,
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not persist connection history", "error", err)
		}
	}
}

func NewHistoryLifecycle(writer cqrs.ConnectionHistoryWriter) connect.ConnectGatewayLifecycleListener {
	return &historyLifecycles{
		writer: writer,
	}
}

func (h *historyLifecycles) upsertConnection(ctx context.Context, conn *state.Connection, status connectpb.ConnectionStatus, lastHeartbeatAt time.Time) error {
	for groupHash, group := range conn.Groups {
		// Persist history in history store
		err := h.writer.InsertWorkerConnection(ctx, &cqrs.WorkerConnection{
			AccountID:   conn.AccountID,
			WorkspaceID: conn.EnvID,

			// App-specific details
			GroupHash:     groupHash,
			AppName:       group.AppName,
			AppID:         group.AppID,
			AppVersion:    group.AppVersion,
			FunctionCount: len(group.FunctionSlugs),
			SyncID:        group.SyncID,

			Id:         conn.ConnectionId,
			GatewayId:  conn.GatewayId,
			InstanceId: conn.Data.InstanceId,
			Status:     status,
			WorkerIP:   conn.WorkerIP,

			ConnectedAt:     ulid.Time(conn.ConnectionId.Time()),
			LastHeartbeatAt: ptr.Time(lastHeartbeatAt),
			DisconnectedAt:  nil,
			RecordedAt:      time.Now(),

			DisconnectReason: nil,

			SDKLang:     conn.Data.SdkLanguage,
			SDKVersion:  conn.Data.SdkVersion,
			SDKPlatform: conn.Data.GetPlatform(),

			CpuCores: conn.Data.SystemAttributes.CpuCores,
			MemBytes: conn.Data.SystemAttributes.MemBytes,
			Os:       conn.Data.SystemAttributes.Os,
		})
		if err != nil {
			return fmt.Errorf("could not persist connection history: %w", err)
		}
	}

	return nil
}
