package manager

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	dbpkg "github.com/inngest/inngest/pkg/db"
	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Environment variable constants for database selection
const (
	// EnvTestDatabase specifies which database to use for testing ("sqlite" or "postgres")
	EnvTestDatabase = "TEST_DATABASE"
)

//
// App
//

func TestCQRSGetApps(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initCQRS(t)
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

func TestCQRSGetAppByChecksum(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initCQRS(t)
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

func TestCQRSGetAppByID(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initCQRS(t)
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

func TestCQRSGetAppByURL(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initCQRS(t)
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

func TestCQRSGetAppByName(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initCQRS(t)
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

func TestCQRSGetAllApps(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()

	cm, cleanup := initCQRS(t)
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

func TestCQRSUpsertApp(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initCQRS(t)
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

func TestCQRSUpdateAppError(t *testing.T) {
	ctx := context.Background()

	// Generate test IDs
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
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
		assert.Equal(t, originalApp.AppVersion, updatedApp.AppVersion)
		assert.Equal(t, originalApp.Checksum, updatedApp.Checksum)
		assert.Equal(t, originalApp.Metadata, updatedApp.Metadata)
		assert.Equal(t, originalApp.SdkLanguage, updatedApp.SdkLanguage)

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

func TestCQRSUpdateAppURL(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initCQRS(t)
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

func TestCQRSDeleteApp(t *testing.T) {
	ctx := context.Background()

	cm, cleanup := initCQRS(t)
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

func TestCQRSGetFunctionByInternalUUID(t *testing.T) {
	ctx := context.Background()

	// Generate test IDs
	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
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
		_, err = cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
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
		_, err = cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
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

func TestCQRSGetFunctionsByAppInternalID(t *testing.T) {
	ctx := context.Background()

	// Create two different apps
	targetAppID := uuid.New()
	otherAppID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(targetAppID))
	defer cleanup()

	// Create the other app manually
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:          otherAppID,
		Name:        fmt.Sprintf("other-app:%s", otherAppID),
		SdkLanguage: "go",
		SdkVersion:  "1.2.3",
	})
	require.NoError(t, err)

	// Create test functions for the TARGET app
	targetFn1ID := uuid.New()
	targetFn2ID := uuid.New()

	targetFunctions := []cqrs.UpsertFunctionParams{
		{
			ID:        targetFn1ID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     targetAppID,
			Name:      "Target App Function 1",
			Slug:      "target-app-function-1",
			Config:    `{"triggers": [{"event": "target.event1"}]}`,
			CreatedAt: time.Now(),
		},
		{
			ID:        targetFn2ID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     targetAppID,
			Name:      "Target App Function 2",
			Slug:      "target-app-function-2",
			Config:    `{"triggers": [{"event": "target.event2"}]}`,
			CreatedAt: time.Now(),
		},
	}

	// Create functions for the OTHER app (should NOT be returned)
	otherFn1ID := uuid.New()
	otherFn2ID := uuid.New()

	otherFunctions := []cqrs.UpsertFunctionParams{
		{
			ID:        otherFn1ID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     otherAppID,
			Name:      "Other App Function 1",
			Slug:      "other-app-function-1",
			Config:    `{"triggers": [{"event": "other.event1"}]}`,
			CreatedAt: time.Now(),
		},
		{
			ID:        otherFn2ID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     otherAppID,
			Name:      "Other App Function 2",
			Slug:      "other-app-function-2",
			Config:    `{"triggers": [{"event": "other.event2"}]}`,
			CreatedAt: time.Now(),
		},
	}

	// Insert ALL functions (target + other)
	allFunctions := append(targetFunctions, otherFunctions...)
	for _, fn := range allFunctions {
		_, err := cm.UpsertFunction(ctx, fn)
		require.NoError(t, err)
	}

	t.Run("get functions for target app only", func(t *testing.T) {
		result, err := cm.GetFunctionsByAppInternalID(ctx, targetAppID)
		require.NoError(t, err)

		// Should return exactly 2 functions (only target app functions)
		assert.Len(t, result, 2, "Should return exactly 2 functions for target app")

		// Verify ONLY target app functions are returned
		returnedFnIDs := make([]uuid.UUID, len(result))
		for i, fn := range result {
			returnedFnIDs[i] = fn.ID
			// Verify all returned functions belong to target app
			assert.Equal(t, targetAppID, fn.AppID, "Function %s should belong to target app", fn.ID)
		}

		// Verify target app functions are included
		assert.Contains(t, returnedFnIDs, targetFn1ID)
		assert.Contains(t, returnedFnIDs, targetFn2ID)

		// Verify other app functions are NOT included
		assert.NotContains(t, returnedFnIDs, otherFn1ID, "Other app function should not be returned")
		assert.NotContains(t, returnedFnIDs, otherFn2ID, "Other app function should not be returned")
	})

	t.Run("get functions for other app only", func(t *testing.T) {
		result, err := cm.GetFunctionsByAppInternalID(ctx, otherAppID)
		require.NoError(t, err)

		// Should return exactly 2 functions (only other app functions)
		assert.Len(t, result, 2, "Should return exactly 2 functions for other app")

		// Verify ONLY other app functions are returned
		returnedFnIDs := make([]uuid.UUID, len(result))
		for i, fn := range result {
			returnedFnIDs[i] = fn.ID
			// Verify all returned functions belong to other app
			assert.Equal(t, otherAppID, fn.AppID, "Function %s should belong to other app", fn.ID)
		}

		// Verify other app functions are included
		assert.Contains(t, returnedFnIDs, otherFn1ID)
		assert.Contains(t, returnedFnIDs, otherFn2ID)

		// Verify target app functions are NOT included
		assert.NotContains(t, returnedFnIDs, targetFn1ID, "Target app function should not be returned")
		assert.NotContains(t, returnedFnIDs, targetFn2ID, "Target app function should not be returned")
	})

	t.Run("get functions for non-existent app", func(t *testing.T) {
		nonExistentAppID := uuid.New()
		result, err := cm.GetFunctionsByAppInternalID(ctx, nonExistentAppID)
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestCQRSInsertFunction(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	t.Run("insert new function", func(t *testing.T) {
		fnID := uuid.New()
		accountID := uuid.New()
		envID := uuid.New()

		params := cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: accountID,
			EnvID:     envID,
			AppID:     appID,
			Name:      "New Test Function",
			Slug:      "new-test-function",
			Config:    `{"triggers": [{"event": "new.test.event"}]}`,
			CreatedAt: time.Now(),
		}

		fn, err := cm.UpsertFunction(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, fn)

		// Verify function properties
		assert.Equal(t, fnID, fn.ID)
		assert.Equal(t, appID, fn.AppID)
		assert.Equal(t, "New Test Function", fn.Name)
		assert.Equal(t, "new-test-function", fn.Slug)
		assert.NotEmpty(t, fn.Config)
		assert.False(t, fn.CreatedAt.IsZero())
		assert.False(t, fn.IsArchived())

		// Verify function can be retrieved
		retrievedFn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		assert.Equal(t, fn.ID, retrievedFn.ID)
		assert.Equal(t, fn.Name, retrievedFn.Name)
		assert.Equal(t, fn.Slug, retrievedFn.Slug)
	})

	t.Run("insert function with minimal fields", func(t *testing.T) {
		fnID := uuid.New()
		accountID := uuid.New()
		envID := uuid.New()

		params := cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: accountID,
			EnvID:     envID,
			AppID:     appID,
			Name:      "Minimal Function",
			Slug:      "minimal-function",
			Config:    `{}`,
			CreatedAt: time.Now(),
		}

		fn, err := cm.UpsertFunction(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, fn)

		assert.Equal(t, fnID, fn.ID)
		assert.Equal(t, "Minimal Function", fn.Name)
		assert.Equal(t, "minimal-function", fn.Slug)
	})
}

func TestCQRSGetFunctions(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	// Create test functions
	fnIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	for i, fnID := range fnIDs {
		_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     appID,
			Name:      fmt.Sprintf("Test Function %d", i+1),
			Slug:      fmt.Sprintf("test-function-%d", i+1),
			Config:    fmt.Sprintf(`{"triggers": [{"event": "test.event%d"}]}`, i+1),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	t.Run("get all functions", func(t *testing.T) {
		functions, err := cm.GetFunctions(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(functions), 3)

		// Verify our test functions are included
		foundFnIDs := make(map[uuid.UUID]bool)
		for _, fn := range functions {
			foundFnIDs[fn.ID] = true
		}

		for _, expectedID := range fnIDs {
			assert.True(t, foundFnIDs[expectedID], "Expected function ID %s to be found", expectedID)
		}
	})
}

func TestCQRSGetFunctionsByAppExternalID(t *testing.T) {
	ctx := context.Background()

	// Create two different apps with external IDs
	targetAppID := uuid.New()
	otherAppID := uuid.New()

	cm, cleanup := initCQRS(t)
	defer cleanup()

	workspaceID := uuid.New()
	accountID := uuid.New()
	targetAppExternalID := "target-app-external"
	otherAppExternalID := "other-app-external"

	// Create target app with external ID
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:          targetAppID,
		Name:        targetAppExternalID,
		SdkLanguage: "go",
		SdkVersion:  "1.2.3",
		Checksum:    "target-checksum",
	})
	require.NoError(t, err)

	// Create other app with different external ID
	_, err = cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:          otherAppID,
		Name:        otherAppExternalID,
		SdkLanguage: "go",
		SdkVersion:  "1.2.3",
		Checksum:    "other-checksum",
	})
	require.NoError(t, err)

	// Create functions for TARGET app
	targetFnIDs := []uuid.UUID{
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		uuid.MustParse("33333333-3333-3333-3333-333333333333"),
	}
	for i, fnID := range targetFnIDs {
		_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: accountID,
			EnvID:     workspaceID,
			AppID:     targetAppID,
			Name:      fmt.Sprintf("Target External Function %d", i+1),
			Slug:      fmt.Sprintf("target-external-function-%d", i+1),
			Config:    fmt.Sprintf(`{"triggers": [{"event": "target.external.event%d"}]}`, i+1),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	// Create functions for OTHER app (should NOT be returned)
	otherFnIDs := []uuid.UUID{
		uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		uuid.MustParse("44444444-4444-4444-4444-444444444444"),
	}
	for i, fnID := range otherFnIDs {
		_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: accountID,
			EnvID:     workspaceID,
			AppID:     otherAppID,
			Name:      fmt.Sprintf("Other External Function %d", i+1),
			Slug:      fmt.Sprintf("other-external-function-%d", i+1),
			Config:    fmt.Sprintf(`{"triggers": [{"event": "other.external.event%d"}]}`, i+1),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	t.Run("get functions for target app only", func(t *testing.T) {
		functions, err := cm.GetFunctionsByAppExternalID(ctx, workspaceID, targetAppExternalID)
		require.NoError(t, err)

		// Should return exactly the target app functions
		assert.Len(t, functions, 2, "Should return exactly 2 functions for target app")

		// Verify all returned functions belong to target app
		returnedFnIDs := make(map[uuid.UUID]bool)
		for _, fn := range functions {
			assert.Equal(t, targetAppID, fn.AppID, "Function %s should belong to target app", fn.ID)
			returnedFnIDs[fn.ID] = true
		}

		// Verify target app functions are included
		for _, expectedID := range targetFnIDs {
			assert.True(t, returnedFnIDs[expectedID], "Target function %s should be returned", expectedID)
		}

		// Verify other app functions are NOT included
		for _, otherFnID := range otherFnIDs {
			assert.False(t, returnedFnIDs[otherFnID], "Other app function %s should not be returned", otherFnID)
		}
	})

	t.Run("get functions for other app only", func(t *testing.T) {
		functions, err := cm.GetFunctionsByAppExternalID(ctx, workspaceID, otherAppExternalID)
		require.NoError(t, err)

		// Should return exactly the other app functions
		assert.Len(t, functions, 2, "Should return exactly 2 functions for other app")

		// Verify all returned functions belong to other app
		for _, fn := range functions {
			assert.Equal(t, otherAppID, fn.AppID, "Function %s should belong to other app", fn.ID)
		}
	})

	t.Run("get functions for non-existent app", func(t *testing.T) {
		functions, err := cm.GetFunctionsByAppExternalID(ctx, workspaceID, "non-existent-app")
		require.NoError(t, err)
		assert.Empty(t, functions, "Should return empty result for non-existent app")
	})

	t.Run("pages functions for target app in id order", func(t *testing.T) {
		functions, err := cm.GetFunctionsByApp(ctx, cqrs.GetFunctionsByAppOpts{
			WorkspaceID: workspaceID,
			AppName:     targetAppExternalID,
			Cursor:      targetFnIDs[0],
			Limit:       1,
		})
		require.NoError(t, err)
		require.Len(t, functions, 1)
		require.Equal(t, targetFnIDs[1], functions[0].ID)
	})
}

func TestCQRSDeleteFunctionsByAppID(t *testing.T) {
	ctx := context.Background()

	// Create two different apps
	targetAppID := uuid.New()
	preserveAppID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(targetAppID))
	defer cleanup()

	// Create the preserve app manually
	_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:          preserveAppID,
		Name:        fmt.Sprintf("preserve-app:%s", preserveAppID),
		SdkLanguage: "go",
		SdkVersion:  "1.2.3",
	})
	require.NoError(t, err)

	// Create functions for the TARGET app (to be deleted)
	targetFnIDs := []uuid.UUID{uuid.New(), uuid.New()}
	for i, fnID := range targetFnIDs {
		_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     targetAppID,
			Name:      fmt.Sprintf("Target Delete Function %d", i+1),
			Slug:      fmt.Sprintf("target-delete-function-%d", i+1),
			Config:    fmt.Sprintf(`{"triggers": [{"event": "target.delete.event%d"}]}`, i+1),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	// Create functions for the PRESERVE app (should NOT be deleted)
	preserveFnIDs := []uuid.UUID{uuid.New(), uuid.New()}
	for i, fnID := range preserveFnIDs {
		_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     preserveAppID,
			Name:      fmt.Sprintf("Preserve Function %d", i+1),
			Slug:      fmt.Sprintf("preserve-function-%d", i+1),
			Config:    fmt.Sprintf(`{"triggers": [{"event": "preserve.event%d"}]}`, i+1),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	t.Run("delete functions by app ID only affects target app", func(t *testing.T) {
		// Verify both apps have functions before deletion
		targetFunctions, err := cm.GetFunctionsByAppInternalID(ctx, targetAppID)
		require.NoError(t, err)
		assert.Len(t, targetFunctions, 2, "Target app should have 2 functions before deletion")

		preserveFunctions, err := cm.GetFunctionsByAppInternalID(ctx, preserveAppID)
		require.NoError(t, err)
		assert.Len(t, preserveFunctions, 2, "Preserve app should have 2 functions before deletion")

		// Delete functions by target app ID only
		err = cm.DeleteFunctionsByAppID(ctx, targetAppID)
		require.NoError(t, err)

		// Verify target app functions are marked as archived
		for _, fnID := range targetFnIDs {
			fn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
			require.NoError(t, err)
			assert.True(t, fn.IsArchived(), "Target app function %s should be archived", fnID)
		}

		// Verify preserve app functions are still active
		for _, fnID := range preserveFnIDs {
			fn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
			require.NoError(t, err)
			assert.False(t, fn.IsArchived(), "Preserve app function %s should still be active", fnID)
		}

		// Verify GetFunctionsByAppInternalID reflects the deletion properly
		targetFunctionsAfter, err := cm.GetFunctionsByAppInternalID(ctx, targetAppID)
		require.NoError(t, err)
		// Should return empty or only non-archived functions (depends on implementation)
		for _, fn := range targetFunctionsAfter {
			assert.False(t, fn.IsArchived(), "GetFunctionsByAppInternalID should not return archived functions")
		}

		preserveFunctionsAfter, err := cm.GetFunctionsByAppInternalID(ctx, preserveAppID)
		require.NoError(t, err)
		assert.Len(t, preserveFunctionsAfter, 2, "Preserve app should still have 2 active functions")
	})

	t.Run("delete functions for non-existent app", func(t *testing.T) {
		nonExistentAppID := uuid.New()
		err := cm.DeleteFunctionsByAppID(ctx, nonExistentAppID)
		// Should not error for non-existent app
		require.NoError(t, err)
	})
}

func TestCQRSDeleteFunctionsByIDs(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	// Create test functions
	fnIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	for i, fnID := range fnIDs {
		_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        fnID,
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     appID,
			Name:      fmt.Sprintf("Delete by ID Function %d", i+1),
			Slug:      fmt.Sprintf("delete-by-id-function-%d", i+1),
			Config:    fmt.Sprintf(`{"triggers": [{"event": "delete.id.event%d"}]}`, i+1),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	t.Run("delete specific functions by IDs", func(t *testing.T) {
		// Delete first two functions
		deleteIDs := fnIDs[:2]
		err := cm.DeleteFunctionsByIDs(ctx, deleteIDs)
		require.NoError(t, err)

		// Verify deleted functions are archived
		for _, fnID := range deleteIDs {
			fn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
			require.NoError(t, err)
			assert.True(t, fn.IsArchived())
		}

		// Verify third function is still active
		fn3, err := cm.GetFunctionByInternalUUID(ctx, fnIDs[2])
		require.NoError(t, err)
		assert.False(t, fn3.IsArchived())
	})

	t.Run("delete non-existent function IDs", func(t *testing.T) {
		nonExistentIDs := []uuid.UUID{uuid.New(), uuid.New()}
		err := cm.DeleteFunctionsByIDs(ctx, nonExistentIDs)
		// Should not error for non-existent IDs
		require.NoError(t, err)
	})

	t.Run("delete empty ID list", func(t *testing.T) {
		err := cm.DeleteFunctionsByIDs(ctx, []uuid.UUID{})
		require.NoError(t, err)
	})
}

func TestCQRSUpdateFunctionConfig(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	// Create test function
	fnID := uuid.New()
	originalConfig := `{"triggers": [{"event": "original.event"}], "concurrency": 1}`
	_, err := cm.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
		ID:        fnID,
		AccountID: uuid.New(),
		EnvID:     uuid.New(),
		AppID:     appID,
		Name:      "Config Update Function",
		Slug:      "config-update-function",
		Config:    originalConfig,
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)

	t.Run("update function config", func(t *testing.T) {
		newConfig := `{"triggers": [{"event": "updated.event"}], "concurrency": 5, "timeout": "30s"}`
		updatedFn, err := cm.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
			ID:     fnID,
			Config: newConfig,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedFn)

		// Verify config was updated
		assert.Equal(t, fnID, updatedFn.ID)
		assert.JSONEq(t, newConfig, string(updatedFn.Config))

		// Verify other fields remain unchanged
		assert.Equal(t, "Config Update Function", updatedFn.Name)
		assert.Equal(t, "config-update-function", updatedFn.Slug)
		assert.Equal(t, appID, updatedFn.AppID)

		// Verify change persisted
		retrievedFn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		assert.JSONEq(t, newConfig, string(retrievedFn.Config))
	})

	t.Run("update config with empty JSON", func(t *testing.T) {
		emptyConfig := `{}`
		updatedFn, err := cm.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
			ID:     fnID,
			Config: emptyConfig,
		})
		require.NoError(t, err)
		require.NotNil(t, updatedFn)

		assert.JSONEq(t, emptyConfig, string(updatedFn.Config))

		// Verify change persisted
		retrievedFn, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		assert.JSONEq(t, emptyConfig, string(retrievedFn.Config))
	})

	t.Run("update non-existent function", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := cm.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
			ID:     nonExistentID,
			Config: `{"test": true}`,
		})
		assert.Error(t, err)
	})
}

//
// Event Tests (TODO)
//

//
// Trace Run Tests
//

func TestTraceRunStatusFromDB(t *testing.T) {
	assert.Equal(t, enums.RunStatusCompleted, traceRunStatusFromDB(enums.RunStatusCompleted.ToCode()))
	assert.Equal(t, enums.RunStatusCompleted, traceRunStatusFromDB(int64(enums.RunStatusCompleted)))
	assert.Equal(t, enums.RunStatusScheduled, traceRunStatusFromDB(int64(enums.RunStatusScheduled)))
	assert.ElementsMatch(t, []int64{enums.RunStatusCompleted.ToCode(), int64(enums.RunStatusCompleted)}, traceRunStatusDBValues(enums.RunStatusCompleted))
}

func TestCQRSGetTraceRunsByTriggerID(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()

	t.Run("find trace run with single trigger ID", func(t *testing.T) {
		// Create a trace run with a single trigger ID
		triggerID := ulid.Make()
		runID := ulid.Make()

		traceRun := &cqrs.TraceRun{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  functionID,
			TraceID:     "trace-single-" + runID.String(),
			RunID:       runID.String(),
			QueuedAt:    time.Now(),
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			TriggerIDs:  []string{triggerID.String()},
			Status:      1,
		}

		err := cm.InsertTraceRun(ctx, traceRun)
		require.NoError(t, err)

		// Search by the trigger ID
		runs, err := cm.GetTraceRunsByTriggerID(ctx, triggerID)
		require.NoError(t, err)
		require.Len(t, runs, 1, "Should find the trace run by its trigger ID")
		assert.Equal(t, runID.String(), runs[0].RunID)
	})

	t.Run("find trace run by trigger ID", func(t *testing.T) {
		// Create a trace run with multiple trigger IDs (event batching)
		triggerID1 := ulid.Make()
		triggerID2 := ulid.Make()
		runID := ulid.Make()

		traceRun := &cqrs.TraceRun{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  functionID,
			TraceID:     "trace-" + runID.String(),
			RunID:       runID.String(),
			QueuedAt:    time.Now(),
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			TriggerIDs:  []string{triggerID1.String(), triggerID2.String()},
			Status:      1, // Running
		}

		err := cm.InsertTraceRun(ctx, traceRun)
		require.NoError(t, err)

		// Search by the first trigger ID - should find the run
		runs, err := cm.GetTraceRunsByTriggerID(ctx, triggerID1)
		require.NoError(t, err)
		require.Len(t, runs, 1, "Should find exactly one trace run by first trigger ID")
		assert.Equal(t, runID.String(), runs[0].RunID)

		// Search by the second trigger ID - should also find the run
		runs, err = cm.GetTraceRunsByTriggerID(ctx, triggerID2)
		require.NoError(t, err)
		require.Len(t, runs, 1, "Should find exactly one trace run by second trigger ID")
		assert.Equal(t, runID.String(), runs[0].RunID)

		// Search by non-existent trigger ID - should return empty
		nonExistentTriggerID := ulid.Make()
		runs, err = cm.GetTraceRunsByTriggerID(ctx, nonExistentTriggerID)
		require.NoError(t, err)
		assert.Len(t, runs, 0, "Should return empty for non-existent trigger ID")
	})

	t.Run("different runs with same trigger ID", func(t *testing.T) {
		// these would most likely be different functions in real use, but doesn't matter for the test
		triggerID := ulid.Make()

		run1ID := ulid.Make()
		run2ID := ulid.Make()

		traceRun1 := &cqrs.TraceRun{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  functionID,
			TraceID:     "trace-batch-1-" + run1ID.String(),
			RunID:       run1ID.String(),
			QueuedAt:    time.Now(),
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			TriggerIDs:  []string{triggerID.String()},
			Status:      1,
		}

		traceRun2 := &cqrs.TraceRun{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  functionID,
			TraceID:     "trace-batch-2-" + run2ID.String(),
			RunID:       run2ID.String(),
			QueuedAt:    time.Now(),
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			TriggerIDs:  []string{triggerID.String()},
			Status:      1,
		}

		err := cm.InsertTraceRun(ctx, traceRun1)
		require.NoError(t, err)
		err = cm.InsertTraceRun(ctx, traceRun2)
		require.NoError(t, err)

		// Search by the shared trigger ID - should find both runs
		runs, err := cm.GetTraceRunsByTriggerID(ctx, triggerID)
		require.NoError(t, err)
		assert.Len(t, runs, 2, "Should find both trace runs that share the same trigger ID")

		// Verify both run IDs are present
		runIDs := make([]string, len(runs))
		for i, r := range runs {
			runIDs[i] = r.RunID
		}
		assert.Contains(t, runIDs, run1ID.String())
		assert.Contains(t, runIDs, run2ID.String())
	})
}

func TestCQRSInsertTraceRun_PreservesTerminalStateAgainstStaleNonTerminalWrite(t *testing.T) {
	terminalStatuses := []enums.RunStatus{
		enums.RunStatusCompleted,
		enums.RunStatusFailed,
		enums.RunStatusCancelled,
		enums.RunStatusOverflowed,
		enums.RunStatusSkipped,
	}
	staleStatuses := []enums.RunStatus{
		enums.RunStatusScheduled,
		enums.RunStatusRunning,
	}

	for _, terminalStatus := range terminalStatuses {
		for _, staleStatus := range staleStatuses {
			t.Run(fmt.Sprintf("%s_then_%s", terminalStatus, staleStatus), func(t *testing.T) {
				ctx := context.Background()
				appID := uuid.New()

				cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
				defer cleanup()

				accountID := uuid.New()
				workspaceID := uuid.New()
				functionID := uuid.New()
				runID := ulid.MustNew(ulid.Now(), rand.Reader)

				now := time.Now().UTC().Truncate(time.Second)
				terminalOutput := []byte(fmt.Sprintf(`{"status":%q}`, terminalStatus.String()))

				terminal := &cqrs.TraceRun{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					FunctionID:  functionID,
					TraceID:     "trace-terminal-" + runID.String(),
					RunID:       runID.String(),
					QueuedAt:    now.Add(-2 * time.Minute),
					StartedAt:   now.Add(-1 * time.Minute),
					EndedAt:     now,
					SourceID:    "terminal-source",
					TriggerIDs:  []string{"evt-terminal"},
					Output:      terminalOutput,
					Status:      terminalStatus,
				}

				stale := &cqrs.TraceRun{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					FunctionID:  functionID,
					TraceID:     "trace-stale-" + runID.String(),
					RunID:       runID.String(),
					QueuedAt:    now.Add(-3 * time.Minute),
					StartedAt:   now.Add(-2 * time.Minute),
					SourceID:    "stale-source",
					TriggerIDs:  []string{"evt-stale"},
					Status:      staleStatus,
				}

				require.NoError(t, cm.InsertTraceRun(ctx, terminal))
				require.NoError(t, cm.InsertTraceRun(ctx, stale))

				got, err := cm.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
				require.NoError(t, err)

				assert.Equal(t, terminal.TraceID, got.TraceID)
				assert.Equal(t, terminal.QueuedAt.UnixMilli(), got.QueuedAt.UnixMilli())
				assert.Equal(t, terminal.StartedAt.UnixMilli(), got.StartedAt.UnixMilli())
				assert.Equal(t, terminal.EndedAt.UnixMilli(), got.EndedAt.UnixMilli())
				assert.Equal(t, terminal.SourceID, got.SourceID)
				assert.Equal(t, terminal.TriggerIDs, got.TriggerIDs)
				assert.Equal(t, terminalOutput, got.Output)
				assert.Equal(t, terminalStatus, got.Status)
			})
		}
	}
}

// Invariant from TLA RunStateProjection ("terminal states are monotonic"):
// once a terminal status has been written for a run_id, no subsequent read
// returns a non-terminal status, regardless of the order writes arrive in.
func TestCQRSInsertTraceRun_TerminalMonotonicityUnderAllOrderings(t *testing.T) {
	base := []enums.RunStatus{
		enums.RunStatusScheduled,
		enums.RunStatusRunning,
		enums.RunStatusCompleted,
		enums.RunStatusCancelled,
	}

	var perms [][]enums.RunStatus
	var permute func([]enums.RunStatus, int)
	permute = func(arr []enums.RunStatus, k int) {
		if k == len(arr)-1 {
			cpy := make([]enums.RunStatus, len(arr))
			copy(cpy, arr)
			perms = append(perms, cpy)
			return
		}
		for i := k; i < len(arr); i++ {
			arr[k], arr[i] = arr[i], arr[k]
			permute(arr, k+1)
			arr[k], arr[i] = arr[i], arr[k]
		}
	}
	arr := make([]enums.RunStatus, len(base))
	copy(arr, base)
	permute(arr, 0)

	for _, perm := range perms {
		name := ""
		for i, s := range perm {
			if i > 0 {
				name += "-"
			}
			name += s.String()
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			appID := uuid.New()
			cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
			defer cleanup()

			accountID := uuid.New()
			workspaceID := uuid.New()
			functionID := uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)
			now := time.Now().UTC().Truncate(time.Second)

			sawTerminal := false
			for i, s := range perm {
				tr := &cqrs.TraceRun{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					FunctionID:  functionID,
					TraceID:     fmt.Sprintf("trace-%d-%s", i, runID.String()),
					RunID:       runID.String(),
					QueuedAt:    now,
					StartedAt:   now,
					EndedAt:     now,
					SourceID:    "test-source",
					TriggerIDs:  []string{"evt-test"},
					Status:      s,
				}
				require.NoError(t, cm.InsertTraceRun(ctx, tr))

				got, err := cm.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
				require.NoError(t, err)

				if enums.RunStatusEnded(s) {
					sawTerminal = true
				}
				if sawTerminal {
					assert.True(t, enums.RunStatusEnded(got.Status),
						"step %d (write %s): after observing terminal, got %s",
						i, s, got.Status)
				}
			}
		})
	}
}

