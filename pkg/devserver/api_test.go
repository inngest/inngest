package devserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestRegister_FunctionVersionIncrement(t *testing.T) {
	ctx := context.Background()

	sdkFunction1 := sdk.SDKFunction{
		Name: "Test Function 1",
		Slug: "test-function-1",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event: "test/event",
				},
			},
		},
		Steps: map[string]sdk.SDKStep{
			"step-1": {
				ID:   "step-1",
				Name: "test step",
				Runtime: map[string]any{
					"url": "http://localhost:3000/api/inngest",
				},
			},
		},
	}

	sdkFunction2 := sdkFunction1
	sdkFunction2.Name = "Test Function 2"
	sdkFunction2.Slug = "test-function-2"

	// request initially only has one function
	req := sdk.RegisterRequest{
		URL:     "http://localhost:3000/api/inngest",
		AppName: "test-app",
		V:       "1",
		Functions: []sdk.SDKFunction{
			sdkFunction1,
		},
	}

	t.Run("new function starts with version 0", func(t *testing.T) {
		// Create a test devserver with in-memory data store
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Register the app with one function
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify the function was created with version 0
		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 0, fnVersion)
		}
	})

	t.Run("re-registering same app config does not increment version", func(t *testing.T) {
		// Create a test devserver with in-memory data store
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Register the app with one function
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify the function was created with version 0
		fnVersions1 := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions1, 1)
		for _, fnVersion := range fnVersions1 {
			require.Equal(t, 0, fnVersion)
		}

		// Register the same app again
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Get the updated version
		fnVersions2 := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions2, 1)

		// fn versions don't change
		require.EqualValues(t, fnVersions1, fnVersions2)
	})

	t.Run("multiple re-registrations increment version correctly", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		expectedVersions := []int{0, 1, 2, 3, 4}

		// Register the function multiple times with different config
		for i, expectedVersion := range expectedVersions {

			// change function config in each iteration
			sdkFunction1.Timeouts = &inngest.Timeouts{
				Start: inngestgo.StrPtr(fmt.Sprintf("%dm", i)),
			}
			req.Functions[0] = sdkFunction1

			// Re-register the app
			_, err := api.register(ctx, req)
			require.NoError(t, err, "registration %d failed", i)

			// Verify the version is incremented
			fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
			require.Len(t, fnVersions, 1)
			for _, fnVersion := range fnVersions {
				require.Equal(t, expectedVersion, fnVersion, "function version should be %d after %d registration(s)", expectedVersion, i+1)
			}
		}
	})

	t.Run("different functions have independent versions", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// First registration with a single function has version=0
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 0, fnVersion)
		}

		// Second registration - add another function
		// existing function bumped up to version 1
		// new function set to version 0
		req.Functions = []sdk.SDKFunction{sdkFunction1, sdkFunction2}
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		require.Contains(t, fnVersions, sdkFunction1.Name)
		require.Equal(t, fnVersions[sdkFunction1.Name], 1)
		require.Contains(t, fnVersions, sdkFunction2.Name)
		require.Equal(t, fnVersions[sdkFunction2.Name], 0)

		// Now register only function1 again, removing function2
		req.Functions = []sdk.SDKFunction{sdkFunction1}
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Function1 should bumped up to version 2, function2 should be removed.
		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		require.Contains(t, fnVersions, sdkFunction1.Name)
		require.Equal(t, fnVersions[sdkFunction1.Name], 2)
	})

	// When one function's config is changes, all functions get their versions udpated, even those that don't have any change in config.
	t.Run("all function versions incremented on app sync", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		req.Functions = []sdk.SDKFunction{sdkFunction1, sdkFunction2}

		// Register the app
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify the functions were created with version 0
		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 0, fnVersion)
		}

		// update fn config for sdkFunction1
		sdkFunction1.Timeouts = &inngest.Timeouts{
			Start: inngestgo.StrPtr("2m"),
		}
		req.Functions[0] = sdkFunction1

		// Re-Register the app
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Verify both functions had versions incremented even though function2 had no change in config.
		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 1, fnVersion)
		}
	})

	t.Run("removing function increments versions of other functions on app sync", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		req.Functions = []sdk.SDKFunction{sdkFunction1, sdkFunction2}

		// Register the app
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify the functions were created with version 0
		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		for _, fnVersion := range fnVersions {
			require.Equal(t, fnVersion, 0)
		}

		// remove function1
		req.Functions = []sdk.SDKFunction{
			sdkFunction2,
		}

		// Re-Register the app
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Verify function1 is gone and function2 is now on version=1
		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		require.Contains(t, fnVersions, sdkFunction2.Name)
		require.Equal(t, fnVersions[sdkFunction2.Name], 1)
	})
}

