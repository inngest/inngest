package state

import (
	"context"

	"github.com/google/uuid"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type ConnectionStateManager interface {
	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
	GetConnectionsByEnvID(ctx context.Context, wsID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByAppID(ctx context.Context, appID uuid.UUID) ([]*connpb.ConnMetadata, error)
	AddConnection(ctx context.Context, data *connpb.WorkerConnectRequestData, sessionDetails *connpb.SessionDetails) error
	DeleteConnection(ctx context.Context, connID string) error
}

type AuthContext struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

// WorkerGroup groups a list of connected workers to simplify operations, which
// otherwise could be cumbersome if handled individually
type WorkerGroup struct {
	// Identifiers
	AccountID uuid.UUID `json:"acct_id"`
	EnvID     uuid.UUID `json:"env_id"`
	// AppID represents the app that this worker group is associated with.
	// If set, it means this worker group is already synced
	AppID *uuid.UUID `json:"app_id,omitempty"`

	// Typical metadata associated with the SDK
	SDKLang    string `json:"sdk_lang"`
	SDKVersion string `json:"sdk_version"`

	// FunctionSlugs stores the list of slugs of functions associated by the workers
	// This allows the gateway to know what functions the workers in this group can handle,
	// allowing smarter routing to different versions of workers
	FunctionSlugs []string `json:"fns"`

	// SyncID is the ID of the "deploy" for this group
	// If this is set, it's expected that this worker group is already synced
	SyncID *uuid.UUID `json:"sync_id,omitempty"`

	// Hash is the hashed value for the SDK attributes
	// - AccountID
	// - EnvID
	// - SDKLang
	// - SDKVersion
	// - Function Configurations
	// - User provided identifier (e.g. git sha, release tag, etc)
	Hash string `json:"hash"`

	// used for syncing
	// hashedSigningKey string
}

func NewWorkerGroupFromConnRequest(ctx context.Context, req *connpb.WorkerConnectRequestData) (*WorkerGroup, error) {
	return nil, notImplementedError
}

// Sync handles the sync of the worker group
func (g *WorkerGroup) Sync(ctx context.Context) error {
	// TODO:
	// - checks state to see if it's already synced
	// - if not, then attempt a sync
	// - retrieve the deploy ID for the sync and update state with it

	return notImplementedError
}