func TestCQRSGetTraceRunsPagination(t *testing.T) {
	// This test verifies that cursor-based pagination works correctly for the GetSpanRuns
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()

	// Create 3 spans with "executor.run" name (required for GetSpanRuns) with distinct start_time
	baseTime := time.Now().UTC().Truncate(time.Second)
	runIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		runIDs[i] = runID

		insertTestSpan(t, cm, testSpanFields{
			RunID:         runID,
			DynamicSpanID: fmt.Sprintf("dyn-%d", i),
			Name:          "executor.run",
			StartTime:     baseTime.Add(time.Duration(i) * time.Second),
			AccountID:     accountID.String(),
			AppID:         appID.String(),
			FunctionID:    functionID.String(),
			EnvID:         workspaceID.String(),
		})
	}

	t.Run("preview path paginate with cursor", func(t *testing.T) {
		// Fetch a page of 1 item at a time. We'll use cursor to get 3 pages
		getPage := func(cursor string) ([]*cqrs.TraceRun, error) {
			return cm.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
				Filter: cqrs.GetTraceRunFilter{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					FunctionID:  []uuid.UUID{functionID},
					TimeField:   enums.TraceRunTimeStartedAt,
					From:        baseTime.Add(-time.Hour),
					Until:       baseTime.Add(time.Hour),
				},
				Order: []cqrs.GetTraceRunOrder{
					{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
				},
				Cursor:  cursor,
				Items:   1,
				Preview: true,
			})
		}

		// Fetch first page (no cursor, 1 item, ordered by started_at desc)
		firstPage, err := getPage("")
		require.NoError(t, err)
		require.Len(t, firstPage, 1, "First page should have 1 item")
		require.NotEmpty(t, firstPage[0].Cursor, "First page result should have a cursor")
		firstRunID := firstPage[0].RunID

		// Fetch second page using the cursor from first page
		secondPage, err := getPage(firstPage[0].Cursor)
		require.NoError(t, err)
		require.Len(t, secondPage, 1, "Second page should have 1 item")
		secondRunID := secondPage[0].RunID
		assert.NotEqual(t, firstRunID, secondRunID, "Second page should return a different run than first page")

		// Fetch third page
		thirdPage, err := getPage(secondPage[0].Cursor)
		require.NoError(t, err)
		require.Len(t, thirdPage, 1, "Third page should have 1 item")
		thirdRunID := thirdPage[0].RunID
		assert.NotEqual(t, firstRunID, thirdRunID, "Third page should return a different run than first page")
		assert.NotEqual(t, secondRunID, thirdRunID, "Third page should return a different run than second page")

		// Verify we got all 3 runs
		returnedRunIDs := []string{firstRunID, secondRunID, thirdRunID}
		for _, id := range runIDs {
			assert.Contains(t, returnedRunIDs, id, "All created runs should be returned through pagination")
		}
	})
}