// newTestDevServer creates a test devserver with in-memory data store
func newTestDevServer(t *testing.T) *devserver {
	t.Helper()

	// Create in-memory database
	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false, ForTest: true})
	require.NoError(t, err)

	// Initialize CQRS manager
	dbDriver := "sqlite"
	data := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})

	ds := &devserver{
		Data:        data,
		log:         logger.StdlibLogger(t.Context()),
		handlerLock: &sync.Mutex{},
		handlers:    []SDKHandler{},
	}

	return ds
}

func getFunctionIDandVersion(t *testing.T, ds *devserver, appName string) map[string]int {
	t.Helper()

	functionVersions := make(map[string]int)

	appID := inngest.DeterministicAppUUID(appName)
	funcs, err := ds.Data.GetFunctionsByAppInternalID(t.Context(), appID)
	require.NoError(t, err)

	for _, function := range funcs {
		var fn inngest.Function
		err = json.Unmarshal([]byte(function.Config), &fn)
		require.NoError(t, err)
		functionVersions[fn.Name] = fn.FunctionVersion
	}
	return functionVersions
}

func TestDevEndpoint_ReturnsInfoInDevMode(t *testing.T) {
	// Create devserver with dev mode (default)
	ds := newTestDevServer(t)
	ds.Opts = StartOpts{
		Config: config.Config{
			ServerKind: headers.ServerKindDev,
		},
	}

	// Create API router with no-op auth middleware for testing
	noAuthMiddleware := func(next http.Handler) http.Handler { return next }
	api := NewDevAPI(ds, DevAPIOptions{AuthMiddleware: noAuthMiddleware})

	// Create test request
	req := httptest.NewRequest("GET", "/dev", nil)
	w := httptest.NewRecorder()

	// Call through the router
	api.ServeHTTP(w, req)

	// Should return 200 with info
	require.Equal(t, http.StatusOK, w.Code)

	// Verify response is valid JSON
	var info InfoResponse
	err := json.Unmarshal(w.Body.Bytes(), &info)
	require.NoError(t, err)
}

