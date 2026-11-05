package queue

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestMigrationSleepID(t *testing.T) {
	fnID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	item := QueueItem{
		ID: "legacy-sleep-id", FunctionID: fnID,
		Data: Item{Identifier: state.Identifier{
			AccountID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			WorkspaceID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			WorkflowID:  fnID, RunID: ulid.MustParse("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		}},
	}
	// Redis and FDB must keep recognizing IDs stored by earlier migrations.
	const want = "migration-sleep-89c42185aafa5ffd9240b34718c1351a7edea1cf0a55faaeefd9472419a0b875"
	require.Equal(t, want, MigrationSleepID(item))
	item.ID = want
	require.Equal(t, want, MigrationSleepID(item))
}