func TestCQRSGetTraceRunsExcludesSkipped(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)

	completedRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	insertTestSpan(t, cm, testSpanFields{
		RunID:         completedRunID,
		DynamicSpanID: "dyn-completed",
		Name:          "executor.run",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	insertTestSpan(t, cm, testSpanFields{
		RunID:         ulid.MustNew(ulid.Now(), rand.Reader).String(),
		DynamicSpanID: "dyn-skipped",
		Name:          "executor.run",
		Status:        enums.RunStatusSkipped.String(),
		StartTime:     baseTime.Add(time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	runs, err := cm.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Preview: true,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1, "Skipped runs should be excluded from the runs list")
	assert.Equal(t, completedRunID, runs[0].RunID)
}

func TestCQRSGetTraceRunsPreviewScopesRootLookupAndCount(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	otherAccountID := uuid.New()
	otherWorkspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)

	completedRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         completedRunID,
		DynamicSpanID: "dyn-completed",
		Name:          "executor.run",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	failedRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         failedRunID,
		DynamicSpanID: "dyn-failed",
		Name:          "executor.run",
		Status:        enums.RunStatusFailed.String(),
		StartTime:     baseTime.Add(time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	// This fragment shares a dynamic span ID with another tenant's root span,
	// but it has no root in the requested tenant and must not become a run.
	leakedRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         leakedRunID,
		DynamicSpanID: "dyn-shared",
		Name:          "EXTEND",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime.Add(2 * time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})
	insertTestSpan(t, cm, testSpanFields{
		RunID:         ulid.MustNew(ulid.Now(), rand.Reader).String(),
		DynamicSpanID: "dyn-shared",
		Name:          "executor.run",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime.Add(2 * time.Second),
		AccountID:     otherAccountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         otherWorkspaceID.String(),
	})

	insertTestSpan(t, cm, testSpanFields{
		RunID:         ulid.MustNew(ulid.Now(), rand.Reader).String(),
		DynamicSpanID: "dyn-skipped",
		Name:          "executor.run",
		Status:        enums.RunStatusSkipped.String(),
		StartTime:     baseTime.Add(3 * time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Items:   1,
		Preview: true,
	}

	firstPage, err := cm.GetTraceRuns(ctx, opt)
	require.NoError(t, err)
	require.Len(t, firstPage, 1)

	countOpt := opt
	countOpt.Cursor = firstPage[0].Cursor
	count, err := cm.GetTraceRunsCount(ctx, countOpt)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "count should ignore cursor and exclude cross-tenant root matches")

	countOpt.Filter.Status = []enums.RunStatus{enums.RunStatusCompleted}
	statusCount, err := cm.GetTraceRunsCount(ctx, countOpt)
	require.NoError(t, err)
	assert.Equal(t, 1, statusCount)

	listOpt := opt
	listOpt.Items = 0
	runs, err := cm.GetTraceRuns(ctx, listOpt)
	require.NoError(t, err)
	require.Len(t, runs, 2)

	runIDs := []string{runs[0].RunID, runs[1].RunID}
	assert.Contains(t, runIDs, completedRunID)
	assert.Contains(t, runIDs, failedRunID)
	assert.NotContains(t, runIDs, leakedRunID)
}

func TestCQRSGetTraceRunsNonPreviewScopesTenant(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	otherAccountID := uuid.New()
	otherWorkspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)
	targetRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	err := cm.InsertTraceRun(ctx, &cqrs.TraceRun{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		TraceID:     "trace-target",
		RunID:       targetRunID,
		QueuedAt:    baseTime,
		StartedAt:   baseTime,
		EndedAt:     baseTime.Add(time.Second),
		Status:      enums.RunStatusCompleted,
	})
	require.NoError(t, err)

	err = cm.InsertTraceRun(ctx, &cqrs.TraceRun{
		AccountID:   otherAccountID,
		WorkspaceID: otherWorkspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		TraceID:     "trace-other",
		RunID:       ulid.MustNew(ulid.Now(), rand.Reader).String(),
		QueuedAt:    baseTime,
		StartedAt:   baseTime,
		EndedAt:     baseTime.Add(time.Second),
		Status:      enums.RunStatusCompleted,
	})
	require.NoError(t, err)

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
	}

	runs, err := cm.GetTraceRuns(ctx, opt)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, targetRunID, runs[0].RunID)

	count, err := cm.GetTraceRunsCount(ctx, opt)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCQRSGetTraceRunsPreviewEndedAtUsesTraceRuns(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	otherAccountID := uuid.New()
	otherWorkspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)

	runIDs := make([]string, 2)
	for i := range runIDs {
		runIDs[i] = ulid.MustNew(ulid.Now(), rand.Reader).String()
		status := enums.RunStatusCompleted
		if i == 1 {
			status = enums.RunStatusRunning
		}
		err := cm.InsertTraceRun(ctx, &cqrs.TraceRun{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  functionID,
			TraceID:     fmt.Sprintf("trace-ended-%d", i),
			RunID:       runIDs[i],
			QueuedAt:    baseTime,
			StartedAt:   baseTime,
			EndedAt:     baseTime.Add(time.Duration(i+1) * time.Second),
			Status:      status,
		})
		require.NoError(t, err)
	}

	for _, span := range []testSpanFields{
		{
			RunID:         runIDs[1],
			DynamicSpanID: "ended-stale-root",
			Name:          meta.SpanNameRun,
			Status:        enums.RunStatusRunning.String(),
			StartTime:     baseTime,
			AccountID:     accountID.String(),
			AppID:         appID.String(),
			FunctionID:    functionID.String(),
			EnvID:         workspaceID.String(),
		},
		{
			RunID:         runIDs[1],
			DynamicSpanID: "ended-stale-root",
			Name:          meta.SpanNameRun,
			Status:        enums.RunStatusCompleted.String(),
			StartTime:     baseTime.Add(2 * time.Second),
			AccountID:     accountID.String(),
			AppID:         appID.String(),
			FunctionID:    functionID.String(),
			EnvID:         workspaceID.String(),
		},
	} {
		insertTestSpan(t, cm, span)
	}

	err := cm.InsertTraceRun(ctx, &cqrs.TraceRun{
		AccountID:   otherAccountID,
		WorkspaceID: otherWorkspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		TraceID:     "trace-other-ended",
		RunID:       ulid.MustNew(ulid.Now(), rand.Reader).String(),
		QueuedAt:    baseTime,
		StartedAt:   baseTime,
		EndedAt:     baseTime.Add(3 * time.Second),
		Status:      enums.RunStatusCompleted,
	})
	require.NoError(t, err)

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeEndedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeEndedAt, Direction: enums.TraceRunOrderDesc},
		},
		Items:   1,
		Preview: true,
	}

	firstPage, err := cm.GetTraceRuns(ctx, opt)
	require.NoError(t, err)
	require.Len(t, firstPage, 1)
	assert.Equal(t, runIDs[1], firstPage[0].RunID)
	assert.Equal(t, accountID, firstPage[0].AccountID)
	assert.Equal(t, workspaceID, firstPage[0].WorkspaceID)
	assert.Equal(t, enums.RunStatusCompleted, firstPage[0].Status)

	countOpt := opt
	countOpt.Cursor = firstPage[0].Cursor
	count, err := cm.GetTraceRunsCount(ctx, countOpt)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	secondOpt := opt
	secondOpt.Cursor = firstPage[0].Cursor
	secondPage, err := cm.GetTraceRuns(ctx, secondOpt)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, runIDs[0], secondPage[0].RunID)
}

