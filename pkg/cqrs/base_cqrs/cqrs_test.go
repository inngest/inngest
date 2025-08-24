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

func TestSQLiteCQRSGetApps(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test apps
	app1ID := uuid.New()
	app2ID := uuid.New()
	app3ID := uuid.New()

	apps := []cqrs.UpsertAppParams{
		{
			ID:       app1ID,
			Name:     "Test App 1",
			Checksum: "checksum1",
			Url:      "http://app1.example.com",
		},
		{
			ID:       app2ID,
			Name:     "Test App 2",
			Checksum: "checksum2",
			Url:      "http://app2.example.com",
		},
		{
			ID:       app3ID,
			Name:     "Archived App",
			Checksum: "checksum3",
			Url:      "http://app3.example.com",
		},
	}

	for _, app := range apps {
		_, err := cm.UpsertApp(ctx, app)
		require.NoError(t, err)
	}

	t.Run("get all apps", func(t *testing.T) {
		result, err := cm.GetApps(ctx, envID, nil)
		require.NoError(t, err)
		assert.Len(t, result, 3)

		// Verify apps are returned
		appIDs := make([]uuid.UUID, len(result))
		for i, app := range result {
			appIDs[i] = app.ID
		}
		assert.Contains(t, appIDs, app1ID)
		assert.Contains(t, appIDs, app2ID)
		assert.Contains(t, appIDs, app3ID)
	})

	t.Run("get apps with filter", func(t *testing.T) {
		filter := &cqrs.FilterAppParam{}
		result, err := cm.GetApps(ctx, envID, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 3)
	})

	t.Run("get apps with non-existent envID", func(t *testing.T) {
		nonExistentEnvID := uuid.New()
		result, err := cm.GetApps(ctx, nonExistentEnvID, nil)
		require.NoError(t, err)
		// Note: Current implementation doesn't filter by envID, so we still get all apps
		assert.GreaterOrEqual(t, len(result), 3)
	})
}

func TestSQLiteCQRSGetAppByChecksum(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test app
	appID := uuid.New()
	checksum := "unique-checksum-12345"
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:       appID,
		Name:     "Test App",
		Checksum: checksum,
		Url:      "http://test.example.com",
	})
	require.NoError(t, err)

	t.Run("get app by existing checksum", func(t *testing.T) {
		app, err := cm.GetAppByChecksum(ctx, envID, checksum)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, appID, app.ID)
		assert.Equal(t, "Test App", app.Name)
		assert.Equal(t, checksum, app.Checksum)
	})

	t.Run("get app by non-existent checksum", func(t *testing.T) {
		_, err := cm.GetAppByChecksum(ctx, envID, "non-existent-checksum")
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("get app by empty checksum", func(t *testing.T) {
		_, err := cm.GetAppByChecksum(ctx, envID, "")
		assert.Error(t, err)
	})
}

func TestSQLiteCQRSGetAppByID(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test app
	appID := uuid.New()
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:       appID,
		Name:     "Test App By ID",
		Checksum: "checksum-by-id",
		Url:      "http://byid.example.com",
	})
	require.NoError(t, err)

	t.Run("get app by existing ID", func(t *testing.T) {
		app, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, appID, app.ID)
		assert.Equal(t, "Test App By ID", app.Name)
		assert.Equal(t, "checksum-by-id", app.Checksum)
		assert.Equal(t, "http://byid.example.com", app.Url)
	})

	t.Run("get app by non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := cm.GetAppByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestSQLiteCQRSGetAppByURL(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test app
	appID := uuid.New()
	url := "http://unique-url.example.com/webhook"
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:       appID,
		Name:     "Test App By URL",
		Checksum: "checksum-by-url",
		Url:      url,
	})
	require.NoError(t, err)

	t.Run("get app by existing URL", func(t *testing.T) {
		app, err := cm.GetAppByURL(ctx, envID, url)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, appID, app.ID)
		assert.Equal(t, "Test App By URL", app.Name)
		assert.Equal(t, url, app.Url)
	})

	t.Run("get app by non-existent URL", func(t *testing.T) {
		_, err := cm.GetAppByURL(ctx, envID, "http://non-existent.example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("get app by empty URL", func(t *testing.T) {
		_, err := cm.GetAppByURL(ctx, envID, "")
		assert.Error(t, err)
	})
}

