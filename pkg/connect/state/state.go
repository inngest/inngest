package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/sdk"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type ConnectionStateManager interface {
	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
	GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByAppID(ctx context.Context, appID uuid.UUID) ([]*connpb.ConnMetadata, error)
	AddConnection(ctx context.Context, conn *Connection) error
	DeleteConnection(ctx context.Context, connID string) error
	GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*WorkerGroup, error)
	UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *WorkerGroup) error
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

// Connection have all the metadata assocaited with a worker connection
type Connection struct {
	Data    *connpb.WorkerConnectRequestData
	Session *connpb.SessionDetails
	Group   *WorkerGroup
}

func (c *Connection) Sync(ctx context.Context) error {
	if c.Group == nil {
		return fmt.Errorf("worker group is required for syncing")
	}

	// TODO: Check state to see if group already exists

	// Construct sync request via off-band sync
	// Can't do in-band
	connURL := url.URL{Scheme: "ws", Host: "connect"}
	sdkVersion := fmt.Sprintf("%s:%s", c.Group.SDKLang, c.Group.SDKVersion)

	config := sdk.RegisterRequest{
		V:          "1",
		URL:        connURL.String(),
		DeployType: "ping", // TODO: should allow 'connect' as an input
		SDK:        sdkVersion,
		AppName:    c.Data.GetAppName(),
		Headers: sdk.Headers{
			Env:      c.Data.GetEnvironment(),
			Platform: c.Data.GetPlatform(),
		},
		Capabilities: sdk.Capabilities{},
		UseConnect:   true, // NOTE: probably not needed if `DeployType` can have `connect` as input?
		Functions:    c.Group.SyncData.Functions,
	}

	// NOTE: pick this up via SDK
	// technically it should only be accessible to the system that's the gateway is associated with
	apiOrigin := "http://127.0.0.1:8288"
	registerURL := fmt.Sprintf("%s/fn/register", apiOrigin)

	byt, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing function config: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, registerURL, bytes.NewReader(byt))
	if err != nil {
		return fmt.Errorf("error creating new sync request: %w", err)
	}

	// Set basic headers
	// TODO: use constants for these header keys
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inngest-SDK", sdkVersion)
	req.Header.Set("User-Agent", sdkVersion)

	hashedSigningKey := string(c.Data.AuthData.HashedSigningKey)
	if hashedSigningKey == "" {
		return fmt.Errorf("no signing key available for syncing")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", hashedSigningKey))

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making sync request: %w", err)
	}

	// TODO:
	// - retrieve the deploy ID for the sync and update state with it

	return nil
}
