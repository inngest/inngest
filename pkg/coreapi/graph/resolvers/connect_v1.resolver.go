package resolvers

import (
	"context"
	"fmt"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"time"
)

const (
	defaultConnectionItems = 40
	maxConnectionItems     = 400
)

func (r *connectV1workerConnectionConnResolver) App(ctx context.Context, obj *models.ConnectV1WorkerConnection) (*cqrs.App, error) {
	if obj.AppID == nil {
		return nil, nil
	}

	return r.Data.GetAppByID(ctx, *obj.AppID)
}

func (r *connectV1workerConnectionResolver) TotalCount(ctx context.Context, obj *models.WorkerConnectionsConnection) (int, error) {
	opts := toWorkerConnectionsQueryOpt(0, obj.After, obj.OrderBy, obj.Filter)
	count, err := r.Data.GetWorkerConnectionsCount(ctx, opts)
	if err != nil {
		return 0, fmt.Errorf("error retrieving count for worker connections: %w", err)
	}

	return count, nil
}

func (qr *queryResolver) WorkerConnections(ctx context.Context, first int, after *string, orderBy []*models.ConnectV1WorkerConnectionsOrderBy, filter models.ConnectV1WorkerConnectionsFilter) (*models.WorkerConnectionsConnection, error) {
	opts := toWorkerConnectionsQueryOpt(first, after, orderBy, filter)
	workerConns, err := qr.Data.GetWorkerConnections(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving worker connections: %w", err)
	}

	var (
		scursor *string
		ecursor *string
	)

	edges := []*models.ConnectV1WorkerConnectionEdge{}
	total := len(workerConns)
	for i, conn := range workerConns {
		c := conn.Cursor
		if i == 0 {
			scursor = &c // start cursor
		}
		if i == total-1 {
			ecursor = &c // end cursor
		}

		edges = append(edges, &models.ConnectV1WorkerConnectionEdge{
			Node:   connToNode(conn),
			Cursor: conn.Cursor,
		})
	}

	pageInfo := &models.PageInfo{
		HasNextPage: total == int(opts.Items),
		StartCursor: scursor,
		EndCursor:   ecursor,
	}

	return &models.WorkerConnectionsConnection{
		Edges:    edges,
		PageInfo: pageInfo,
		After:    after,
		Filter:   filter,
		OrderBy:  orderBy,
	}, nil
}

func connToNode(conn *cqrs.WorkerConnection) *models.ConnectV1WorkerConnection {
	var (
		disconnectedAt  *time.Time
		lastHeartbeatAt *time.Time
	)

	if conn.DisconnectedAt != nil && conn.DisconnectedAt.UnixMilli() > 0 {
		disconnectedAt = conn.DisconnectedAt
	}
	if conn.LastHeartbeatAt != nil && conn.LastHeartbeatAt.UnixMilli() > 0 {
		lastHeartbeatAt = conn.LastHeartbeatAt
	}

	var status models.ConnectV1ConnectionStatus
	switch conn.Status {
	case connpb.ConnectionStatus_READY:
		status = models.ConnectV1ConnectionStatusReady
	case connpb.ConnectionStatus_CONNECTED:
		status = models.ConnectV1ConnectionStatusConnected
	case connpb.ConnectionStatus_DRAINING:
		status = models.ConnectV1ConnectionStatusDraining
	case connpb.ConnectionStatus_DISCONNECTING:
		status = models.ConnectV1ConnectionStatusDisconnecting
	case connpb.ConnectionStatus_DISCONNECTED:
		status = models.ConnectV1ConnectionStatusDisconnected
	}

	node := &models.ConnectV1WorkerConnection{
		AppName: &conn.AppName,
		AppID:   conn.AppID,

		ID:         conn.Id,
		GatewayID:  conn.GatewayId,
		InstanceID: conn.InstanceId,
		Status:     status,
		WorkerIP:   conn.WorkerIP,

		ConnectedAt:     conn.ConnectedAt,
		LastHeartbeatAt: lastHeartbeatAt,
		DisconnectedAt:  disconnectedAt,

		DisconnectReason: conn.DisconnectReason,

		GroupHash:   conn.GroupHash,
		SdkLang:     conn.SDKLang,
		SdkVersion:  conn.SDKVersion,
		SdkPlatform: conn.SDKPlatform,
		SyncID:      conn.SyncID,
		BuildID:     conn.AppVersion,
		AppVersion:  conn.AppVersion,

		FunctionCount: conn.FunctionCount,

		CPUCores: int(conn.CpuCores),
		MemBytes: int(conn.MemBytes),
		Os:       conn.Os,
	}

	return node
}

