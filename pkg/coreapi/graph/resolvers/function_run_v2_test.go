package resolvers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/generated"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type runsMetadataManager struct {
	cqrs.Manager

	runs        []*cqrs.TraceRun
	apps        map[uuid.UUID]*cqrs.App
	functions   map[uuid.UUID]*cqrs.Function
	appErr      error
	functionErr error

	singleAppLookups      atomic.Int64
	singleFunctionLookups atomic.Int64
	bulkAppLookups        atomic.Int64
	bulkFunctionLookups   atomic.Int64
}

func (m *runsMetadataManager) GetTraceRuns(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	return m.runs, nil
}

func (m *runsMetadataManager) GetEventsByInternalIDs(context.Context, []ulid.ULID) ([]*cqrs.Event, error) {
	return nil, nil
}

func (m *runsMetadataManager) GetAppByID(_ context.Context, id uuid.UUID) (*cqrs.App, error) {
	m.singleAppLookups.Add(1)
	return m.apps[id], nil
}

func (m *runsMetadataManager) GetFunctionByInternalUUID(_ context.Context, id uuid.UUID) (*cqrs.Function, error) {
	m.singleFunctionLookups.Add(1)
	return m.functions[id], nil
}

func (m *runsMetadataManager) GetAppsByIDs(_ context.Context, ids []uuid.UUID) ([]*cqrs.App, error) {
	m.bulkAppLookups.Add(1)
	if m.appErr != nil {
		return nil, m.appErr
	}
	apps := make([]*cqrs.App, 0, len(ids))
	for _, id := range ids {
		if app := m.apps[id]; app != nil {
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func (m *runsMetadataManager) GetFunctionsByInternalUUIDs(_ context.Context, ids []uuid.UUID) ([]*cqrs.Function, error) {
	m.bulkFunctionLookups.Add(1)
	if m.functionErr != nil {
		return nil, m.functionErr
	}
	functions := make([]*cqrs.Function, 0, len(ids))
	for _, id := range ids {
		if fn := m.functions[id]; fn != nil {
			functions = append(functions, fn)
		}
	}
	return functions, nil
}

func TestRunsMetadataLookupsBatchAcrossGraphQLRuns(t *testing.T) {
	for _, tc := range []struct {
		name              string
		rows              int
		repeatMetadataIDs bool
	}{
		{name: "one row", rows: 1},
		{name: "100 repeated metadata IDs", rows: 100, repeatMetadataIDs: true},
		{name: "100 distinct metadata IDs", rows: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := newRunsMetadataManager(tc.rows, tc.repeatMetadataIDs)
			response := executeRunsMetadataQuery(t, runsMetadataHandler(manager))

			require.Empty(t, response.Errors, response.Body)
			require.Len(t, response.Data.Runs.Edges, tc.rows)
			for i, edge := range response.Data.Runs.Edges {
				metadataIndex := i
				if tc.repeatMetadataIDs {
					metadataIndex = 0
				}
				require.Equal(t, fmt.Sprintf("app-%d", metadataIndex), edge.Node.App.Name)
				require.Equal(t, fmt.Sprintf("Function %d", metadataIndex), edge.Node.Function.Name)
				require.Equal(t, fmt.Sprintf("fn-%d", metadataIndex), edge.Node.Function.Slug)
			}

			t.Logf(
				"rows=%d single app=%d function=%d bulk app=%d function=%d",
				tc.rows,
				manager.singleAppLookups.Load(),
				manager.singleFunctionLookups.Load(),
				manager.bulkAppLookups.Load(),
				manager.bulkFunctionLookups.Load(),
			)
			require.Zero(t, manager.singleAppLookups.Load(), "runs metadata must not issue one app CQRS call per row")
			require.Zero(t, manager.singleFunctionLookups.Load(), "runs metadata must not issue one function CQRS call per row")
			require.Greater(t, manager.bulkAppLookups.Load(), int64(0))
			require.Greater(t, manager.bulkFunctionLookups.Load(), int64(0))
			if tc.rows == 1 || tc.repeatMetadataIDs {
				require.Equal(t, int64(1), manager.bulkAppLookups.Load(), "same-request app metadata should be batched")
				require.Equal(t, int64(1), manager.bulkFunctionLookups.Load(), "same-request function metadata should be batched")
			} else {
				require.Less(t, manager.bulkAppLookups.Load(), int64(tc.rows), "distinct app metadata must not fan out one call per row")
				require.Less(t, manager.bulkFunctionLookups.Load(), int64(tc.rows), "distinct function metadata must not fan out one call per row")
			}
		})
	}
}

func TestRunsMetadataLookupErrorsRemainGraphQLErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*runsMetadataManager)
		message   string
	}{
		{
			name: "app database error",
			configure: func(m *runsMetadataManager) {
				m.appErr = errors.New("app database unavailable")
			},
			message: "app database unavailable",
		},
		{
			name: "function database error",
			configure: func(m *runsMetadataManager) {
				m.functionErr = errors.New("function database unavailable")
			},
			message: "error retrieving function: function database unavailable",
		},
		{
			name: "missing app",
			configure: func(m *runsMetadataManager) {
				delete(m.apps, m.runs[0].AppID)
			},
			message: "sql: no rows in result set",
		},
		{
			name: "missing function",
			configure: func(m *runsMetadataManager) {
				delete(m.functions, m.runs[0].FunctionID)
			},
			message: "error retrieving function: sql: no rows in result set",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := newRunsMetadataManager(1, false)
			tc.configure(manager)
			response := executeRunsMetadataQuery(t, runsMetadataHandler(manager))
			require.NotEmpty(t, response.Errors, response.Body)
			require.Contains(t, response.Body, tc.message)
		})
	}
}