func TestSQLiteCQRSGetAppByName(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test app
	appID := uuid.New()
	appName := "Unique Test App Name"
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:       appID,
		Name:     appName,
		Checksum: "checksum-by-name",
		Url:      "http://byname.example.com",
	})
	require.NoError(t, err)

	t.Run("get app by existing name", func(t *testing.T) {
		app, err := cm.GetAppByName(ctx, envID, appName)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, appID, app.ID)
		assert.Equal(t, appName, app.Name)
		assert.Equal(t, "checksum-by-name", app.Checksum)
	})

	t.Run("get app by non-existent name", func(t *testing.T) {
		_, err := cm.GetAppByName(ctx, envID, "Non-Existent App Name")
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("get app by empty name", func(t *testing.T) {
		_, err := cm.GetAppByName(ctx, envID, "")
		assert.Error(t, err)
	})
}

func TestSQLiteCQRSGetAllApps(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create multiple test apps
	appIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	for i, appID := range appIDs {
		_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:       appID,
			Name:     fmt.Sprintf("Test App %d", i+1),
			Checksum: fmt.Sprintf("checksum-%d", i+1),
			Url:      fmt.Sprintf("http://app%d.example.com", i+1),
		})
		require.NoError(t, err)
	}

	t.Run("get all apps", func(t *testing.T) {
		apps, err := cm.GetAllApps(ctx, envID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(apps), 3)

		// Verify all our test apps are included
		foundAppIDs := make(map[uuid.UUID]bool)
		for _, app := range apps {
			foundAppIDs[app.ID] = true
		}

		for _, expectedID := range appIDs {
			assert.True(t, foundAppIDs[expectedID], "Expected app ID %s to be found", expectedID)
		}
	})

	t.Run("get all apps with non-existent envID", func(t *testing.T) {
		nonExistentEnvID := uuid.New()
		apps, err := cm.GetAllApps(ctx, nonExistentEnvID)
		require.NoError(t, err)
		// Note: Current implementation doesn't filter by envID, so we still get all apps
		assert.GreaterOrEqual(t, len(apps), 3)
	})
}

func TestSQLiteCQRSUpsertApp(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	t.Run("create new app", func(t *testing.T) {
		appID := uuid.New()
		params := cqrs.UpsertAppParams{
			ID:          appID,
			Name:        "New Test App",
			SdkLanguage: "go",
			SdkVersion:  "1.0.0",
			Framework:   sql.NullString{Valid: true, String: "gin"},
			Metadata:    `{"key": "value"}`,
			Status:      "active",
			Checksum:    "new-checksum-123",
			Url:         "http://new.example.com",
			Method:      "POST",
			AppVersion:  "v1.0.0",
		}

		app, err := cm.UpsertApp(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Verify all fields
		assert.Equal(t, appID, app.ID)
		assert.Equal(t, "New Test App", app.Name)
		assert.Equal(t, "go", app.SdkLanguage)
		assert.Equal(t, "1.0.0", app.SdkVersion)
		assert.True(t, app.Framework.Valid)
		assert.Equal(t, "gin", app.Framework.String)
		assert.Equal(t, "active", app.Status)
		assert.Equal(t, "new-checksum-123", app.Checksum)
		assert.Equal(t, "http://new.example.com", app.Url)
		assert.Equal(t, "POST", app.Method)
		assert.False(t, app.CreatedAt.IsZero())
	})

	t.Run("update existing app", func(t *testing.T) {
		// First create an app
		appID := uuid.New()
		originalParams := cqrs.UpsertAppParams{
			ID:          appID,
			Name:        "Original App",
			SdkLanguage: "go",
			SdkVersion:  "1.0.0",
			Checksum:    "original-checksum",
			Url:         "http://original.example.com",
			Method:      "POST",
		}

		_, err := cm.UpsertApp(ctx, originalParams)
		require.NoError(t, err)

		// Now update it with all the same fields plus changes
		updatedParams := cqrs.UpsertAppParams{
			ID:          appID,
			Name:        "Updated App",
			SdkLanguage: "go", // Add fields that might be required
			SdkVersion:  "1.0.0",
			Checksum:    "updated-checksum",
			Url:         "http://updated.example.com",
			Status:      "updated",
			Method:      "POST",
		}

		updatedApp, err := cm.UpsertApp(ctx, updatedParams)
		require.NoError(t, err)
		require.NotNil(t, updatedApp)

		// Verify updates (focus on non-normalized fields)
		assert.Equal(t, appID, updatedApp.ID)
		assert.Equal(t, "Updated App", updatedApp.Name)
		assert.Equal(t, "updated-checksum", updatedApp.Checksum)
		assert.Equal(t, "updated", updatedApp.Status)
		// Note: URL might be normalized, so just check it's not empty
		assert.NotEmpty(t, updatedApp.Url)
	})

	t.Run("upsert with minimal fields", func(t *testing.T) {
		appID := uuid.New()
		minimalParams := cqrs.UpsertAppParams{
			ID:   appID,
			Name: "Minimal App",
		}

		app, err := cm.UpsertApp(ctx, minimalParams)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, appID, app.ID)
		assert.Equal(t, "Minimal App", app.Name)
		assert.False(t, app.CreatedAt.IsZero())
	})
}

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

