package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/sdk"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

const (
	pkgName = "connect.state"
)

type StateManager interface {
	ConnectionManager
	WorkerGroupManager
	GatewayManager

	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
}

type ConnectionManager interface {
	GetConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) (*connpb.ConnMetadata, error)
	GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByAppID(ctx context.Context, envId uuid.UUID, appID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByGroupID(ctx context.Context, envID uuid.UUID, groupID string) ([]*connpb.ConnMetadata, error)
	UpsertConnection(ctx context.Context, conn *Connection, status connpb.ConnectionStatus, lastHeartbeatAt time.Time) error
	DeleteConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) error
}

type WorkerGroupManager interface {
	GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*WorkerGroup, error)
	UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *WorkerGroup) error
}

type GatewayManager interface {
	UpsertGateway(ctx context.Context, gateway *Gateway) error
	DeleteGateway(ctx context.Context, gatewayId ulid.ULID) error
	GetGateway(ctx context.Context, gatewayId ulid.ULID) (*Gateway, error)
}

type AuthContext struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

type SyncData struct {
	SyncToken string
	AppConfig *connpb.AppConfiguration
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

	AppName string `json:"app_name"`

	// User-supplied app version
	AppVersion *string `json:"app_version,omitempty"`

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

	// CreatedAt records the time this worker group was first created
	CreatedAt time.Time `json:"created_at"`

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
	Id              ulid.ULID     `json:"id"`
	Status          GatewayStatus `json:"status"`
	LastHeartbeatAt time.Time     `json:"last_heartbeat_at"`

	Hostname string `json:"hostname"`
}

// Connection have all the metadata associated with a worker connection
type Connection struct {
	AccountID    uuid.UUID
	EnvID        uuid.UUID
	ConnectionId ulid.ULID
	WorkerIP     string

	Data      *connpb.WorkerConnectRequestData
	Groups    map[string]*WorkerGroup
	GatewayId ulid.ULID
}

func (c *Connection) AppNames() []string {
	appNames := make([]string, len(c.Data.Apps))
	for i, app := range c.Data.Apps {
		appNames[i] = app.AppName
	}
	return appNames
}

// Sync attempts to sync the worker group configuration
func (g *WorkerGroup) Sync(ctx context.Context, groupManager WorkerGroupManager, apiBaseUrl string, initialReq *connpb.WorkerConnectRequestData) error {
	// The group is expected to exist in the state, as UpsertConnection also creates the group if it doesn't exist
	existingGroup, err := groupManager.GetWorkerGroupByHash(ctx, g.EnvID, g.Hash)
	if err != nil {
		return fmt.Errorf("error attempting to retrieve worker group: %w", err)
	}

	if existingGroup != nil {
		fmt.Println("SYNC: Found existing group", g.AppName, g.Hash, existingGroup.AppID, existingGroup.SyncID)
	}
	// Don't attempt to sync if it's already sync'd
	if existingGroup != nil && existingGroup.SyncID != nil && existingGroup.AppID != nil {
		g.AppID = existingGroup.AppID
		g.SyncID = existingGroup.SyncID
		g.CreatedAt = existingGroup.CreatedAt
		return nil
	}

	start := time.Now()
	defer func() {
		metrics.HistogramConnectSyncDuration(ctx, time.Since(start).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
		})
	}()

	// Construct sync request via off-band sync
	// Can't do in-band
	connURL := url.URL{Scheme: "ws", Host: "connect"}
	sdkVersion := fmt.Sprintf("%s:%s", g.SDKLang, g.SDKVersion)

	var cap sdk.Capabilities
	if err := json.Unmarshal(initialReq.Capabilities, &cap); err != nil {
		return fmt.Errorf("error deserializing sync capabilities: %w", err)
	}

	appVersion := ""
	if g.AppVersion != nil {
		appVersion = *g.AppVersion
	}

	config := sdk.RegisterRequest{
		V:          "1",
		URL:        connURL.String(),
		DeployType: sdk.DeployTypeConnect,
		SDK:        sdkVersion,
		AppName:    g.AppName,
		Headers: sdk.Headers{
			Env:      initialReq.GetEnvironment(),
			Platform: initialReq.GetPlatform(),
		},
		Capabilities: cap,
		Functions:    g.SyncData.Functions,
		AppVersion:   appVersion,

		// Deduplicate syncs in case multiple workers are coming up at the same time
		IdempotencyKey: g.Hash,
	}

	// NOTE: pick this up via SDK
	// technically it should only be accessible to the system that's the gateway is associated with
	registerURL := fmt.Sprintf("%s/fn/register", apiBaseUrl)

	byt, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing function config: %w", err)
	}

	maxRetryAttempts := 10
	attempt := 0

	var resp *http.Response
	for {
		if attempt == maxRetryAttempts {
			return fmt.Errorf("existing sync took too long to complete")
		}

		req, err := http.NewRequest(http.MethodPost, registerURL, bytes.NewReader(byt))
		if err != nil {
			return fmt.Errorf("error creating new sync request: %w", err)
		}

		// Set basic headers
		req.Header.Set(headers.HeaderContentType, "application/json")
		req.Header.Set(headers.HeaderKeySDK, sdkVersion)
		req.Header.Set(headers.HeaderUserAgent, sdkVersion)

		// Use sync token returned by initial Connect API handshake
		if g.SyncData.SyncToken != "" {
			req.Header.Set(headers.HeaderAuthorization, fmt.Sprintf("Bearer %s", g.SyncData.SyncToken))
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error making sync request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			var perr publicerr.Error
			if err := json.NewDecoder(resp.Body).Decode(&perr); err != nil {
				return fmt.Errorf("error parsing public err response: %w", err)
			}

			// Wait for other sync to complete and retry
			if perr.Code == syscode.CodeSyncAlreadyPending {
				select {
				case <-ctx.Done():
					return fmt.Errorf("could not wait for sync to complete")
				case <-time.After(2 * time.Second):
				}
				attempt++
				continue
			}
		}

		break
	}

	// Retrieve the deploy ID for the sync and update state with it if available
	var syncReply cqrs.SyncReply
	if err := json.NewDecoder(resp.Body).Decode(&syncReply); err != nil {
		return fmt.Errorf("error parsing sync response: %w", err)
	}

	// Update the worker group to make sure it store the appropriate IDs
	if !syncReply.IsSuccess() {
		return fmt.Errorf("invalid sync result")
	}

	g.AppID = syncReply.AppID
	g.SyncID = syncReply.SyncID

	if existingGroup != nil {
		fmt.Println("SYNC SUCCESS", g.AppName, g.Hash, g.AppID, g.SyncID)
	}

	// Update the worker group with the syncID so it's aware that it's already sync'd before
	// Always update the worker group for consistency, even if the context is cancelled
	if err := groupManager.UpdateWorkerGroup(context.Background(), g.EnvID, g); err != nil {
		return fmt.Errorf("error updating worker group: %w", err)
	}

	return nil
}
