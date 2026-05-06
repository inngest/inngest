package cqrs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
)

type Function struct {
	ID         uuid.UUID       `json:"internal_id"`
	EnvID      uuid.UUID       `json:"env_id"`
	AppID      uuid.UUID       `json:"app_id"`
	Slug       string          `json:"id"`
	Name       string          `json:"name"`
	Config     json.RawMessage `json:"config"`
	CreatedAt  time.Time       `json:"created_at"`
	ArchivedAt time.Time       `json:"archived_at"`
}

func (f Function) InngestFunction() (*inngest.Function, error) {
	fn := inngest.Function{}
	err := json.Unmarshal([]byte(f.Config), &fn)
	if err != nil {
		return nil, err
	}
	return &fn, nil
}

func (f Function) IsArchived() bool {
	if f.ArchivedAt.After(time.Time{}) && time.Now().After(f.ArchivedAt) {
		return true
	}
	return false
}

// FunctionReader finds functions for use across the API and dev server.
type FunctionReader interface {
	// GetFunctionsByAppInternalID returns functions given the string ID of an app as defined
	// by users in our SDKs.
	GetFunctionsByAppExternalID(ctx context.Context, workspaceID uuid.UUID, app string) ([]*Function, error)
	// GetFunctionsByAppInternalID returns functions given an internal app UUID.
	GetFunctionsByAppInternalID(ctx context.Context, appID uuid.UUID) ([]*Function, error)
	// GetFunctionByExternalID returns a function given a workspace ID and the SDK's client ID / function ID,
	// defined as a string.
	GetFunctionByExternalID(ctx context.Context, wsID uuid.UUID, appID string, functionID string) (*Function, error)
	// GetFunctionByInternalUUID returns a function given the internal ID.
	GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*Function, error)
	// GetActiveFunctionByAppAndSlug returns an active (non-archived) function
	// given its parent app's user-facing name and the function's slug.
	//
	// Keyed on app name (not internal UUID) because the dev server derives app
	// UUIDs from different inputs across paths — placeholder rows hash the URL,
	// post-sync rows hash the name — so a UUID-keyed lookup can miss the row
	// the caller actually wants. The name is the one identifier that's stable
	// across both paths.
	//
	// (app_id, slug) remains the natural identity at the storage layer (see the
	// partial unique index `functions_app_id_slug_active_key`); this method is
	// the caller-facing lookup that resolves to the right `app_id` via name.
	GetActiveFunctionByAppAndSlug(ctx context.Context, appName string, slug string) (*Function, error)
}

// DevFunctionManager is a development-only function manager
type DevFunctionManager interface {
	// Embeds production & API related functionality.

	FunctionReader

	// Also embeds the development functionality.

	DevFunctionReader
	DevFunctionWriter
}

// FunctionCreator creates functions in the backing store.
type FunctionCreator interface {
	UpsertFunction(ctx context.Context, params UpsertFunctionParams) (*Function, error)
	UpdateFunctionConfig(ctx context.Context, arg UpdateFunctionConfigParams) (*Function, error)
}

// DevFunctionReader is a development-only function reader.
type DevFunctionReader interface {
	GetFunctions(ctx context.Context) ([]*Function, error)
}

type DevFunctionWriter interface {
	FunctionCreator

	// DeleteFunctionsByAppID deletes all functions for a specific app.
	DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error
	// DeleteFunctionsByIDs deletes all functions with the given IDs
	DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error
}

type UpsertFunctionParams struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	EnvID     uuid.UUID
	AppID     uuid.UUID
	Name      string
	Slug      string
	Config    string
	CreatedAt time.Time
}

type UpdateFunctionConfigParams struct {
	Config    string
	ID        uuid.UUID
	AccountID uuid.UUID
	EnvID     uuid.UUID
	AppID     uuid.UUID
}