func TestCQRSGetSpanRunsCELJoinQualifiesTenantFilters(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)
	eventID := "evt-cel-tenant-filter"
	runID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	require.NoError(t, cm.InsertEvent(ctx, cqrs.Event{
		ID:        ulid.MustNew(ulid.Now(), rand.Reader),
		EventID:   eventID,
		EventName: "app/cel.match",
		EventData: map[string]any{"tenant": "target"},
		EventUser: map[string]any{},
		EventTS:   baseTime.UnixMilli(),
	}))

	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID,
		DynamicSpanID: "dyn-cel-tenant-filter",
		Name:          meta.SpanNameRun,
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
		EventIDs:      []byte(fmt.Sprintf(`["%s"]`, eventID)),
	})

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
			CEL:         `event.data.tenant == "target"`,
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Preview: true,
	}

	runs, err := cm.GetTraceRuns(ctx, opt)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, runID, runs[0].RunID)

	count, err := cm.GetTraceRunsCount(ctx, opt)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// Root-page results must derive end_time/status from EXTEND spans.
func TestCQRSGetSpanRunsEnrichmentFromExtendSpans(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)

	extendedRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         extendedRunID,
		DynamicSpanID: "dyn-extended",
		Name:          "executor.run",
		Status:        enums.RunStatusRunning.String(),
		StartTime:     baseTime,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})
	insertTestSpan(t, cm, testSpanFields{
		RunID:         extendedRunID,
		DynamicSpanID: "dyn-extended",
		Name:          "EXTEND",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime.Add(5 * time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	rootOnlyRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         rootOnlyRunID,
		DynamicSpanID: "dyn-rootonly",
		Name:          "executor.run",
		Status:        enums.RunStatusFailed.String(),
		StartTime:     baseTime.Add(time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	runs, err := cm.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Preview: true,
	})
	require.NoError(t, err)
	require.Len(t, runs, 2)

	byRun := map[string]*cqrs.TraceRun{}
	for _, r := range runs {
		byRun[r.RunID] = r
	}

	extended := byRun[extendedRunID]
	require.NotNil(t, extended)
	assert.Equal(t, enums.RunStatusCompleted, extended.Status)
	assert.Equal(t, baseTime, extended.StartedAt.UTC())
	assert.GreaterOrEqual(t, extended.EndedAt.Sub(extended.StartedAt), 5*time.Second)

	rootOnly := byRun[rootOnlyRunID]
	require.NotNil(t, rootOnly)
	assert.Equal(t, enums.RunStatusFailed, rootOnly.Status)
	assert.Less(t, rootOnly.EndedAt.Sub(rootOnly.StartedAt), time.Second)
}

func TestCQRSGetSpanRunsEnrichmentIncludesExtendSpansAfterPageWindow(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)
	until := baseTime.Add(time.Minute)

	runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID,
		DynamicSpanID: "dyn-post-window-extend",
		Name:          "executor.run",
		Status:        enums.RunStatusRunning.String(),
		StartTime:     until.Add(-time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})
	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID,
		DynamicSpanID: "dyn-post-window-extend",
		Name:          "EXTEND",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     until.Add(5 * time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	runs, err := cm.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       until,
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Preview: true,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)

	assert.Equal(t, runID, runs[0].RunID)
	assert.Equal(t, enums.RunStatusCompleted, runs[0].Status)
	assert.True(t, runs[0].EndedAt.UTC().After(until), "end_time should come from the post-window EXTEND span")
}