func TestRegister_DuplicateAppCleanup(t *testing.T) {
	ctx := context.Background()

	sdkFunction := sdk.SDKFunction{
		Name: "Test Function",
		Slug: "test-function",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event: "test/event",
				},
			},
		},
		Steps: map[string]sdk.SDKStep{
			"step-1": {
				ID:   "step-1",
				Name: "test step",
				Runtime: map[string]any{
					"url": "http://localhost:3000/api/inngest",
				},
			},
		},
	}

	req := sdk.RegisterRequest{
		URL:       "http://localhost:3000/api/inngest",
		AppName:   "my-app",
		V:         "1",
		Functions: []sdk.SDKFunction{sdkFunction},
	}

	t.Run("placeholder from UI is cleaned up on re-register with same checksum", func(t *testing.T) {
		// This test reproduces a bug where adding an app URL through the
		// dev server UI creates a placeholder app that is never cleaned up
		// when the SDK re-registers with a matching checksum.
		//
		// The flow:
		// 1. SDK registers successfully → app created with ID based on app name
		// 2. User adds the same URL via UI → placeholder created with ID based on URL
		// 3. SDK re-registers (same checksum) → checksum early-return skips cleanup
		// 4. BUG: Two apps exist — the real one and the zombie placeholder

		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Step 1: SDK registers the app normally
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify: one app exists
		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1)
		require.Equal(t, "my-app", apps[0].Name)

		// Step 2: Simulate the UI "Add App" flow (CreateApp mutation).
		// This creates a placeholder app with an ID derived from the URL,
		// which differs from the registered app's ID (derived from app name).
		placeholderID := inngest.DeterministicAppUUID(req.URL)
		_, err = ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:  placeholderID,
			Url: req.URL,
			Error: sql.NullString{
				Valid:  true,
				String: deploy.DeployErrUnreachable.Error(),
			},
		})
		require.NoError(t, err)

		// Verify: two apps now exist (the real app + the placeholder)
		apps, err = ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 2, "expected both real app and placeholder to exist before re-registration")

		// Step 3: SDK re-registers with the same request (same checksum).
		// This triggers the checksum early-return path in register().
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Step 4: Assert that the placeholder was cleaned up.
		// There should be exactly one app — the real registered app.
		apps, err = ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "placeholder app should have been cleaned up during re-registration")
		require.Equal(t, "my-app", apps[0].Name)
	})

	t.Run("placeholder from -u flag is cleaned up on re-register with same checksum", func(t *testing.T) {
		// This test reproduces the same bug but via the `-u` flag path.
		// When pollSDKs() creates a placeholder before the SDK registers,
		// and then the SDK re-registers with a matching checksum, the
		// placeholder should be cleaned up.

		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Step 1: Simulate the `-u` flag creating a placeholder (pollSDKs initial loop)
		placeholderID := inngest.DeterministicAppUUID(req.URL)
		_, err := ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:  placeholderID,
			Url: req.URL,
			Error: sql.NullString{
				Valid:  true,
				String: deploy.DeployErrUnreachable.Error(),
			},
		})
		require.NoError(t, err)

		// Step 2: SDK registers (first time — placeholder has no name, so cleanup works)
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Verify: one app, placeholder was cleaned up
		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1)
		require.Equal(t, "my-app", apps[0].Name)

		// Step 3: Simulate a server restart where pollSDKs recreates the placeholder
		// (this happens if the database persists or if the polling loop re-creates it)
		_, err = ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:  placeholderID,
			Url: req.URL,
			Error: sql.NullString{
				Valid:  true,
				String: deploy.DeployErrUnreachable.Error(),
			},
		})
		require.NoError(t, err)

		// Verify: two apps exist again
		apps, err = ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 2)

		// Step 4: SDK re-registers with the same checksum
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Step 5: Assert that the placeholder was cleaned up
		apps, err = ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "placeholder app should have been cleaned up during re-registration")
		require.Equal(t, "my-app", apps[0].Name)
	})

	t.Run("two URLs serving same app ID results in one app", func(t *testing.T) {
		// When two different URLs serve the same app name, registering both
		// should result in only one app (the last one wins).

		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Register from URL1
		req1 := sdk.RegisterRequest{
			URL:       "http://localhost:3000/api/inngest",
			AppName:   "my-app",
			V:         "1",
			Functions: []sdk.SDKFunction{sdkFunction},
		}
		_, err := api.register(ctx, req1)
		require.NoError(t, err)

		// Register from URL2 with the same app name
		req2 := sdk.RegisterRequest{
			URL:       "http://localhost:3001/api/inngest",
			AppName:   "my-app",
			V:         "1",
			Functions: []sdk.SDKFunction{sdkFunction},
		}
		_, err = api.register(ctx, req2)
		require.NoError(t, err)

		// Should only have one app
		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "two URLs with the same app name should result in one app")
		require.Equal(t, "my-app", apps[0].Name)
	})
}

func TestDevEndpoint_Returns404InCloudMode(t *testing.T) {
	// Create devserver with cloud mode (self-hosted)
	ds := newTestDevServer(t)
	ds.Opts = StartOpts{
		Config: config.Config{
			ServerKind: headers.ServerKindCloud,
		},
	}

	// Create API router with no-op auth middleware for testing
	noAuthMiddleware := func(next http.Handler) http.Handler { return next }
	api := NewDevAPI(ds, DevAPIOptions{AuthMiddleware: noAuthMiddleware})

	// Create test request
	req := httptest.NewRequest("GET", "/dev", nil)
	w := httptest.NewRecorder()

	// Call through the router
	api.ServeHTTP(w, req)

	// Should return 404
	require.Equal(t, http.StatusNotFound, w.Code)
}
