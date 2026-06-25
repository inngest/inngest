package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
)

const (
	SessionKeysLimit = 100
	SessionsLimit    = 100
	SessionRunsLimit = 500
)

type SessionKey struct {
	SessionKey string
	CreatedAt  time.Time
}

type SessionFunction struct {
	Slug string
	Name string
}

type SessionGroup struct {
	SessionKey     string
	SessionID      string
	RunCount       int
	FailedRunCount int
	FailureRate    float64
	LastActiveAt   time.Time
	Functions      []SessionFunction
}

type SessionRun struct {
	ID           ulid.ULID
	FunctionSlug string
	EventName    *string
	Status       enums.RunStatus
	QueuedAt     time.Time
	StartedAt    *time.Time
	EndedAt      *time.Time
}

type SessionKeyRecord struct {
	WorkspaceID uuid.UUID
	Key         string
}

type SessionTimeRange struct {
	From  time.Time
	Until time.Time
}

type SessionManager interface {
	RecordSessionKeys(ctx context.Context, workspaceID uuid.UUID, eventSessions event.Sessions) error
	GetSessionKeys(ctx context.Context, workspaceID uuid.UUID, search string) ([]*SessionKey, error)
	GetSessions(ctx context.Context, workspaceID uuid.UUID, sessionKey string, sessionIDSearch string, tr SessionTimeRange) ([]*SessionGroup, error)
	GetSessionRuns(ctx context.Context, workspaceID uuid.UUID, sessionKey string, sessionID string, tr SessionTimeRange) ([]*SessionRun, error)
}
