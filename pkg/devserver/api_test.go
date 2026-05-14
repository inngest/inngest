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
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/enums"
	cronpkg "github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

type capturingCronSyncer struct {
	items []cronpkg.CronItem
}

func (c *capturingCronSyncer) Sync(_ context.Context, ci cronpkg.CronItem) error {
	c.items = append(c.items, ci)
	return nil
}

func (c *capturingCronSyncer) Items() []cronpkg.CronItem {
	return c.items
}

type semaphoreSetCall struct {
	accountID      uuid.UUID
	name           string
	idempotencyKey string
	capacity       int64
}

type capturingSemaphoreManager struct {
	setCalls []semaphoreSetCall
}

func (c *capturingSemaphoreManager) SetCapacity(_ context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) (constraintapi.SetResult, error) {
	c.setCalls = append(c.setCalls, semaphoreSetCall{
		accountID:      accountID,
		name:           name,
		idempotencyKey: idempotencyKey,
		capacity:       capacity,
	})
	return constraintapi.SetResult{Applied: true, Capacity: capacity}, nil
}

func (c *capturingSemaphoreManager) AdjustCapacity(context.Context, uuid.UUID, string, string, int64) (constraintapi.AdjustResult, error) {
	return constraintapi.AdjustResult{}, nil
}

func (c *capturingSemaphoreManager) GetCapacity(context.Context, uuid.UUID, string, string) (int64, int64, error) {
	return 0, 0, nil
}

func (c *capturingSemaphoreManager) ReleaseSemaphore(context.Context, uuid.UUID, string, string, string, int64) error {
	return nil
}

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

	t.Run("new function starts with version 1", func(t *testing.T) {
		// Create a test devserver with in-memory data store
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Register the app with one function
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify the function was created with version 1
		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 1, fnVersion)
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

		// Verify the function was created with version 1
		fnVersions1 := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions1, 1)
		for _, fnVersion := range fnVersions1 {
			require.Equal(t, 1, fnVersion)
		}

		// Register the same app again
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Get the updated version — same config means no increment
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

		expectedVersions := []int{1, 2, 3, 4, 5}

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

		// First registration with a single function has version=1
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 1, fnVersion)
		}

		// Second registration - add another function
		// existing function bumped up to version 2
		// new function set to version 1
		req.Functions = []sdk.SDKFunction{sdkFunction1, sdkFunction2}
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		require.Contains(t, fnVersions, sdkFunction1.Name)
		require.Equal(t, fnVersions[sdkFunction1.Name], 2)
		require.Contains(t, fnVersions, sdkFunction2.Name)
		require.Equal(t, fnVersions[sdkFunction2.Name], 1)

		// Now register only function1 again, removing function2
		req.Functions = []sdk.SDKFunction{sdkFunction1}
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Function1 should bumped up to version 3, function2 should be removed.
		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		require.Contains(t, fnVersions, sdkFunction1.Name)
		require.Equal(t, fnVersions[sdkFunction1.Name], 3)
	})

	// When one function's config changes, all functions get their versions updated, even those that don't have any change in config.
	t.Run("all function versions incremented on app sync", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		req.Functions = []sdk.SDKFunction{sdkFunction1, sdkFunction2}

		// Register the app
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		// Verify the functions were created with version 1
		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		for _, fnVersion := range fnVersions {
			require.Equal(t, 1, fnVersion)
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
			require.Equal(t, 2, fnVersion)
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

		// Verify the functions were created with version 1
		fnVersions := getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 2)
		for _, fnVersion := range fnVersions {
			require.Equal(t, fnVersion, 1)
		}

		// remove function1
		req.Functions = []sdk.SDKFunction{
			sdkFunction2,
		}

		// Re-Register the app
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Verify function1 is gone and function2 is now on version=2
		fnVersions = getFunctionIDandVersion(t, ds, req.AppName)
		require.Len(t, fnVersions, 1)
		require.Contains(t, fnVersions, sdkFunction2.Name)
		require.Equal(t, fnVersions[sdkFunction2.Name], 2)
	})
}