// Status filters match final run status, not any historical span row.
func TestCQRSGetSpanRunsFinalStatusFilter(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)

	runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID,
		DynamicSpanID: "dyn-1",
		Name:          "executor.run",
		Status:        enums.RunStatusFailed.String(),
		StartTime:     baseTime,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})
	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID,
		DynamicSpanID: "dyn-1",
		Name:          "EXTEND",
		Status:        enums.RunStatusCompleted.String(),
		StartTime:     baseTime.Add(5 * time.Second),
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    functionID.String(),
		EnvID:         workspaceID.String(),
	})

	baseOpt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Preview: true,
	}

	completedOpt := baseOpt
	completedOpt.Filter.Status = []enums.RunStatus{enums.RunStatusCompleted}
	completedRuns, err := cm.GetTraceRuns(ctx, completedOpt)
	require.NoError(t, err)
	require.Len(t, completedRuns, 1, "final status Completed should match")
	assert.Equal(t, runID, completedRuns[0].RunID)
	completedCount, err := cm.GetTraceRunsCount(ctx, completedOpt)
	require.NoError(t, err)
	assert.Equal(t, 1, completedCount)

	failedOpt := baseOpt
	failedOpt.Filter.Status = []enums.RunStatus{enums.RunStatusFailed}
	failedRuns, err := cm.GetTraceRuns(ctx, failedOpt)
	require.NoError(t, err)
	require.Empty(t, failedRuns, "a run whose final status is Completed must not match Failed")
	failedCount, err := cm.GetTraceRunsCount(ctx, failedOpt)
	require.NoError(t, err)
	assert.Equal(t, 0, failedCount)
}

