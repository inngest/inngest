package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
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
		fnVersions := getFunctionIDandVersion(t, ds, req.URL)
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
		fnVersions1 := getFunctionIDandVersion(t, ds, req.URL)
		require.Len(t, fnVersions1, 1)
		for _, fnVersion := range fnVersions1 {
			require.Equal(t, 0, fnVersion)
		}

		// Register the same app again
		_, err = api.register(ctx, req)
		require.NoError(t, err)

		// Get the updated version
		fnVersions2 := getFunctionIDandVersion(t, ds, req.URL)
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
			fnVersions := getFunctionIDandVersion(t, ds, req.URL)
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

		fnVersions := getFunctionIDandVersion(t, ds, req.URL)
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

		fnVersions = getFunctionIDandVersion(t, ds, req.URL)
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
		fnVersions = getFunctionIDandVersion(t, ds, req.URL)
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
		fnVersions := getFunctionIDandVersion(t, ds, req.URL)
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
		fnVersions = getFunctionIDandVersion(t, ds, req.URL)
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
		fnVersions := getFunctionIDandVersion(t, ds, req.URL)
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
		fnVersions = getFunctionIDandVersion(t, ds, req.URL)
		require.Len(t, fnVersions, 1)
		require.Contains(t, fnVersions, sdkFunction2.Name)
		require.Equal(t, fnVersions[sdkFunction2.Name], 1)
	})
}

// newTestDevServer creates a test devserver with in-memory data store
func newTestDevServer(t *testing.T) *devserver {
	t.Helper()

	// Create in-memory database
	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{InMemory: true, ForTest: true})
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

func getFunctionIDandVersion(t *testing.T, ds *devserver, URL string) map[string]int {
	t.Helper()

	functionVersions := make(map[string]int)

	appID := inngest.DeterministicAppUUID(URL)
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