func TestRegister_BlockedSDKVersion(t *testing.T) {
	ctx := context.Background()

	sdkFunction := sdk.SDKFunction{
		Name: "Test Function",
		Slug: "test-function",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{Event: "test/event"},
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

	newRequest := func(version string) sdk.RegisterRequest {
		return sdk.RegisterRequest{
			URL:       "http://localhost:3000/api/inngest",
			AppName:   "blocked-sdk-app",
			SDK:       fmt.Sprintf("js:%s", version),
			V:         "1",
			Functions: []sdk.SDKFunction{sdkFunction},
		}
	}

	t.Run("blocked versions are rejected and persisted as sync errors", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{devserver: ds}

		_, err := api.register(ctx, newRequest("v3.35.0"))
		require.Error(t, err)

		var publicErr publicerr.Error
		require.ErrorAs(t, err, &publicErr)
		require.Contains(t, publicErr.Message, "known security vulnerability")

		app, err := ds.Data.GetAppByName(ctx, consts.DevServerEnvID, "blocked-sdk-app")
		require.NoError(t, err)
		require.True(t, app.Error.Valid)
		require.Equal(t, deploy.DeployErrBlockedSDKVersion.Error(), app.Error.String)
		require.Equal(t, "v3.35.0", app.SdkVersion)
	})

	t.Run("blocked check runs before successful checksum short circuit", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{devserver: ds}
		req := newRequest("v3.35.0")

		sum, err := req.Checksum()
		require.NoError(t, err)

		_, err = ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:          inngest.DeterministicAppUUID(req.AppName),
			Name:        req.AppName,
			SdkLanguage: req.SDKLanguage(),
			SdkVersion:  req.SDKVersion(),
			Checksum:    sum,
			Url:         req.URL,
			Method:      enums.AppMethodServe.String(),
		})
		require.NoError(t, err)

		_, err = api.register(ctx, req)
		require.Error(t, err)

		var publicErr publicerr.Error
		require.ErrorAs(t, err, &publicErr)
		require.Contains(t, publicErr.Message, "known security vulnerability")

		app, err := ds.Data.GetAppByName(ctx, consts.DevServerEnvID, req.AppName)
		require.NoError(t, err)
		require.True(t, app.Error.Valid)
		require.Equal(t, deploy.DeployErrBlockedSDKVersion.Error(), app.Error.String)
	})
}