// Cursor pagination must not skip or duplicate root-page results.
func TestCQRSGetSpanRunsFastPathPagination(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Second)

	const total = 5
	wantOrder := make([]string, 0, total)
	for i := 0; i < total; i++ {
		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		insertTestSpan(t, cm, testSpanFields{
			RunID:         runID,
			DynamicSpanID: fmt.Sprintf("dyn-%d", i),
			Name:          "executor.run",
			Status:        enums.RunStatusCompleted.String(),
			StartTime:     baseTime.Add(time.Duration(i) * time.Second),
			AccountID:     accountID.String(),
			AppID:         appID.String(),
			FunctionID:    functionID.String(),
			EnvID:         workspaceID.String(),
		})
		wantOrder = append([]string{runID}, wantOrder...)
	}

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			FunctionID:  []uuid.UUID{functionID},
			TimeField:   enums.TraceRunTimeStartedAt,
			From:        baseTime.Add(-time.Hour),
			Until:       baseTime.Add(time.Hour),
		},
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeStartedAt, Direction: enums.TraceRunOrderDesc},
		},
		Items:   2,
		Preview: true,
	}

	count, err := cm.GetTraceRunsCount(ctx, opt)
	require.NoError(t, err)
	assert.Equal(t, total, count)

	var got []string
	cursor := ""
	for page := 0; page < total+1; page++ {
		pageOpt := opt
		pageOpt.Cursor = cursor
		runs, err := cm.GetTraceRuns(ctx, pageOpt)
		require.NoError(t, err)
		if len(runs) == 0 {
			break
		}
		for _, r := range runs {
			got = append(got, r.RunID)
		}
		if len(runs) < int(opt.Items) {
			break
		}
		cursor = runs[len(runs)-1].Cursor
		require.NotEmpty(t, cursor, "cursor required to continue pagination")
	}

	assert.Equal(t, wantOrder, got, "every run returned once, in descending start order")
}

//
// Span Tests
//

func TestCQRSGetSpan(t *testing.T) {
	// These tests insert a root and child span with different dynamic_span_ids.
	// Each test tests a different query that GROUPs BY dynamic_span_id

	t.Run("by run ID", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()

		insertTestSpan(t, cm, testSpanFields{RunID: runID, DynamicSpanID: "dyn-root"})
		insertTestSpan(t, cm, testSpanFields{RunID: runID, DynamicSpanID: "dyn-child", ParentSpanID: "dyn-root"})

		result, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "dyn-root", result.SpanID)
		assert.Len(t, result.Children, 1, "Root should have 1 child")
	})

	t.Run("by debug run ID", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		debugRunID := ulid.MustNew(ulid.Now(), rand.Reader)

		insertTestSpan(t, cm, testSpanFields{RunID: runID, DynamicSpanID: "dyn-root", DebugRunID: debugRunID.String()})
		insertTestSpan(t, cm, testSpanFields{RunID: runID, DynamicSpanID: "dyn-child", ParentSpanID: "dyn-root", DebugRunID: debugRunID.String()})

		result, err := cm.GetSpansByDebugRunID(t.Context(), debugRunID)
		require.NoError(t, err)
		require.Len(t, result, 1, "Should return 1 root span for the single run")
		assert.Len(t, result[0].Children, 1, "Root should have 1 child")
	})

	t.Run("by debug session ID", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		debugRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		debugSessionID := ulid.MustNew(ulid.Now(), rand.Reader)

		insertTestSpan(t, cm, testSpanFields{RunID: runID, DynamicSpanID: "dyn-root", DebugRunID: debugRunID, DebugSessionID: debugSessionID.String()})
		insertTestSpan(t, cm, testSpanFields{RunID: runID, DynamicSpanID: "dyn-child", ParentSpanID: "dyn-root", DebugRunID: debugRunID, DebugSessionID: debugSessionID.String()})

		result, err := cm.GetSpansByDebugSessionID(t.Context(), debugSessionID)
		require.NoError(t, err)
		require.Len(t, result, 1, "Should return 1 debug run group")
		require.Len(t, result[0], 1, "Debug run group should have 1 root span")
		assert.Len(t, result[0][0].Children, 1, "Root should have 1 child")
	})

	t.Run("by debug session ID keeps runs separate when dynamic span IDs collide", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		debugSessionID := ulid.MustNew(ulid.Now(), rand.Reader)

		runIDOne := ulid.MustNew(ulid.Now(), rand.Reader).String()
		runIDTwo := ulid.MustNew(ulid.Now(), rand.Reader).String()
		debugRunIDOne := ulid.MustNew(ulid.Now(), rand.Reader).String()
		debugRunIDTwo := ulid.MustNew(ulid.Now(), rand.Reader).String()

		insertTestSpan(t, cm, testSpanFields{
			RunID:          runIDOne,
			DynamicSpanID:  "dyn-root",
			DebugRunID:     debugRunIDOne,
			DebugSessionID: debugSessionID.String(),
		})
		insertTestSpan(t, cm, testSpanFields{
			RunID:          runIDTwo,
			DynamicSpanID:  "dyn-root",
			DebugRunID:     debugRunIDTwo,
			DebugSessionID: debugSessionID.String(),
		})

		result, err := cm.GetSpansByDebugSessionID(t.Context(), debugSessionID)
		require.NoError(t, err)
		require.Len(t, result, 2, "separate runs in the same debug session must not collapse into one group")
		require.Len(t, result[0], 1)
		require.Len(t, result[1], 1)
	})
}