func (qr *queryResolver) WorkerConnection(ctx context.Context, connectionID ulid.ULID) (*models.ConnectV1WorkerConnection, error) {
	conn, err := qr.Data.GetWorkerConnection(ctx, cqrs.WorkerConnectionIdentifier{
		AccountID:    consts.DevServerAccountID,
		WorkspaceID:  consts.DevServerEnvID,
		ConnectionID: connectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving worker connection: %w", err)
	}

	return connToNode(conn), nil
}

func toWorkerConnectionsQueryOpt(
	num int,
	cur *string,
	order []*models.ConnectV1WorkerConnectionsOrderBy,
	filter models.ConnectV1WorkerConnectionsFilter,
) cqrs.GetWorkerConnectionOpt {
	tsfield := enums.WorkerConnectionTimeFieldConnectedAt
	switch *filter.TimeField {
	case models.ConnectV1WorkerConnectionsOrderByFieldLastHeartbeatAt:
		tsfield = enums.WorkerConnectionTimeFieldLastHeartbeatAt
	case models.ConnectV1WorkerConnectionsOrderByFieldDisconnectedAt:
		tsfield = enums.WorkerConnectionTimeFieldDisconnectedAt
	}

	statuses := []connpb.ConnectionStatus{}
	if len(filter.Status) > 0 {
		for _, s := range filter.Status {
			var status connpb.ConnectionStatus
			switch s {
			case models.ConnectV1ConnectionStatusReady:
				status = connpb.ConnectionStatus_READY
			case models.ConnectV1ConnectionStatusConnected:
				status = connpb.ConnectionStatus_CONNECTED
			case models.ConnectV1ConnectionStatusDraining:
				status = connpb.ConnectionStatus_DRAINING
			case models.ConnectV1ConnectionStatusDisconnecting:
				status = connpb.ConnectionStatus_DISCONNECTING
			case models.ConnectV1ConnectionStatusDisconnected:
				status = connpb.ConnectionStatus_DISCONNECTED
			default:
				// unknown status
				continue
			}
			statuses = append(statuses, status)
		}
	}

	orderBy := []cqrs.GetWorkerConnectionOrder{}
	for _, o := range order {
		var (
			field enums.WorkerConnectionTimeField
			dir   enums.WorkerConnectionSortOrder
		)

		switch o.Field {
		case models.ConnectV1WorkerConnectionsOrderByFieldConnectedAt:
			field = enums.WorkerConnectionTimeFieldConnectedAt
		case models.ConnectV1WorkerConnectionsOrderByFieldLastHeartbeatAt:
			field = enums.WorkerConnectionTimeFieldLastHeartbeatAt
		case models.ConnectV1WorkerConnectionsOrderByFieldDisconnectedAt:
			field = enums.WorkerConnectionTimeFieldDisconnectedAt
		default: // unknown, skip
			continue
		}

		switch o.Direction {
		case models.ConnectV1WorkerConnectionsOrderByDirectionAsc:
			dir = enums.WorkerConnectionSortOrderAsc
		case models.ConnectV1WorkerConnectionsOrderByDirectionDesc:
			dir = enums.WorkerConnectionSortOrderDesc
		default: // unknown, skip
			continue
		}

		orderBy = append(orderBy, cqrs.GetWorkerConnectionOrder{Field: field, Direction: dir})
	}

	var cursor string
	if cur != nil {
		cursor = *cur
	}

	from := time.Time{}
	if filter.From != nil {
		from = *filter.From
	}

	until := time.Now()
	if filter.Until != nil {
		until = *filter.Until
	}

	items := defaultConnectionItems
	if num > 0 && num < maxConnectionItems {
		items = num
	}

	return cqrs.GetWorkerConnectionOpt{
		Filter: cqrs.GetWorkerConnectionFilter{
			AccountID:   consts.DevServerAccountID,
			WorkspaceID: consts.DevServerEnvID,
			AppID:       filter.AppIDs,
			TimeField:   tsfield,
			From:        from,
			Until:       until,
			Status:      statuses,
		},
		Order:  orderBy,
		Cursor: cursor,
		Items:  uint(items),
	}
}
