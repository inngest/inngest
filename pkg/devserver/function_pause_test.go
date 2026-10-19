package devserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

type pauseFunctionResponse struct {
	Data struct {
		ID       string `json:"id"`
		IsPaused bool   `json:"isPaused"`
	} `json:"data"`
}

func TestFunctionPauseSharedState(t *testing.T) {
	id, appID := uuid.New(), uuid.New()
	provider := NewFunctionProvider(&fakeFunctionStore{
		app: &cqrs.App{ID: appID, Name: "app"},
		fns: []*cqrs.Function{{ID: id, AppID: appID, Slug: "app-fn", Config: []byte(`{"name":"Function","slug":"fn"}`)}},
	})
	handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{
		Functions: provider, FunctionPauser: provider,
	}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)

	get := func() bool {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/apps/app/functions/fn", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response pauseFunctionResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		return response.Data.IsPaused
	}
	require.False(t, get())
	var first time.Time
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/pause", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response pauseFunctionResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Equal(t, "fn", response.Data.ID)
		require.True(t, response.Data.IsPaused)
		require.True(t, get())
		if i == 0 {
			first = provider.pauses.get(id)
		}
		require.Equal(t, first, provider.pauses.get(id))
	}
	require.Equal(t, queue.PartitionPausedInfo{Paused: true}, provider.pauses.partitionPaused(t.Context(), id))
	require.Equal(t, queue.PartitionPausedInfo{}, provider.pauses.partitionPaused(t.Context(), uuid.New()))
	fn, err := provider.GetFunction(t.Context(), id.String())
	require.NoError(t, err)
	require.Equal(t, first, fn.PausedAt)
	repeated, err := provider.PauseFunction(t.Context(), fn)
	require.NoError(t, err)
	require.Equal(t, fn, repeated)
	page, err := provider.GetFunctions(t.Context(), "app", apiv2.GetFunctionsOpts{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Functions, 1)
	require.Equal(t, first, page.Functions[0].PausedAt)

	// test api shares state with the public api
	provider.pauses.unpause(id)
	require.False(t, get())
	require.Equal(t, queue.PartitionPausedInfo{}, provider.pauses.partitionPaused(t.Context(), id))
	provider.pauses.pause(id)
	require.True(t, get())
}

func TestFunctionPauseArchived(t *testing.T) {
	id, appID := uuid.New(), uuid.New()
	provider := NewFunctionProvider(&fakeFunctionStore{
		app: &cqrs.App{ID: appID, Name: "app"},
		fns: []*cqrs.Function{{
			ID: id, AppID: appID, Slug: "app-fn", Config: []byte(`{"name":"Function","slug":"fn"}`),
			ArchivedAt: time.Now().Add(-time.Hour),
		}},
	})
	handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{
		Functions: provider, FunctionPauser: provider,
	}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/pause", nil))
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	var response struct {
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Errors, 1)
	require.Equal(t, "not_found", response.Errors[0].Code)
	require.Equal(t, queue.PartitionPausedInfo{}, provider.pauses.partitionPaused(t.Context(), id))

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/v2/apps/app/functions/fn", nil))
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
}

func TestFunctionPauseConcurrentFirstWriteWins(t *testing.T) {
	provider := NewFunctionProvider(&fakeFunctionStore{})
	fn := inngest.DeployedFunction{ID: uuid.New()}
	const workers = 32
	results := make(chan time.Time, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	before := time.Now()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			paused, _ := provider.PauseFunction(t.Context(), fn)
			results <- paused.PausedAt
			other := uuid.New()
			provider.pauses.pause(other)
			provider.pauses.partitionPaused(t.Context(), fn.ID)
			provider.pauses.unpause(other)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	first := <-results
	require.False(t, first.Before(before))
	require.False(t, first.After(time.Now()))
	for at := range results {
		require.Equal(t, first, at)
	}
	require.Equal(t, first, provider.pauses.get(fn.ID))
	require.Equal(t, queue.PartitionPausedInfo{Paused: true}, provider.pauses.partitionPaused(t.Context(), fn.ID))
}
