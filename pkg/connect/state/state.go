package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/sdk"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type StateManager interface {
	ConnectionManager
	WorkerGroupManager
	GatewayManager

	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
}

type ConnectionManager interface {
	GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByAppID(ctx context.Context, envId uuid.UUID, appID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByGroupID(ctx context.Context, envID uuid.UUID, groupID string) ([]*connpb.ConnMetadata, error)
	UpsertConnection(ctx context.Context, conn *Connection) error
	DeleteConnection(ctx context.Context, envID uuid.UUID, appID *uuid.UUID, groupID string, connId string) error
}

type WorkerGroupManager interface {
	GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*WorkerGroup, error)
	UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *WorkerGroup) error
}

type GatewayManager interface {
	UpsertGateway(ctx context.Context, gateway *Gateway) error
	DeleteGateway(ctx context.Context, gatewayId string) error
	GetGateway(ctx context.Context, gatewayId string) (*Gateway, error)
}

type AuthContext struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

type SyncData struct {
	Functions []sdk.SDKFunction
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
	SDKLang     string `json:"sdk_lang"`
	SDKVersion  string `json:"sdk_version"`
	SDKPlatform string `json:"sdk_platform"`

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
	// - SDKPlatform - can be empty string
	// - Function Configurations
	// - User provided identifier (e.g. git sha, release tag, etc)
	Hash string `json:"hash"`

	// used for syncing
	SyncData SyncData `json:"-"`
}

type GatewayStatus string

const (
	GatewayStatusStarting GatewayStatus = "starting"
	GatewayStatusActive   GatewayStatus = "active"
	GatewayStatusDraining GatewayStatus = "draining"
)

type Gateway struct {
	Id              string        `json:"id"`
	Status          GatewayStatus `json:"status"`
	LastHeartbeatAt time.Time     `json:"last_heartbeat_at"`

	Hostname string `json:"hostname"`
}

// Connection have all the metadata assocaited with a worker connection
type Connection struct {
	Status    connpb.ConnectionStatus
	Data      *connpb.WorkerConnectRequestData
	Session   *connpb.SessionDetails
	Group     *WorkerGroup
	GatewayId string
}

// Sync attempts to sync the worker group configuration
//
// TODO:
// this should be dedupped when there's a large number of workers coming up at once
// so it doesn't attempt to sync multiple times prior to the worker group getting
// a response
func (c *Connection) Sync(ctx context.Context, groupManager WorkerGroupManager) error {
	if c.Group == nil {
		return fmt.Errorf("worker group is required for syncing")
	}

	// Check state to see if group already exists
	var envID uuid.UUID
	{
		id, err := uuid.Parse(c.Data.AuthData.EnvId)
		if err != nil {
			return fmt.Errorf("error parsing environment ID: %w", err)
		}
		envID = id
	}

	// The group is expected to exist in the state, as UpsertConnection also creates the group if it doesn't exist
	group, err := groupManager.GetWorkerGroupByHash(ctx, envID, c.Group.Hash)
	if err != nil {
		return fmt.Errorf("error attempting to retrieve worker group: %w", err)
	}

	// Don't attempt to sync if it's already sync'd
	if group != nil && group.SyncID != nil && group.AppID != nil {
		c.Group = group
		return nil
	}

	// Construct sync request via off-band sync
	// Can't do in-band
	connURL := url.URL{Scheme: "ws", Host: "connect"}
	sdkVersion := fmt.Sprintf("%s:%s", c.Group.SDKLang, c.Group.SDKVersion)

	var cap sdk.Capabilities
	if err := json.Unmarshal(c.Data.Config.Capabilities, &cap); err != nil {
		return fmt.Errorf("error deserializing sync capabilities: %w", err)
	}

	config := sdk.RegisterRequest{
		V:          "1",
		URL:        connURL.String(),
		DeployType: sdk.DeployTypeConnect,
		SDK:        sdkVersion,
		AppName:    c.Data.GetAppName(),
		Headers: sdk.Headers{
			Env:      c.Data.GetEnvironment(),
			Platform: c.Data.GetPlatform(),
		},
		Capabilities: cap,
		Functions:    c.Group.SyncData.Functions,
	}

	// NOTE: pick this up via SDK
	// technically it should only be accessible to the system that's the gateway is associated with
	registerURL := fmt.Sprintf("%s/fn/register", c.Data.Config.ApiOrigin)

	byt, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing function config: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, registerURL, bytes.NewReader(byt))
	if err != nil {
		return fmt.Errorf("error creating new sync request: %w", err)
	}

	// Set basic headers
	req.Header.Set(headers.HeaderContentType, "application/json")
	req.Header.Set(headers.HeaderKeySDK, sdkVersion)
	req.Header.Set(headers.HeaderUserAgent, sdkVersion)

	hashedSigningKey := string(c.Data.AuthData.HashedSigningKey)
	if hashedSigningKey == "" {
		return fmt.Errorf("no signing key available for syncing")
	}
	req.Header.Set(headers.HeaderAuthorization, fmt.Sprintf("Bearer %s", hashedSigningKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making sync request: %w", err)
	}

	// Retrieve the deploy ID for the sync and update state with it if available
	var syncReply cqrs.SyncReply
	if err := json.NewDecoder(resp.Body).Decode(&syncReply); err != nil {
		return fmt.Errorf("error parsing sync response: %w", err)
	}

	// Update the worker group to make sure it store the appropriate IDs
	if syncReply.IsSuccess() {
		c.Group.SyncID = syncReply.SyncID
		c.Group.AppID = syncReply.AppID
		// Update the worker group with the syncID so it's aware that it's already sync'd before
		// Always update the worker group for consistency, even if the context is cancelled
		if err := groupManager.UpdateWorkerGroup(context.Background(), envID, c.Group); err != nil {
			return fmt.Errorf("error updating worker group: %w", err)
		}
	}

	return nil
}
