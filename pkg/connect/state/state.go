package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/backoff"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/cqrs/sync"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"

	"github.com/google/uuid"
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
	RequestStateManager
}

type ConnectionManager interface {
	GetConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) (*connpb.ConnMetadata, error)
	GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByAppID(ctx context.Context, envId uuid.UUID, appID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByGroupID(ctx context.Context, envID uuid.UUID, groupID string) ([]*connpb.ConnMetadata, error)
	UpsertConnection(ctx context.Context, conn *Connection, status connpb.ConnectionStatus, lastHeartbeatAt time.Time) error
	DeleteConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) error
	GarbageCollectConnections(ctx context.Context) (int, error)
	GarbageCollectGateways(ctx context.Context) (int, error)
}

type WorkerGroupManager interface {
	GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*WorkerGroup, error)
	UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *WorkerGroup) error
}

type GatewayManager interface {
	UpsertGateway(ctx context.Context, gateway *Gateway) error
	DeleteGateway(ctx context.Context, gatewayId ulid.ULID) error
	GetGateway(ctx context.Context, gatewayId ulid.ULID) (*Gateway, error)
	GetAllGateways(ctx context.Context) ([]*Gateway, error)
	GetAllGatewayIDs(ctx context.Context) ([]string, error)
}

type RequestStateManager interface {
	// LeaseRequest attempts to lease the given requestID for <duration>. If the request is already leased, this will fail with ErrRequestLeased.
	LeaseRequest(ctx context.Context, envID uuid.UUID, requestID string, duration time.Duration, executorIP net.IP) (leaseID *ulid.ULID, err error)

	// ExtendRequestLease attempts to extend a lease for the given request. This will fail if the lease expired (ErrRequestLeaseExpired) or
	// the current lease does not match the passed leaseID (ErrRequestLeased).
	ExtendRequestLease(ctx context.Context, envID uuid.UUID, requestID string, leaseID ulid.ULID, duration time.Duration) (newLeaseID *ulid.ULID, err error)

	// IsRequestLeased checks whether the given request is currently leased and the lease has not expired.
	IsRequestLeased(ctx context.Context, envID uuid.UUID, requestID string) (bool, error)

	// DeleteLease allows the executor to clean up the lease once the request is done processing.
	DeleteLease(ctx context.Context, envID uuid.UUID, requestID string) error

	// GetExecutorIP retrieves the IP of the executor that owns the request's lease.
	GetExecutorIP(ctx context.Context, envID uuid.UUID, requestID string) (net.IP, error)

	// SaveResponse is an idempotent, atomic write for reliably buffering a response for the executor to pick up
	// in case Redis PubSub fails to notify the executor.
	SaveResponse(ctx context.Context, envID uuid.UUID, requestID string, resp *connpb.SDKResponse) error

	// GetResponse retrieves the response for a given request, if exists. Otherwise, the response will be nil.
	GetResponse(ctx context.Context, envID uuid.UUID, requestID string) (*connpb.SDKResponse, error)

	// DeleteResponse is an idempotent delete operation for the temporary response buffer.
	DeleteResponse(ctx context.Context, envID uuid.UUID, requestID string) error
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
	Id                ulid.ULID     `json:"id"`
	Status            GatewayStatus `json:"status"`
	LastHeartbeatAtMS int64         `json:"last_heartbeat"`

	Hostname string `json:"hostname"`

	IPAddress net.IP `json:"ip"`
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

// Sync attempts to sync the worker group configuration.
//
// - If a previous worker group with the same hash exists, the current worker group is updated, so App ID and Sync ID are provided.
// - In case no previous worker group with the same hash exists (or all existing connections disconnected and triggered a cleanup),
// an out-of-band Sync request is sent to the API. This is expected to handle idempotency, so subsequent calls return the same App ID and Sync ID
// given the same idempotency key.
// - To enable rollback functionality, the API should trigger a new sync if, and only if, the requested idempotency key does not match the current deploy.
func (g *WorkerGroup) Sync(ctx context.Context, groupManager WorkerGroupManager, apiBaseUrl string, initialReq *connpb.WorkerConnectRequestData, isDev bool) error {
	// The group is expected to exist in the state, as UpsertConnection also creates the group if it doesn't exist
	existingGroup, err := groupManager.GetWorkerGroupByHash(ctx, g.EnvID, g.Hash)
	if err != nil {
		return fmt.Errorf("error attempting to retrieve worker group: %w", err)
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

	appURL := connURL.String()

	// When running on the dev server, make sure to append the app name to create a deterministic UUID
	// This is necessary for multi-app connections, where each app sends an individual sync request and should
	// always use the same App ID to avoid creating duplicate apps when changing the function configuration.
	if isDev {
		appURL += fmt.Sprintf("?app_name=%s", url.QueryEscape(g.AppName))
	}

	config := sdk.RegisterRequest{
		V:          "1",
		URL:        appURL,
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

	// Retrieve the deploy ID for the sync and update state with it if available
	var syncReply sync.Reply
	for {
		attempt++

		if attempt == maxRetryAttempts {
			return fmt.Errorf("existing sync took too long to complete")
		}

		// Apply exponential backoff for retries
		if attempt > 1 {
			backOffDur := time.Until(backoff.ExponentialJitterBackoff(attempt))

			select {
			case <-ctx.Done():
				return fmt.Errorf("could not wait for sync to complete")
			case <-time.After(backOffDur):
			}
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

		// Provide environment name for branch environments
		if initialReq.GetEnvironment() != "" {
			req.Header.Set("X-Inngest-Env", initialReq.GetEnvironment())
		}

		resp, err := http.DefaultClient.Do(req)
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
				continue
			}

			// Propagate syncing errors to the user
			return connecterrors.SocketError{
				SysCode:    perr.Code,
				Msg:        perr.Message,
				StatusCode: websocket.StatusPolicyViolation,
			}
		}

		if err := json.NewDecoder(resp.Body).Decode(&syncReply); err != nil {
			if errors.Is(err, io.EOF) {
				logger.StdlibLogger(ctx).Warn(
					"got EOF for connect sync, retrying",
					"err", err,
					"conn_id", initialReq.ConnectionId,
					"account_id", g.AccountID,
					"env_id", g.EnvID,
				)
				continue
			}
			return fmt.Errorf("error parsing sync response: %w", err)
		}

		break
	}

	// We always expect the App ID & Sync ID to be included in a sync result, representing either the idempotent reply or the new sync.
	if !syncReply.IsSuccess() {
		if syncReply.Error != nil {
			return fmt.Errorf("invalid sync result: %s", *syncReply.Error)
		}

		return fmt.Errorf("invalid sync result")
	}

	g.AppID = syncReply.AppID
	g.SyncID = syncReply.SyncID

	// Update the worker group with the syncID so it's aware that it's already sync'd before
	// Always update the worker group for consistency, even if the context is cancelled
	if err := groupManager.UpdateWorkerGroup(context.Background(), g.EnvID, g); err != nil {
		return fmt.Errorf("error updating worker group: %w", err)
	}

	return nil
}
