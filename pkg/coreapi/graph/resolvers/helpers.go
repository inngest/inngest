package resolvers

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

const (
	pkgName = "coreapi.graph.resolvers"

	defaultPageSize        = 40
	defaultRunItems        = 40
	maxRunItems            = 400
	defaultConnectionItems = 40
	maxConnectionItems     = 400
)

type EventsV2ConnectionCursor struct {
	ID string
}

func (c *EventsV2ConnectionCursor) Decode(val string) error {
	byt, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, c)
}

func cqrsEventToGQLEvent(e *cqrs.Event) *models.EventV2 {
	eventV2 := models.EventV2{
		EnvID:          e.WorkspaceID,
		ID:             e.InternalID(),
		IdempotencyKey: &e.EventID,
		Name:           e.EventName,
		OccurredAt:     time.UnixMilli(e.EventTS),
		ReceivedAt:     e.ReceivedAt,
		Version:        &e.EventVersion,
	}

	if e.SourceID != nil {
		eventV2.Source = &models.EventSource{
			ID:         e.SourceID.String(),
			Name:       &e.Source,
			SourceKind: "TODO",
		}
	}

	return &eventV2
}

func marshalRaw(e *cqrs.Event) (string, error) {
	data := e.EventData
	if data == nil {
		data = make(map[string]any)
	}

	var version *string
	if len(e.EventVersion) > 0 {
		version = &e.EventVersion
	}

	id := e.InternalID().String()
	if len(e.EventID) > 0 {
		id = e.EventID
	}

	byt, err := json.Marshal(map[string]any{
		"data": data,
		"id":   id,
		"name": e.EventName,
		"ts":   e.EventTS,
		"v":    version,
	})
	if err != nil {
		return "", err
	}
	return string(byt), nil
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

		ID:                   conn.Id,
		GatewayID:            conn.GatewayId,
		InstanceID:           conn.InstanceId,
		Status:               status,
		WorkerIP:             conn.WorkerIP,
		MaxWorkerConcurrency: conn.MaxWorkerConcurrency,

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

func toRunsQueryOpt(
	first int,
	after *string,
	order []*models.RunsV2OrderBy,
	filter models.RunsFilterV2,
	preview *bool,
) cqrs.GetTraceRunOpt {
	tsfield := enums.TraceRunTimeQueuedAt
	switch *filter.TimeField {
	case models.RunsV2OrderByFieldStartedAt:
		tsfield = enums.TraceRunTimeStartedAt
	case models.RunsV2OrderByFieldEndedAt:
		tsfield = enums.TraceRunTimeEndedAt
	}

	statuses := []enums.RunStatus{}
	if len(filter.Status) > 0 {
		for _, s := range filter.Status {
			var status enums.RunStatus
			switch s {
			case models.FunctionRunStatusQueued:
				status = enums.RunStatusScheduled
			case models.FunctionRunStatusRunning:
				status = enums.RunStatusRunning
			case models.FunctionRunStatusCompleted:
				status = enums.RunStatusCompleted
			case models.FunctionRunStatusCancelled:
				status = enums.RunStatusCancelled
			case models.FunctionRunStatusFailed:
				status = enums.RunStatusFailed
			default:
				// unknown status
				continue
			}
			statuses = append(statuses, status)
		}
	}

	orderBy := []cqrs.GetTraceRunOrder{}
	for _, o := range order {
		var (
			field enums.TraceRunTime
			dir   enums.TraceRunOrder
		)

		switch o.Field {
		case models.RunsV2OrderByFieldQueuedAt:
			field = enums.TraceRunTimeQueuedAt
		case models.RunsV2OrderByFieldStartedAt:
			field = enums.TraceRunTimeStartedAt
		case models.RunsV2OrderByFieldEndedAt:
			field = enums.TraceRunTimeEndedAt
		default: // unknown, skip
			continue
		}

		switch o.Direction {
		case models.RunsOrderByDirectionAsc:
			dir = enums.TraceRunOrderAsc
		case models.RunsOrderByDirectionDesc:
			dir = enums.TraceRunOrderDesc
		default: // unknown, skip
			continue
		}

		orderBy = append(orderBy, cqrs.GetTraceRunOrder{Field: field, Direction: dir})
	}

	var cursor string
	if after != nil {
		cursor = *after
	}

	var cel string
	if filter.Query != nil {
		cel = *filter.Query
	}

	until := time.Now()
	if filter.Until != nil {
		until = *filter.Until
	}

	items := defaultRunItems
	if first > 0 && first < maxRunItems {
		items = first
	}

	return cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AppID:      filter.AppIDs,
			FunctionID: filter.FunctionIDs,
			TimeField:  tsfield,
			From:       filter.From,
			Until:      until,
			Status:     statuses,
			CEL:        cel,
		},
		Order:   orderBy,
		Cursor:  cursor,
		Items:   uint(items),
		Preview: preview == nil || *preview,
	}
}