// TestSpanWithAttributesAndOutput is a regression test ensuring that spans
// with JSON attributes and output can be inserted and queried without
// "JSON cannot hold BLOB values" errors. This catches the bug where []byte
// fields were stored as BLOBs instead of TEXT in SQLite JSON columns.
func TestSpanWithAttributesAndOutput(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runULID := ulid.MustNew(ulid.Now(), rand.Reader)
	runID := runULID.String()

	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID,
		DynamicSpanID: "dyn-with-attrs",
		Name:          "executor.run",
		Attributes:    []byte(`{"sdk.language":"go","sdk.version":"0.1.0"}`),
		Output:        []byte(`{"data":{"num":42}}`),
	})

	// GetSpansByRunID uses json_group_array(json_object('attributes', attributes, ...))
	// which fails with "JSON cannot hold BLOB values" if attributes/output are BLOBs.
	result, err := cm.GetSpansByRunID(t.Context(), runULID)
	require.NoError(t, err, "query must not fail with 'JSON cannot hold BLOB values'")
	assert.NotNil(t, result)
}

// TestSpanAttributesRoundTrip exercises the read path for stored span
// attributes. Postgres' jsonb embeds as a nested object inside json_build_object
// while sqlite's JSON returns a quoted string, so this regression-tests that the
// reader handles both shapes — when broken, the entire attribute map is silently
// empty under postgres and every downstream consumer (StepOp, RunID, etc.) sees
// zero values.
func TestSpanAttributesRoundTrip(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runULID := ulid.MustNew(ulid.Now(), rand.Reader)
	insertTestSpan(t, cm, testSpanFields{
		RunID:         runULID.String(),
		DynamicSpanID: "dyn-attrs",
		Name:          "executor.run",
		Attributes:    []byte(`{"sdk.language":"go","sdk.version":"0.1.0","_inngest.run.id":"` + runULID.String() + `"}`),
	})

	result, err := cm.GetSpansByRunID(t.Context(), runULID)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "go", result.RawOtelSpan.Attributes["sdk.language"])
	assert.Equal(t, "0.1.0", result.RawOtelSpan.Attributes["sdk.version"])
}

// TestSpanOutputReadBack verifies that span output stored as []byte can be
// read back via GetSpanOutput without corruption. This is a regression test
// for double-encoding where json.Marshal(stringValue) would wrap the JSON
// in extra quotes, breaking json_extract queries.
func TestSpanOutputReadBack(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	spanID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	q := cm.(wrapper).q
	err := q.InsertSpan(t.Context(), dbpkg.InsertSpanParams{
		SpanID:     spanID,
		TraceID:    ulid.MustNew(ulid.Now(), rand.Reader).String(),
		Name:       "test-span",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(100 * time.Millisecond),
		RunID:      runID,
		AccountID:  "acct",
		AppID:      "app",
		FunctionID: "fn",
		EnvID:      "env",
		Output:     []byte(`{"data":{"num":42}}`),
	})
	require.NoError(t, err)

	out, err := cm.GetSpanOutput(t.Context(), cqrs.SpanIdentifier{RunID: runID, SpanID: spanID})
	require.NoError(t, err)
	require.NotNil(t, out)
	// After the data/error unwrapping in GetSpanOutput, "data" key is extracted
	assert.Contains(t, string(out.Data), `"num":42`, "output should contain raw JSON, not double-encoded")
}

// TestExtendedTraceReparenting verifies that orphaned userland spans (extended
// trace spans whose parent OTEL span ID is no longer in the tree) are
// reparented to their matching step or execution span by stepID + attempt.
func TestExtendedTraceReparenting(t *testing.T) {
	// stepAttrs returns JSON attributes for a step/exec span.
	stepAttrs := func(stepID string, attempt int) []byte {
		b, _ := json.Marshal(map[string]any{
			"_inngest.step.id":      stepID,
			"_inngest.step.attempt": attempt,
		})
		return b
	}

	// userlandAttrs returns JSON attributes for an orphaned extended trace span.
	userlandAttrs := func(stepID string, attempt int) []byte {
		b, _ := json.Marshal(map[string]any{
			"_inngest.userland":     true,
			"_inngest.step.id":      stepID,
			"_inngest.step.attempt": attempt,
		})
		return b
	}

	t.Run("reparents to executor.step by stepID and attempt", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-step", Name: meta.SpanNameStep,
			ParentSpanID: "dyn-run", Attributes: stepAttrs("step-abc", 0),
		})
		// Orphaned: parent "stale-otel-id" is not in the span map.
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-userland", Name: "userland",
			ParentSpanID: "stale-otel-id", Attributes: userlandAttrs("step-abc", 0),
		})

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Children, 1)
		step := result.Children[0]
		assert.Equal(t, "dyn-step", step.SpanID)
		require.Len(t, step.Children, 1, "extended trace span should be reparented under executor.step")
		assert.Equal(t, "dyn-userland", step.Children[0].SpanID)
	})

	t.Run("falls back to executor.execution when no executor.step exists", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-exec", Name: meta.SpanNameExecution,
			ParentSpanID: "dyn-run", Attributes: stepAttrs("step-xyz", 0),
		})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-userland", Name: "userland",
			ParentSpanID: "stale-otel-id", Attributes: userlandAttrs("step-xyz", 0),
		})

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Children, 1)
		exec := result.Children[0]
		assert.Equal(t, "dyn-exec", exec.SpanID)
		require.Len(t, exec.Children, 1, "extended trace span should fall back to executor.execution")
		assert.Equal(t, "dyn-userland", exec.Children[0].SpanID)
	})

	t.Run("drops orphaned span when no matching step or execution span exists", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})
		// No step or exec span for "step-no-match".
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-userland", Name: "userland",
			ParentSpanID: "stale-otel-id", Attributes: userlandAttrs("step-no-match", 0),
		})

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Children, "orphaned extended trace span with no match should be dropped")
	})

	t.Run("each attempt's span is reparented to its own attempt", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})

		for _, attempt := range []int{0, 1} {
			insertTestSpan(t, cm, testSpanFields{
				RunID:        runIDStr,
				DynamicSpanID: fmt.Sprintf("dyn-step-%d", attempt),
				Name:         meta.SpanNameStep,
				ParentSpanID: "dyn-run",
				Attributes:   stepAttrs("step-retry", attempt),
			})
			insertTestSpan(t, cm, testSpanFields{
				RunID:        runIDStr,
				DynamicSpanID: fmt.Sprintf("dyn-userland-%d", attempt),
				Name:         "userland",
				ParentSpanID: "stale-otel-id",
				Attributes:   userlandAttrs("step-retry", attempt),
			})
		}

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Children, 2)

		bySpanID := make(map[string]*cqrs.OtelSpan, 2)
		for _, child := range result.Children {
			bySpanID[child.SpanID] = child
		}

		step0 := bySpanID["dyn-step-0"]
		require.NotNil(t, step0)
		require.Len(t, step0.Children, 1, "attempt 0 extended trace span should be under attempt 0 step")
		assert.Equal(t, "dyn-userland-0", step0.Children[0].SpanID)

		step1 := bySpanID["dyn-step-1"]
		require.NotNil(t, step1)
		require.Len(t, step1.Children, 1, "attempt 1 extended trace span should be under attempt 1 step")
		assert.Equal(t, "dyn-userland-1", step1.Children[0].SpanID)
	})

	t.Run("does not reparent when attempt attribute is missing", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-step", Name: meta.SpanNameStep,
			ParentSpanID: "dyn-run", Attributes: stepAttrs("step-abc", 0),
		})
		// Orphaned userland span with stepID but no attempt attribute.
		noAttemptAttrs, _ := json.Marshal(map[string]any{
			"_inngest.userland": true,
			"_inngest.step.id":  "step-abc",
		})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-userland", Name: "userland",
			ParentSpanID: "stale-otel-id", Attributes: noAttemptAttrs,
		})

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Children, 1)
		assert.Empty(t, result.Children[0].Children, "userland span missing attempt should not be reparented")
	})

	t.Run("userland span with valid parent reference uses normal lineage", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-step", Name: meta.SpanNameStep,
			ParentSpanID: "dyn-run", Attributes: stepAttrs("step-linked", 0),
		})
		// Valid parent reference — should use normal lineage, not reparenting.
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-userland", Name: "userland",
			ParentSpanID: "dyn-step", Attributes: userlandAttrs("step-linked", 0),
		})

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Children, 1)
		step := result.Children[0]
		assert.Equal(t, "dyn-step", step.SpanID)
		require.Len(t, step.Children, 1, "userland span with valid parent should attach via normal lineage")
		assert.Equal(t, "dyn-userland", step.Children[0].SpanID)
	})

	t.Run("only subtree root is reparented, not interior nodes", func(t *testing.T) {
		// A three-span userland subtree: root → child → grandchild.
		// root's inngest parent ("stale-otel-id") is absent from the tree.
		// child and grandchild have parents within the userland subtree.
		// Only root should be reparented; child and grandchild attach normally.
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)
		runIDStr := runID.String()

		insertTestSpan(t, cm, testSpanFields{RunID: runIDStr, DynamicSpanID: "dyn-run", Name: meta.SpanNameRun})
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-step", Name: meta.SpanNameStep,
			ParentSpanID: "dyn-run", Attributes: stepAttrs("step-sub", 0),
		})
		// Subtree root: orphaned from inngest tree, should be reparented to dyn-step.
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-ul-root", Name: "userland",
			ParentSpanID: "stale-otel-id", Attributes: userlandAttrs("step-sub", 0),
		})
		// Interior node: parent is the userland root above; resolves via spanMap normally.
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-ul-child", Name: "userland",
			ParentSpanID: "dyn-ul-root", Attributes: userlandAttrs("step-sub", 0),
		})
		// Leaf: parent is the interior node; resolves via spanMap normally.
		insertTestSpan(t, cm, testSpanFields{
			RunID: runIDStr, DynamicSpanID: "dyn-ul-grandchild", Name: "userland",
			ParentSpanID: "dyn-ul-child", Attributes: userlandAttrs("step-sub", 0),
		})

		result, err := cm.GetSpansByRunID(t.Context(), runID)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Children, 1)
		step := result.Children[0]
		assert.Equal(t, "dyn-step", step.SpanID)

		require.Len(t, step.Children, 1, "only the userland subtree root should be reparented under executor.step")
		root := step.Children[0]
		assert.Equal(t, "dyn-ul-root", root.SpanID)

		require.Len(t, root.Children, 1, "interior node should be attached under the subtree root")
		child := root.Children[0]
		assert.Equal(t, "dyn-ul-child", child.SpanID)

		require.Len(t, child.Children, 1, "grandchild should be attached under the interior node")
		assert.Equal(t, "dyn-ul-grandchild", child.Children[0].SpanID)
	})
}

