package base_cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	sqlc_psql "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteCQRSGetFunctionByInternalUUID(t *testing.T) {
	ctx := context.Background()

	// Generate test IDs
	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	t.Run("when function is active", func(t *testing.T) {
		fnID := uuid.New()

		// Create function config
		fnConfig := map[string]any{
			"triggers": []map[string]any{
				{"event": "test.event"},
			},
		}
		configJSON, err := json.Marshal(fnConfig)
		require.NoError(t, err)

		// Insert the function
		_, err = cm.InsertFunction(ctx, cqrs.InsertFunctionParams{
			ID:        fnID,
			AccountID: accountID,
			EnvID:     envID,
			AppID:     appID,
			Name:      "Test Function",
			Slug:      "test-function",
			Config:    string(configJSON),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		// Test GetFunctionByInternalUUID
		function, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		require.NotNil(t, function)

		// Verify function properties
		assert.Equal(t, fnID, function.ID)
		// assert.Equal(t, envID, function.EnvID)
		assert.Equal(t, appID, function.AppID)
		assert.Equal(t, "Test Function", function.Name)
		assert.Equal(t, "test-function", function.Slug)
		assert.NotEmpty(t, function.Config)
		assert.False(t, function.CreatedAt.IsZero())

		// Verify function config can be unmarshaled
		var config map[string]any
		err = json.Unmarshal(function.Config, &config)
		require.NoError(t, err)
		assert.NotEmpty(t, config["triggers"])

		// Test non-existent function
		nonExistentID := uuid.New()
		_, err = cm.GetFunctionByInternalUUID(ctx, nonExistentID)
		assert.ErrorIs(t, err, sql.ErrNoRows)

		// Function should be considered active (not archived)
		assert.False(t, function.IsArchived())

		// Verify that ArchivedAt is zero time
		assert.True(t, function.ArchivedAt.IsZero())

		// Function should be retrievable and have valid properties
		retrievedFn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		assert.Equal(t, function.ID, retrievedFn.ID)
		assert.Equal(t, function.Name, retrievedFn.Name)
		assert.Equal(t, function.Slug, retrievedFn.Slug)
	})

	t.Run("when function is deleted/archived", func(t *testing.T) {
		// Create another function to archive
		fnID := uuid.New()
		archivedFnConfig := map[string]any{
			"triggers": []map[string]any{
				{"event": "archived.event"},
			},
		}
		archivedConfigJSON, err := json.Marshal(archivedFnConfig)
		require.NoError(t, err)

		// Insert the function to be archived
		_, err = cm.InsertFunction(ctx, cqrs.InsertFunctionParams{
			AccountID: accountID,
			EnvID:     envID,
			AppID:     appID,
			ID:        fnID,
			Name:      "Archived Function",
			Slug:      "archived-function",
			Config:    string(archivedConfigJSON),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		// Archive the function by setting archived_at
		err = cm.DeleteFunctionsByIDs(ctx, []uuid.UUID{fnID})
		require.NoError(t, err)

		// Retrieve the archived function - should still be retrievable
		archivedFunction, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		require.NotNil(t, archivedFunction)

		// Verify function is marked as archived
		assert.True(t, archivedFunction.IsArchived())
		assert.False(t, archivedFunction.ArchivedAt.IsZero())

		// Verify other properties are still correct
		assert.Equal(t, fnID, archivedFunction.ID)
		assert.Equal(t, "Archived Function", archivedFunction.Name)
		assert.Equal(t, "archived-function", archivedFunction.Slug)
	})
}

//
// Helpers
//

type withInitCQRSOpt func(*initCQRSOpt)

type initCQRSOpt struct {
	appID uuid.UUID
}

func withInitCQRSOptApp(id uuid.UUID) withInitCQRSOpt {
	return func(o *initCQRSOpt) {
		o.appID = id
	}
}

func initSQLiteCQRS(t *testing.T, opts ...withInitCQRSOpt) (cqrs.Manager, func()) {
	ctx := context.Background()

	opt := initCQRSOpt{}
	for _, apply := range opts {
		apply(&opt)
	}

	db, err := New(BaseCQRSOptions{InMemory: true})
	require.NoError(t, err)

	cm := NewCQRS(db, "sqlite", sqlc_psql.NewNormalizedOpts{})

	cleanup := func() {
		db.Close()
	}

	if opt.appID != uuid.Nil {
		// Upsert the app
		_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:   opt.appID,
			Name: fmt.Sprintf("app:%s", opt.appID),
		})
		require.NoError(t, err)
	}

	return cm, cleanup
}