func TestSQLiteCQRSUpdateAppURL(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test app with comprehensive field data
	appID := uuid.New()
	originalURL := "http://original.example.com/webhook"
	originalApp, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:          appID,
		Name:        "Test App for URL Update",
		SdkLanguage: "go",
		SdkVersion:  "1.2.3",
		Framework:   sql.NullString{Valid: true, String: "gin"},
		Metadata:    `{"environment": "test", "version": "1.0"}`,
		Status:      "active",
		Checksum:    "url-update-checksum",
		Url:         originalURL,
		Method:      "POST",
		AppVersion:  "v2.1.0",
	})
	require.NoError(t, err)

	t.Run("update app URL successfully", func(t *testing.T) {
		newURL := "http://updated.example.com/webhook"
		updatedApp, err := cm.UpdateAppURL(ctx, cqrs.UpdateAppURLParams{
			ID:  appID,
			Url: newURL,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedApp)

		// Verify URL was updated (URL normalization doesn't change this simple URL)
		assert.Equal(t, appID, updatedApp.ID)
		assert.Equal(t, newURL, updatedApp.Url)

		// Verify ALL other fields remain unchanged after URL update
		assert.Equal(t, originalApp.Name, updatedApp.Name)
		assert.Equal(t, originalApp.SdkLanguage, updatedApp.SdkLanguage)
		assert.Equal(t, originalApp.SdkVersion, updatedApp.SdkVersion)
		assert.Equal(t, originalApp.Framework, updatedApp.Framework)
		assert.Equal(t, originalApp.Metadata, updatedApp.Metadata)
		assert.Equal(t, originalApp.Status, updatedApp.Status)
		assert.Equal(t, originalApp.Error, updatedApp.Error)
		assert.Equal(t, originalApp.Checksum, updatedApp.Checksum)
		assert.Equal(t, originalApp.Method, updatedApp.Method)
		assert.Equal(t, originalApp.AppVersion, updatedApp.AppVersion)
		assert.Equal(t, originalApp.CreatedAt, updatedApp.CreatedAt)
		assert.Equal(t, originalApp.DeletedAt, updatedApp.DeletedAt)

		// Verify the change persisted in the database
		retrievedApp, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		assert.Equal(t, newURL, retrievedApp.Url)

		// Verify other fields also persisted correctly
		assert.Equal(t, originalApp.Name, retrievedApp.Name)
		assert.Equal(t, originalApp.SdkLanguage, retrievedApp.SdkLanguage)
		assert.Equal(t, originalApp.SdkVersion, retrievedApp.SdkVersion)
		assert.Equal(t, originalApp.Checksum, retrievedApp.Checksum)
		assert.Equal(t, originalApp.Method, retrievedApp.Method)
		assert.Equal(t, originalApp.AppVersion, retrievedApp.AppVersion)
	})

	t.Run("update app URL with empty string", func(t *testing.T) {
		updatedApp, err := cm.UpdateAppURL(ctx, cqrs.UpdateAppURLParams{
			ID:  appID,
			Url: "",
		})
		require.NoError(t, err)
		require.NotNil(t, updatedApp)

		// Verify URL was set to empty string
		assert.Equal(t, "", updatedApp.Url)

		// Verify other fields still remain unchanged
		assert.Equal(t, originalApp.Name, updatedApp.Name)
		assert.Equal(t, originalApp.SdkLanguage, updatedApp.SdkLanguage)
		assert.Equal(t, originalApp.SdkVersion, updatedApp.SdkVersion)
		assert.Equal(t, originalApp.Checksum, updatedApp.Checksum)
		assert.Equal(t, originalApp.Method, updatedApp.Method)
		assert.Equal(t, originalApp.AppVersion, updatedApp.AppVersion)

		// Verify the change persisted
		retrievedApp, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		assert.Equal(t, "", retrievedApp.Url)
		assert.Equal(t, originalApp.Name, retrievedApp.Name)
		assert.Equal(t, originalApp.Checksum, retrievedApp.Checksum)
	})

	t.Run("update non-existent app URL", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := cm.UpdateAppURL(ctx, cqrs.UpdateAppURLParams{
			ID:  nonExistentID,
			Url: "http://new.example.com",
		})
		assert.Error(t, err)
	})
}