func TestRegister_CronJitterPropagation(t *testing.T) {
	ctx := context.Background()

	jitter := "5m"
	sdkFunction := sdk.SDKFunction{
		Name: "Cron Function",
		Slug: "cron-function",
		Triggers: []inngest.Trigger{
			{
				CronTrigger: &inngest.CronTrigger{
					Cron:   "0 * * * *",
					Jitter: &jitter,
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
		AppName:   "cron-app",
		V:         "1",
		Functions: []sdk.SDKFunction{sdkFunction},
	}

	// Create a test devserver with in-memory data store
	// and a capturing cron syncer
	ds := newTestDevServer(t)
	syncer := &capturingCronSyncer{}
	ds.CronSyncer = syncer
	api := &devapi{devserver: ds}

	// register the app with a cron function that has jitter configured
	_, err := api.register(ctx, req)
	require.NoError(t, err)

	// Verify the cron item was synced
	items := syncer.Items()
	require.Len(t, items, 1)
	require.Equal(t, "0 * * * *", items[0].Expression)

	// Verify jitter is persisted in the function config and retrievable.
	fns, err := ds.Data.Functions(ctx)
	require.NoError(t, err)
	require.Len(t, fns, 1)
	require.Equal(t, 5*time.Minute, fns[0].CronJitter("0 * * * *"))

	updatedJitter := "1m"
	sdkFunction.Triggers[0].CronTrigger.Jitter = &updatedJitter
	req.Functions[0] = sdkFunction

	_, err = api.register(ctx, req)
	require.NoError(t, err)

	// Verify the cron item was synced a second time again with the updated jitter
	items = syncer.Items()
	require.Len(t, items, 2)

	// Verify updated jitter is in the function config.
	fns, err = ds.Data.Functions(ctx)
	require.NoError(t, err)
	require.Len(t, fns, 1)
	require.Equal(t, 1*time.Minute, fns[0].CronJitter("0 * * * *"))
}

// newTestDevServer creates a test devserver with in-memory data store
func newTestDevServer(t *testing.T) *devserver {
	t.Helper()

	// Create in-memory database
	db, err := base_cqrs.New(t.Context(), base_cqrs.BaseCQRSOptions{Persist: false, ForTest: true})
	require.NoError(t, err)

	// Initialize CQRS manager
	adapter := dbsqlite.New(db)
	data := base_cqrs.NewCQRS(adapter)

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

	t.Run("port mismatch between UI URL and SDK URL does not create duplicate", func(t *testing.T) {
		// This tests the scenario where a user adds a URL without an explicit
		// port (e.g. "http://myhost/api/inngest") via the UI, but the SDK
		// reports its URL with the default port (e.g. "http://myhost:80/api/inngest").
		//
		// Because NormalizeAppURL does not strip default ports, these are
		// treated as different URLs, so the GetAppByURL cleanup can't find
		// the placeholder even on the first registration.

		ds := newTestDevServer(t)
		api := &devapi{
			devserver: ds,
		}

		// Step 1: User adds "http://myhost/api/inngest" via UI (no port).
		// This creates a placeholder with ID based on the URL without port.
		uiURL := "http://myhost/api/inngest"
		placeholderID := inngest.DeterministicAppUUID(uiURL)
		_, err := ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:  placeholderID,
			Url: uiURL,
			Error: sql.NullString{
				Valid:  true,
				String: deploy.DeployErrUnreachable.Error(),
			},
		})
		require.NoError(t, err)

		// Step 2: SDK registers with the explicit default port in its URL.
		// The SDK self-reports "http://myhost:80/api/inngest".
		sdkReq := sdk.RegisterRequest{
			URL:       "http://myhost:80/api/inngest",
			AppName:   "my-app",
			V:         "1",
			Functions: []sdk.SDKFunction{sdkFunction},
		}
		_, err = api.register(ctx, sdkReq)
		require.NoError(t, err)

		// There should be exactly one app. The placeholder should have been
		// cleaned up since both URLs refer to the same host.
		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "port mismatch (no port vs :80) should not create duplicate apps")
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

	t.Run("URL rotation keeps fn concurrency semaphore keyed by persisted function ID", func(t *testing.T) {
		ds := newTestDevServer(t)
		sm := &capturingSemaphoreManager{}
		ds.SemaphoreManager = sm
		api := &devapi{
			devserver: ds,
		}

		fn := sdkFunction
		fn.Concurrency = &inngest.ConcurrencyLimits{
			Fn: []inngest.FnConcurrency{
				{
					Limit: 3,
				},
			},
		}

		req1 := sdk.RegisterRequest{
			URL:       "http://localhost:3000/api/inngest",
			AppName:   "my-app",
			V:         "1",
			Functions: []sdk.SDKFunction{fn},
		}
		_, err := api.register(ctx, req1)
		require.NoError(t, err)

		funcs, err := ds.Data.GetFunctionsByAppInternalID(ctx, inngest.DeterministicAppUUID("my-app"))
		require.NoError(t, err)
		require.Len(t, funcs, 1)
		persistedID := funcs[0].ID
		require.Len(t, sm.setCalls, 1)
		require.Equal(t, constraintapi.SemaphoreIDFn(persistedID), sm.setCalls[0].name)
		require.Equal(t, int64(3), sm.setCalls[0].capacity)

		rotatedFn := fn
		rotatedFn.Steps = map[string]sdk.SDKStep{
			"step-1": {
				ID:   "step-1",
				Name: "test step",
				Runtime: map[string]any{
					"url": "http://localhost:3001/api/inngest",
				},
			},
		}
		req2 := sdk.RegisterRequest{
			URL:       "http://localhost:3001/api/inngest",
			AppName:   "my-app",
			V:         "1",
			Functions: []sdk.SDKFunction{rotatedFn},
		}
		_, err = api.register(ctx, req2)
		require.NoError(t, err)

		funcs, err = ds.Data.GetFunctionsByAppInternalID(ctx, inngest.DeterministicAppUUID("my-app"))
		require.NoError(t, err)
		require.Len(t, funcs, 1)
		require.Equal(t, persistedID, funcs[0].ID)
		require.Len(t, sm.setCalls, 2)
		require.Equal(t, constraintapi.SemaphoreIDFn(persistedID), sm.setCalls[1].name)
		require.Equal(t, int64(3), sm.setCalls[1].capacity)
	})

	t.Run("v1.13 legacy URL-keyed app row is adopted on resync, function UUID survives", func(t *testing.T) {
		// In v1.13.x and earlier, register() seeded the deterministic app id
		// from r.URL. v1.15.0 (#3361) flipped the seed to r.AppName. Customer
		// DBs created on the old code therefore hold rows keyed by sha1(URL),
		// while a re-sync on the new code computes sha1(AppName) — a
		// different uuid. Without the partial unique index + UpsertAppByName,
		// the new register would insert a parallel row and orphan the
		// pre-existing functions. This test pre-populates the v1.13 state and
		// asserts adoption: legacy id is reused, function uuid is unchanged.
		ds := newTestDevServer(t)
		api := &devapi{devserver: ds}

		legacyAppID := inngest.DeterministicAppUUID(util.NormalizeAppURL(req.URL, false))
		nameAppID := inngest.DeterministicAppUUID(req.AppName)
		require.NotEqual(t, legacyAppID, nameAppID, "fixture must reproduce the seed mismatch")

		_, err := ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:   legacyAppID,
			Name: req.AppName,
			Url:  req.URL,
		})
		require.NoError(t, err)

		legacyFnID := uuid.New()
		fnConfig, err := json.Marshal(inngest.Function{
			ID:   legacyFnID,
			Name: "Test Function",
			Slug: "test-function",
			Triggers: []inngest.Trigger{
				{EventTrigger: &inngest.EventTrigger{Event: "test/event"}},
			},
		})
		require.NoError(t, err)
		_, err = ds.Data.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        legacyFnID,
			AppID:     legacyAppID,
			Name:      "Test Function",
			Slug:      "test-function",
			Config:    string(fnConfig),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		_, err = api.register(ctx, req)
		require.NoError(t, err)

		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "register must adopt the legacy row, not insert a parallel one")
		require.Equal(t, legacyAppID, apps[0].ID, "legacy URL-derived id must be preserved")
		require.Equal(t, req.AppName, apps[0].Name)

		funcs, err := ds.Data.GetFunctionsByAppInternalID(ctx, legacyAppID)
		require.NoError(t, err)
		require.Len(t, funcs, 1)
		require.Equal(t, legacyFnID, funcs[0].ID, "function uuid must survive the upgrade")
		require.Equal(t, legacyAppID, funcs[0].AppID, "function must remain attached to the legacy app id")
	})

	t.Run("placeholder upsert with empty name does not clobber an existing real name", func(t *testing.T) {
		// The -u startup loop, autodiscovery, and the UI add-app flow upsert
		// with name="" to set the unreachable error on a URL-derived id.
		// Before the fix in queries.sql, that ON CONFLICT(id) update would
		// blank the existing row's name to '' (the "state iii" precondition
		// for the v1.13 → v1.19 orphaning bug). The query now skips the name
		// assignment when excluded.name is empty.
		ds := newTestDevServer(t)

		legacyAppID := inngest.DeterministicAppUUID(util.NormalizeAppURL(req.URL, false))
		_, err := ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:   legacyAppID,
			Name: req.AppName,
			Url:  req.URL,
		})
		require.NoError(t, err)

		// Mimic service.go:315: placeholder UpsertApp with name="" and an
		// unreachable error. This must not erase the existing row's name.
		_, err = ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:  legacyAppID,
			Url: req.URL,
			Error: sql.NullString{
				Valid:  true,
				String: deploy.DeployErrUnreachable.Error(),
			},
		})
		require.NoError(t, err)

		app, err := ds.Data.GetAppByID(ctx, legacyAppID)
		require.NoError(t, err)
		require.Equal(t, req.AppName, app.Name, "placeholder upsert must not blank the name")
	})

	t.Run("fresh install with no legacy row uses name-derived ID", func(t *testing.T) {
		ds := newTestDevServer(t)
		api := &devapi{devserver: ds}

		_, err := api.register(ctx, req)
		require.NoError(t, err)

		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1)
		require.Equal(t, inngest.DeterministicAppUUID(req.AppName), apps[0].ID,
			"fresh install must mint the name-derived id, not adopt anything")
		require.Equal(t, req.AppName, apps[0].Name)
	})

	t.Run("URL-keyed row with blanked name and attached functions is adopted, not deleted", func(t *testing.T) {
		// Customer's actual post-upgrade state (T-7175 re-open):
		//   - id = sha1(URL)  (the pre-v1.15 deterministic id)
		//   - name = ''       (blanked by an earlier -u / autodiscovery upsert
		//                      before the CASE WHEN excluded.name = '' guard
		//                      shipped)
		//   - active, with the original function rows still attached
		//
		// The pre-fix cleanup loop matched (name='' && URL matches), soft-
		// deleted the row, and let UpsertAppByName insert a fresh row at
		// sha1(r.AppName). GetActiveFunctionByAppAndSlug then couldn't see
		// the legacy functions (their app is archived), so SDK-minted uuids
		// replaced the originals - same orphaning failure mode as the
		// original v1.13 → v1.19 bug, just from a different starting state.
		//
		// The fix adopts the row by id (renaming '' → r.AppName) when it
		// has attached functions; UpsertAppByName then resolves it by name.
		ds := newTestDevServer(t)
		api := &devapi{devserver: ds}

		legacyAppID := inngest.DeterministicAppUUID(util.NormalizeAppURL(req.URL, false))

		_, err := ds.Data.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:   legacyAppID,
			Name: "", // blanked by the upstream defect
			Url:  req.URL,
		})
		require.NoError(t, err)

		legacyFnID := uuid.New()
		fnConfig, err := json.Marshal(inngest.Function{
			ID:   legacyFnID,
			Name: "Test Function",
			Slug: "test-function",
			Triggers: []inngest.Trigger{
				{EventTrigger: &inngest.EventTrigger{Event: "test/event"}},
			},
		})
		require.NoError(t, err)
		_, err = ds.Data.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
			ID:        legacyFnID,
			AppID:     legacyAppID,
			Name:      "Test Function",
			Slug:      "test-function",
			Config:    string(fnConfig),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		_, err = api.register(ctx, req)
		require.NoError(t, err)

		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "adoption must leave exactly one active row")
		require.Equal(t, legacyAppID, apps[0].ID, "legacy id must be preserved by adoption")
		require.Equal(t, req.AppName, apps[0].Name, "adoption must restore the real app name")

		// The original function uuid must survive — both because the apps
		// row was not archived (the adoption path) and because
		// GetActiveFunctionByAppAndSlug now finds the row via apps.name.
		funcs, err := ds.Data.GetFunctionsByAppInternalID(ctx, legacyAppID)
		require.NoError(t, err)
		require.Len(t, funcs, 1)
		require.Equal(t, legacyFnID, funcs[0].ID, "function uuid must survive adoption")
		require.Equal(t, legacyAppID, funcs[0].AppID)

		// And the candidate name-derived id was never written: only the
		// adopted legacy row exists.
		nameAppID := inngest.DeterministicAppUUID(req.AppName)
		if nameAppID != legacyAppID {
			_, err = ds.Data.GetAppByID(ctx, nameAppID)
			require.ErrorIs(t, err, sql.ErrNoRows)
		}
	})

	t.Run("archived app row is revived on resync, function UUID survives", func(t *testing.T) {
		// Customer archives an app via the UI (or any path that flips
		// archived_at), then re-syncs from the SDK under the same name.
		// UpsertAppByName must revive the archived row in place: same id,
		// archived_at cleared, function uuid unchanged.
		ds := newTestDevServer(t)
		api := &devapi{devserver: ds}

		// First sync to create the app + a function. The deterministic
		// candidate id IS the row's id since this is a fresh install.
		_, err := api.register(ctx, req)
		require.NoError(t, err)

		appID := inngest.DeterministicAppUUID(req.AppName)
		funcs, err := ds.Data.GetFunctionsByAppInternalID(ctx, appID)
		require.NoError(t, err)
		require.Len(t, funcs, 1)
		originalFnID := funcs[0].ID

		// Archive the app (simulating the UI archive action).
		require.NoError(t, ds.Data.DeleteApp(ctx, appID))
		apps, err := ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Empty(t, apps, "archived row must drop out of the active listing")

		// Resync with the same payload. The archived row should revive.
		_, err = api.register(ctx, req)
		require.NoError(t, err, "resync after archive must not collide on PK")

		apps, err = ds.Data.GetAllApps(ctx, consts.DevServerEnvID)
		require.NoError(t, err)
		require.Len(t, apps, 1, "revival must reuse the row, not create a parallel one")
		require.Equal(t, appID, apps[0].ID, "revived row keeps its original id")
		require.Equal(t, req.AppName, apps[0].Name)

		funcs, err = ds.Data.GetFunctionsByAppInternalID(ctx, appID)
		require.NoError(t, err)
		require.Len(t, funcs, 1)
		require.Equal(t, originalFnID, funcs[0].ID, "function uuid must survive archive + revive")
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