func TestRunsMetadataLoaderCacheIsRequestLocal(t *testing.T) {
	manager := newRunsMetadataManager(1, true)
	handler := runsMetadataHandler(manager)

	for range 2 {
		response := executeRunsMetadataQuery(t, handler)
		require.Empty(t, response.Errors, response.Body)
	}

	t.Logf(
		"requests=2 single app=%d function=%d bulk app=%d function=%d",
		manager.singleAppLookups.Load(),
		manager.singleFunctionLookups.Load(),
		manager.bulkAppLookups.Load(),
		manager.bulkFunctionLookups.Load(),
	)
	// A request-local loader cache batches each request independently. A
	// process-wide cache would incorrectly leave these counters at one.
	require.Zero(t, manager.singleAppLookups.Load())
	require.Zero(t, manager.singleFunctionLookups.Load())
	require.Equal(t, int64(2), manager.bulkAppLookups.Load())
	require.Equal(t, int64(2), manager.bulkFunctionLookups.Load())
}

type runsMetadataResponse struct {
	Errors []json.RawMessage `json:"errors"`
	Data   struct {
		Runs struct {
			Edges []struct {
				Node struct {
					App struct {
						Name string `json:"name"`
					} `json:"app"`
					Function struct {
						Name string `json:"name"`
						Slug string `json:"slug"`
					} `json:"function"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"runs"`
	} `json:"data"`
	Body string
}

func runsMetadataHandler(manager *runsMetadataManager) http.Handler {
	schema := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: &Resolver{Data: manager},
	}))
	return loader.Middleware(loader.LoaderParams{DB: manager})(schema)
}

func executeRunsMetadataQuery(t *testing.T, h http.Handler) runsMetadataResponse {
	t.Helper()

	requestBody := []byte(`{"query":"query { runs(first: 100, orderBy: [{field: QUEUED_AT, direction: DESC}], filter: {from: \"2020-01-01T00:00:00Z\", until: \"2030-01-01T00:00:00Z\"}) { edges { node { app { name } function { name slug } } } } }"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/gql", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, request)

	var response runsMetadataResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response), recorder.Body.String())
	response.Body = recorder.Body.String()
	require.Equal(t, http.StatusOK, recorder.Code, response.Body)
	return response
}

func newRunsMetadataManager(rows int, repeatMetadataIDs bool) *runsMetadataManager {
	manager := &runsMetadataManager{
		apps:      make(map[uuid.UUID]*cqrs.App, rows),
		functions: make(map[uuid.UUID]*cqrs.Function, rows),
		runs:      make([]*cqrs.TraceRun, 0, rows),
	}

	for i := 0; i < rows; i++ {
		metadataIndex := i
		if repeatMetadataIDs {
			metadataIndex = 0
		}
		appID := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("app-%d", metadataIndex)))
		functionID := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("function-%d", metadataIndex)))
		manager.apps[appID] = &cqrs.App{ID: appID, Name: fmt.Sprintf("app-%d", metadataIndex)}
		manager.functions[functionID] = &cqrs.Function{
			ID:     functionID,
			AppID:  appID,
			Name:   fmt.Sprintf("Function %d", metadataIndex),
			Slug:   fmt.Sprintf("fn-%d", metadataIndex),
			Config: json.RawMessage(`{"steps":[{"id":"step","name":"step","uri":"ws://example.com"}]}`),
		}
		manager.runs = append(manager.runs, &cqrs.TraceRun{
			AppID:      appID,
			FunctionID: functionID,
			TraceID:    fmt.Sprintf("trace-%d", i),
			RunID:      ulid.Make().String(),
			QueuedAt:   time.Now().UTC(),
			Status:     enums.RunStatusCompleted,
			Cursor:     fmt.Sprintf("cursor-%d", i),
		})
	}

	return manager
}