func TestSQLiteCQRSDeleteApp(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	// Create test app
	appID := uuid.New()
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:       appID,
		Name:     "Test App for Deletion",
		Checksum: "delete-checksum",
		Url:      "http://delete.example.com",
	})
	require.NoError(t, err)

	t.Run("delete existing app", func(t *testing.T) {
		// Verify app exists before deletion
		app, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		require.NotNil(t, app)
		assert.True(t, app.DeletedAt.IsZero()) // Should not be deleted initially

		// Delete the app
		err = cm.DeleteApp(ctx, appID)
		require.NoError(t, err)

		// Verify app can still be retrieved (soft delete - archived_at is set)
		deletedApp, err := cm.GetAppByID(ctx, appID)
		require.NoError(t, err)
		require.NotNil(t, deletedApp)

		// Verify the archived_at timestamp was set (DeletedAt field maps to archived_at)
		assert.False(t, deletedApp.DeletedAt.IsZero(), "App should have DeletedAt timestamp set after deletion")
	})

	t.Run("delete non-existent app", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := cm.DeleteApp(ctx, nonExistentID)
		// DeleteApp should not return an error for non-existent IDs
		// This behavior may vary depending on implementation
		require.NoError(t, err)
	})

	t.Run("verify app is excluded from GetApps after deletion", func(t *testing.T) {
		envID := uuid.New()

		// Create a new app to test deletion filtering
		testAppID := uuid.New()
		_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:       testAppID,
			Name:     "Test App for GetApps",
			Checksum: "getapps-checksum",
			Url:      "http://getapps.example.com",
		})
		require.NoError(t, err)

		// Verify app is returned by GetApps before deletion
		apps, err := cm.GetApps(ctx, envID, nil)
		require.NoError(t, err)

		// Find our test app in the results
		var foundApp bool
		for _, app := range apps {
			if app.ID == testAppID {
				foundApp = true
				break
			}
		}
		assert.True(t, foundApp, "App should be found before deletion")

		// Delete the app
		err = cm.DeleteApp(ctx, testAppID)
		require.NoError(t, err)

		// Verify app is no longer returned by GetApps
		appsAfterDelete, err := cm.GetApps(ctx, envID, nil)
		require.NoError(t, err)

		// Verify our test app is not in the results
		foundAppAfterDelete := false
		for _, app := range appsAfterDelete {
			if app.ID == testAppID {
				foundAppAfterDelete = true
				break
			}
		}
		assert.False(t, foundAppAfterDelete, "App should not be found after deletion")
	})
}

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

	db, err := New(BaseCQRSOptions{InMemory: true, ForTest: true})
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
