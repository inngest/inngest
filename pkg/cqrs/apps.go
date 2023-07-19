package cqrs

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
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
}

type AppManager interface {
	AppReader
	AppWriter
}

type AppReader interface {
	// GetApps returns apps that have not been deleted.
	GetApps(ctx context.Context) ([]*App, error)
	// GetAppByChecksum returns an app by checksum.
	GetAppByChecksum(ctx context.Context, checksum string) (*App, error)
	// GetAppByURL returns an app by URL
	GetAppByURL(ctx context.Context, url string) (*App, error)
	// GetAllApps returns all apps.
	GetAllApps(ctx context.Context) ([]*App, error)
}

type AppWriter interface {
	// InsertApp creates a new app.
	InsertApp(ctx context.Context, arg InsertAppParams) (*App, error)
	// UpdateAppError sets an app error.  A nil string
	// clears the app error.
	UpdateAppError(ctx context.Context, arg UpdateAppErrorParams) (*App, error)
	// UpdateAppURL
	UpdateAppURL(ctx context.Context, arg UpdateAppURLParams) (*App, error)
	// DeleteApp deletes an app.
	DeleteApp(ctx context.Context, id uuid.UUID) error
}

type InsertAppParams struct {
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
}

type UpdateAppErrorParams struct {
	ID    uuid.UUID
	Error sql.NullString
}

type UpdateAppURLParams struct {
	ID  uuid.UUID
	Url string
}

// NormalizeAppURL normalizes localhost and 127.0.0.1 as the same string.  This
// ensures that we don't add duplicate apps.
func NormalizeAppURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}

	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		return u
	}

	switch host {
	case "localhost", "127.0.0.1", "0.0.0.0":
		parsed.Host = fmt.Sprintf("localhost:%s", port)
		return parsed.String()
	default:
		return u
	}
}
