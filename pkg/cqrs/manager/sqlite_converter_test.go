package manager

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainEventParsesAccountAndWorkspaceID(t *testing.T) {
	acctID := uuid.New()
	wsID := uuid.New()

	evt := domainEvent(&dbpkg.Event{
		InternalID:  ulid.Make(),
		AccountID:   sql.NullString{String: acctID.String(), Valid: true},
		WorkspaceID: sql.NullString{String: wsID.String(), Valid: true},
		EventID:     "evt-1",
		EventName:   "test/event",
		EventData:   `{"key":"value"}`,
		EventUser:   `{}`,
		EventTs:     time.Now(),
	})

	require.NotNil(t, evt)
	assert.Equal(t, acctID, evt.AccountID, "AccountID should be parsed from sql.NullString")
	assert.Equal(t, wsID, evt.WorkspaceID, "WorkspaceID should be parsed from sql.NullString")
}

func TestDomainEventNullAccountAndWorkspaceID(t *testing.T) {
	evt := domainEvent(&dbpkg.Event{
		InternalID:  ulid.Make(),
		AccountID:   sql.NullString{Valid: false},
		WorkspaceID: sql.NullString{Valid: false},
		EventID:     "evt-2",
		EventName:   "test/event",
		EventData:   `{}`,
		EventUser:   `{}`,
		EventTs:     time.Now(),
	})

	require.NotNil(t, evt)
	assert.Equal(t, uuid.UUID{}, evt.AccountID, "NULL AccountID should be zero UUID")
	assert.Equal(t, uuid.UUID{}, evt.WorkspaceID, "NULL WorkspaceID should be zero UUID")
}
