package resolvers

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"time"
)

func (r *connectV1workerConnectionConnResolver) App(ctx context.Context, obj *models.ConnectV1WorkerConnection) (*cqrs.App, error) {
	if obj.AppID == nil {
		return nil, nil
	}

	return r.Data.GetAppByID(ctx, *obj.AppID)
}

func (r *connectV1workerConnectionResolver) TotalCount(ctx context.Context, obj *models.ConnectV1WorkerConnectionsConnection) (int, error) {
	cursor, ok := graphql.GetFieldContext(ctx).Parent.Args["after"].(*string)
	if !ok {
		return 0, fmt.Errorf("failed to access cursor")
	}

	orderBy, ok := graphql.GetFieldContext(ctx).Parent.Args["orderBy"].([]*models.ConnectV1WorkerConnectionsOrderBy)
	if !ok {
		return 0, fmt.Errorf("failed to retrieve order")
	}

	filter, ok := graphql.GetFieldContext(ctx).Parent.Args["filter"].(models.ConnectV1WorkerConnectionsFilter)
	if !ok {
		return 0, fmt.Errorf("failed to access query filter")
	}

	opts := toWorkerConnectionsQueryOpt(0, cursor, orderBy, filter)
	count, err := r.Data.GetWorkerConnectionsCount(ctx, opts)
	if err != nil {
		return 0, fmt.Errorf("error retrieving count for worker connections: %w", err)
	}

	return count, nil
}

func (a queryResolver) WorkerConnections(ctx context.Context, first int, after *string, orderBy []*models.ConnectV1WorkerConnectionsOrderBy, filter models.ConnectV1WorkerConnectionsFilter) (*models.ConnectV1WorkerConnectionsConnection, error) {
	//TODO implement me
	panic("implement me")
}

func (a queryResolver) WorkerConnection(ctx context.Context, connectionID ulid.ULID) (*models.ConnectV1WorkerConnection, error) {
	//TODO implement me
	panic("implement me")
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

	until := time.Now()
	if filter.Until != nil {
		until = *filter.Until
	}

	items := defaultRunItems
	if num > 0 && num < maxRunItems {
		items = num
	}

	return cqrs.GetWorkerConnectionOpt{
		Filter: cqrs.GetWorkerConnectionFilter{
			AccountID:   consts.DevServerAccountId,
			WorkspaceID: consts.DevServerEnvId,
			AppID:       filter.AppIDs,
			TimeField:   tsfield,
			From:        filter.From,
			Until:       until,
			Status:      statuses,
		},
		Order:  orderBy,
		Cursor: cursor,
		Items:  uint(items),
	}
}
