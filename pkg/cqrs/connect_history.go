package cqrs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	connpb "github.com/inngest/inngest/proto/gen/connect/v1"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// WorkerConnection represents a worker connection at a point in time, for a single app.
// If a connection serves multiple apps, we will create multiple one connection update per app.
type WorkerConnection struct {
	AccountID   uuid.UUID `json:"account_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`

	// This is always set, even if the app is not synced yet.
	AppName string `json:"app_name"`

	// This is optional and only set when connection is ready
	AppID *uuid.UUID `json:"app_id"`

	// Connection attributes
	Id                   ulid.ULID               `json:"id"`
	GatewayId            ulid.ULID               `json:"gateway_id"`
	InstanceId           string                  `json:"instance_id"`
	Status               connpb.ConnectionStatus `json:"status"`
	WorkerIP             string                  `json:"worker_ip"`
	MaxWorkerConcurrency int64                   `json:"max_worker_concurrency"`

	// Timestamps
	ConnectedAt     time.Time  `json:"connected_at"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
	DisconnectedAt  *time.Time `json:"disconnected_at"`

	// Meta fields for tracking history ingestion
	RecordedAt time.Time `json:"recorded_at"`
	InsertedAt time.Time `json:"inserted_at"`

	DisconnectReason *string `json:"disconnect_reason"`

	// Group attributes
	GroupHash     string     `json:"group_hash"`
	SDKLang       string     `json:"sdk_lang"`
	SDKVersion    string     `json:"sdk_version"`
	SDKPlatform   string     `json:"sdk_platform"`
	SyncID        *uuid.UUID `json:"sync_id,omitempty"`
	AppVersion    *string    `json:"app_version,omitempty"`
	FunctionCount int        `json:"function_count"`

	// System attributes
	CpuCores int32  `json:"cpu_cores"`
	MemBytes int64  `json:"mem_bytes"`
	Os       string `json:"os"`

	// Cursor is a composite cursor used for pagination
	Cursor string `json:"cursor"`
}

type ConnectionHistoryReadWriter interface {
	ConnectionHistoryWriter
	ConnectionHistoryReader
}

type ConnectionHistoryWriter interface {
	// InsertWorkerConnection writes a worker connection into the backing store
	InsertWorkerConnection(ctx context.Context, wc *WorkerConnection) error
}

type ConnectionHistoryReader interface {
	// GetWorkerConnections retrieves a list of WorkerConnection based on the options specified
	GetWorkerConnections(ctx context.Context, opt GetWorkerConnectionOpt) ([]*WorkerConnection, error)
	// GetWorkerConnectionsCount returns the total number of items applicable to the specified filter
	GetWorkerConnectionsCount(ctx context.Context, opt GetWorkerConnectionOpt) (int, error)
	// GetWorkerConnection retrieve the specified worker connection
	GetWorkerConnection(ctx context.Context, id WorkerConnectionIdentifier) (*WorkerConnection, error)
}

type GetWorkerConnectionOpt struct {
	Filter GetWorkerConnectionFilter
	Order  []GetWorkerConnectionOrder
	Cursor string
	Items  uint
}

type GetWorkerConnectionFilter struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID

	AppID     []uuid.UUID
	TimeField enums.WorkerConnectionTimeField
	From      time.Time
	Until     time.Time
	Status    []connpb.ConnectionStatus
}

type GetWorkerConnectionOrder struct {
	Field     enums.WorkerConnectionTimeField
	Direction enums.WorkerConnectionSortOrder
}

type WorkerConnectionIdentifier struct {
	AccountID    uuid.UUID
	WorkspaceID  uuid.UUID
	ConnectionID ulid.ULID
}

// WorkerConnectionPageCursor represents the composite cursor used to handle pagination
type WorkerConnectionPageCursor struct {
	ID      string                            `json:"id"`
	Cursors map[string]WorkerConnectionCursor `json:"c"`
}

func (c *WorkerConnectionPageCursor) IsEmpty() bool {
	return len(c.Cursors) == 0
}

// Find finds a cusor with the provided name
func (c *WorkerConnectionPageCursor) Find(field string) *WorkerConnectionCursor {
	if c.IsEmpty() {
		return nil
	}

	f := strings.ToLower(field)
	if v, ok := c.Cursors[f]; ok {
		return &v
	}
	return nil
}

func (c *WorkerConnectionPageCursor) Add(field string) {
	f := strings.ToLower(field)
	if _, ok := c.Cursors[f]; !ok {
		c.Cursors[f] = WorkerConnectionCursor{Field: f}
	}
}

func (c *WorkerConnectionPageCursor) Encode() (string, error) {
	byt, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(byt), nil
}

func (c *WorkerConnectionPageCursor) Decode(val string) error {
	if c.Cursors == nil {
		c.Cursors = map[string]WorkerConnectionCursor{}
	}
	byt, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, c)
}

// WorkerConnectionCursor represents a cursor that is used as part of the pagination cursor
type WorkerConnectionCursor struct {
	// Field represents the field used for this cursor
	Field string `json:"f"`
	// Value represents the value used for this cursor
	Value int64 `json:"v"`
}