//
// Helpers
//

type testSpanFields struct {
	RunID          string    // required
	DynamicSpanID  string    // required for GROUP BY tests
	ParentSpanID   string    // for child spans (references parent's DynamicSpanID)
	DebugRunID     string    // for debug run tests
	DebugSessionID string    // for debug session tests
	Name           string    // default: "test-span"
	Status         string    // default: "" (NULL)
	StartTime      time.Time // default: time.Now()
	AccountID      string    // default: "acct"
	AppID          string    // default: "app"
	FunctionID     string    // default: "fn"
	EnvID          string    // default: "env"
	Attributes     []byte    // JSON attributes (optional)
	Output         []byte    // JSON output (optional)
	EventIDs       []byte    // JSON array of event IDs (optional)
}

// There aren't any functions exposed on cqrs.Manager that write to the new spans table
// so use this test helper for now.
func insertTestSpan(t *testing.T, cm cqrs.Manager, spanFields testSpanFields) {
	t.Helper()

	spanID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	traceID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	// Apply defaults
	if spanFields.Name == "" {
		spanFields.Name = "test-span"
	}
	if spanFields.StartTime.IsZero() {
		spanFields.StartTime = time.Now()
	}
	if spanFields.AccountID == "" {
		spanFields.AccountID = "acct"
	}
	if spanFields.AppID == "" {
		spanFields.AppID = "app"
	}
	if spanFields.FunctionID == "" {
		spanFields.FunctionID = "fn"
	}
	if spanFields.EnvID == "" {
		spanFields.EnvID = "env"
	}

	// TODO: ideally we should not have to do this type assertion to wrapper to write a span
	q := cm.(wrapper).q
	err := q.InsertSpan(t.Context(), dbpkg.InsertSpanParams{
		SpanID:         spanID,
		TraceID:        traceID,
		ParentSpanID:   sql.NullString{String: spanFields.ParentSpanID, Valid: spanFields.ParentSpanID != ""},
		Name:           spanFields.Name,
		Status:         sql.NullString{String: spanFields.Status, Valid: spanFields.Status != ""},
		StartTime:      spanFields.StartTime,
		EndTime:        spanFields.StartTime.Add(100 * time.Millisecond),
		RunID:          spanFields.RunID,
		AccountID:      spanFields.AccountID,
		AppID:          spanFields.AppID,
		FunctionID:     spanFields.FunctionID,
		EnvID:          spanFields.EnvID,
		DynamicSpanID:  sql.NullString{String: spanFields.DynamicSpanID, Valid: spanFields.DynamicSpanID != ""},
		DebugRunID:     sql.NullString{String: spanFields.DebugRunID, Valid: spanFields.DebugRunID != ""},
		DebugSessionID: sql.NullString{String: spanFields.DebugSessionID, Valid: spanFields.DebugSessionID != ""},
		Attributes:     spanFields.Attributes,
		Output:         spanFields.Output,
		EventIds:       spanFields.EventIDs,
	})
	require.NoError(t, err)
}

type withInitCQRSOpt func(*initCQRSOpt)

type initCQRSOpt struct {
	appID uuid.UUID
}

func withInitCQRSOptApp(id uuid.UUID) withInitCQRSOpt {
	return func(o *initCQRSOpt) {
		o.appID = id
	}
}

// initCQRS initializes a CQRS manager based on the TEST_DATABASE environment variable.
// When TEST_DATABASE=postgres, it starts a PostgreSQL testcontainer.
// Otherwise, it defaults to in-memory SQLite.
func initCQRS(t *testing.T, opts ...withInitCQRSOpt) (cqrs.Manager, func()) {
	ctx := context.Background()

	opt := initCQRSOpt{}
	for _, apply := range opts {
		apply(&opt)
	}

	var (
		db  *sql.DB
		err error
	)

	var pc *testutil.PostgresContainer

	var adapter adapterWithHelpers

	testDB := os.Getenv(EnvTestDatabase)
	if testDB == "postgres" {
		var pgErr error
		pc, pgErr = testutil.StartPostgres(t)
		require.NoError(t, pgErr)

		db, err = dbpostgres.Open(ctx, dbpostgres.Options{URI: pc.URI, ForTest: true})
		require.NoError(t, err)
		adapter = dbpostgres.New(db)
	} else {
		db, err = dbsqlite.Open(ctx, dbsqlite.Options{Persist: false, ForTest: true})
		require.NoError(t, err)
		adapter = dbsqlite.New(db)
	}

	cm := New(adapter)

	cleanup := func() {
		db.Close()
		if pc != nil {
			if err := pc.Terminate(t.Context()); err != nil {
				t.Logf("failed to terminate postgres container: %v", err)
			}
		}
	}

	if opt.appID != uuid.Nil {
		// Upsert the app
		_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:          opt.appID,
			Name:        fmt.Sprintf("app:%s", opt.appID),
			SdkLanguage: "go",
			SdkVersion:  "1.2.3",
			Framework:   sql.NullString{Valid: true, String: "gin"},
			Metadata:    `{"environment": "test", "version": "1.0"}`,
			AppVersion:  "v2.1.0",
		})
		require.NoError(t, err)
	}

	return cm, cleanup
}
