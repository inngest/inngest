package cqrs

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type App struct {
	ID          uuid.UUID
	Name        string
	SdkLanguage string
	SdkVersion  string
	Framework   sql.NullString
	Metadata    map[string]string
	Status      string
	Error       sql.NullString
	Checksum    string
	CreatedAt   time.Time
	DeletedAt   time.Time
	Url         string
	IsConnect   sql.NullBool
}

type AppManager interface {
	AppReader
	AppWriter
}

type AppReader interface {
	// GetApps returns apps that have not been deleted.
	GetApps(ctx context.Context, envID uuid.UUID) ([]*App, error)
	// GetAppByChecksum returns an app by checksum.
	GetAppByChecksum(ctx context.Context, envID uuid.UUID, checksum string) (*App, error)
	// GetAppByURL returns an app by URL
	GetAppByURL(ctx context.Context, envID uuid.UUID, url string) (*App, error)
	// GetAppByName returns an app by name
	GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*App, error)
	// GetAllApps returns all apps.
	GetAllApps(ctx context.Context, envID uuid.UUID) ([]*App, error)

	GetAppByID(ctx context.Context, id uuid.UUID) (*App, error)
}

type AppWriter interface {
	// UpsertApp creates or updates an app.
	UpsertApp(ctx context.Context, arg UpsertAppParams) (*App, error)
	// UpdateAppError sets an app error.  A nil string
	// clears the app error.
	UpdateAppError(ctx context.Context, arg UpdateAppErrorParams) (*App, error)
	// UpdateAppURL
	UpdateAppURL(ctx context.Context, arg UpdateAppURLParams) (*App, error)
	// DeleteApp deletes an app.
	DeleteApp(ctx context.Context, id uuid.UUID) error
}

type UpsertAppParams struct {
	ID          uuid.UUID
	Name        string
	SdkLanguage string
	SdkVersion  string
	Framework   sql.NullString
	Metadata    string
	Status      string
	Error       sql.NullString
	Checksum    string
	Url         string
	IsConnect   sql.NullBool
}

type UpdateAppErrorParams struct {
	ID    uuid.UUID
	Error sql.NullString
}

type UpdateAppURLParams struct {
	ID  uuid.UUID
	Url string
}
