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

//
// App
//

// func TestSQLiteCQRSGetApps(t *testing.T) {}
// func TestSQLiteCQRSGetAppByChecksum(t *testing.T) {}
// func TestSQLiteCQRSGetAppByID(t *testing.T) {}
// func TestSQLiteCQRSGetAppByURL(t *testing.T) {}
// func TestSQLiteCQRSGetAppByName(t *testing.T) {}
// func TestSQLiteCQRSGetAllApps(t *testing.T) {}
// func TestSQLiteCQRSUpsertApp(t *testing.T) {}

func TestSQLiteCQRSUpdateAppError(t *testing.T) {
	ctx := context.Background()

	// Generate test IDs
	appID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	t.Run("set app error", func(t *testing.T) {
		// Get the original app
		originalApp, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		require.NotNil(t, originalApp)

		// Verify initially no error
		assert.False(t, originalApp.Error.Valid)
		assert.Empty(t, originalApp.Error.String)

		// Update app with an error
		errorMessage := "Test error message"
		updatedApp, err := cm.UpdateAppError(ctx, cqrs.UpdateAppErrorParams{
			ID:    appID,
			Error: sql.NullString{Valid: true, String: errorMessage},
		})
		require.NoError(t, err)
		require.NotNil(t, updatedApp)

		// Verify error was set
		assert.True(t, updatedApp.Error.Valid)
		assert.Equal(t, errorMessage, updatedApp.Error.String)
		assert.Equal(t, appID, updatedApp.ID)

		// Verify other fields remain unchanged
		assert.Equal(t, originalApp.ID, updatedApp.ID)
		assert.Equal(t, originalApp.Name, updatedApp.Name)
		assert.Equal(t, originalApp.Checksum, updatedApp.Checksum)

		// Verify the change persisted by getting the app again
		retrievedApp, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		assert.True(t, retrievedApp.Error.Valid)
		assert.Equal(t, errorMessage, retrievedApp.Error.String)
	})

	t.Run("clear app error", func(t *testing.T) {
		// First set an error
		errorMessage := "Initial error"
		_, err := cm.UpdateAppError(ctx, cqrs.UpdateAppErrorParams{
			ID:    appID,
			Error: sql.NullString{Valid: true, String: errorMessage},
		})
		require.NoError(t, err)

		// Now clear the error
		updatedApp, err := cm.UpdateAppError(ctx, cqrs.UpdateAppErrorParams{
			ID:    appID,
			Error: sql.NullString{Valid: false, String: ""},
		})
		require.NoError(t, err)
		require.NotNil(t, updatedApp)

		// Verify error was cleared
		assert.False(t, updatedApp.Error.Valid)
		assert.Empty(t, updatedApp.Error.String)

		// Verify the change persisted
		retrievedApp, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		assert.False(t, retrievedApp.Error.Valid)
		assert.Empty(t, retrievedApp.Error.String)
	})

	t.Run("update non-existent app", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := cm.UpdateAppError(ctx, cqrs.UpdateAppErrorParams{
			ID:    nonExistentID,
			Error: sql.NullString{Valid: true, String: "error"},
		})
		assert.Error(t, err)
	})
}

// func TestSQLiteCQRSUpdateAppURL(t *testing.T) {}
// func TestSQLiteCQRSDeleteApp(t *testing.T) {}

//
// Function
//

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
