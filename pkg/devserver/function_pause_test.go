package devserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/testapi"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type pauseFunctionResponse struct {
	Data struct {
		ID       string `json:"id"`
		IsPaused bool   `json:"isPaused"`
	} `json:"data"`
}

type functionUnpauserFunc func(context.Context, string, queue.Scope) error

func (f functionUnpauserFunc) UnpauseFunction(ctx context.Context, shard string, scope queue.Scope) error {
	return f(ctx, shard, scope)
}

func TestFunctionPauseSharedState(t *testing.T) {
	id, appID := uuid.New(), uuid.New()
	provider := NewFunctionProvider(&fakeFunctionStore{
		app: &cqrs.App{ID: appID, Name: "app"},
		fns: []*cqrs.Function{{ID: id, AppID: appID, Slug: "app-fn", Config: []byte(`{"name":"Function","slug":"fn"}`)}},
	}, nil, nil)
	wakes := 0
	provider.unpauser = functionUnpauserFunc(func(_ context.Context, shard string, scope queue.Scope) error {
		wakes++
		require.Equal(t, consts.DefaultQueueShardName, shard)
		require.Equal(t, queue.Scope{AccountID: consts.DevServerAccountID, EnvID: consts.DevServerEnvID, FunctionID: id}, scope)
		return nil
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

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/resume", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response pauseFunctionResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Equal(t, "fn", response.Data.ID)
		require.False(t, response.Data.IsPaused)
		require.False(t, get())
		require.Equal(t, i+1, wakes)
	}
	resumed, err := provider.GetFunction(t.Context(), id.String())
	require.NoError(t, err)
	fn.PausedAt = time.Time{}
	require.Equal(t, fn, resumed)
	repeated, err = provider.ResumeFunction(t.Context(), resumed)
	require.NoError(t, err)
	require.Equal(t, resumed, repeated)
	require.Equal(t, 3, wakes)
	require.Equal(t, queue.PartitionPausedInfo{}, provider.pauses.partitionPaused(t.Context(), id))

	testHandler := testapi.New(testapi.Options{
		PauseFunction: func(id uuid.UUID) { provider.pauses.pause(id) },
	})
	rec := httptest.NewRecorder()
	testHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/function/pause?fnId="+id.String(), nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "OK", rec.Body.String())
	require.True(t, get())
}

func TestFunctionPauseResumeArchived(t *testing.T) {
	id, appID := uuid.New(), uuid.New()
	provider := NewFunctionProvider(&fakeFunctionStore{
		app: &cqrs.App{ID: appID, Name: "app"},
		fns: []*cqrs.Function{{
			ID: id, AppID: appID, Slug: "app-fn", Config: []byte(`{"name":"Function","slug":"fn"}`),
			ArchivedAt: time.Now().Add(-time.Hour),
		}},
	}, nil, nil)
	handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{
		Functions: provider, FunctionPauser: provider,
	}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)

	for _, action := range []string{"pause", "resume"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/"+action, nil))
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
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/v2/apps/app/functions/fn", nil))
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
}

func TestFunctionPauseConcurrentFirstWriteWins(t *testing.T) {
	provider := NewFunctionProvider(&fakeFunctionStore{}, nil, nil)
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

func TestFunctionResumeMissingUnpauser(t *testing.T) {
	provider := NewFunctionProvider(&fakeFunctionStore{}, nil, nil)
	fn, err := provider.PauseFunction(t.Context(), inngest.DeployedFunction{ID: uuid.New()})
	require.NoError(t, err)

	_, err = provider.ResumeFunction(t.Context(), fn)
	require.ErrorContains(t, err, "queue unpauser is not configured")
	require.Equal(t, fn.PausedAt, provider.pauses.get(fn.ID))
}

func TestFunctionResumeRetriesFailedWake(t *testing.T) {
	provider := NewFunctionProvider(&fakeFunctionStore{}, nil, nil)
	fn, err := provider.PauseFunction(t.Context(), inngest.DeployedFunction{
		ID: uuid.New(), AccountID: consts.DevServerAccountID, EnvironmentID: consts.DevServerEnvID,
	})
	require.NoError(t, err)
	require.Equal(t, queue.PartitionPausedInfo{Paused: true}, provider.pauses.partitionPaused(t.Context(), fn.ID))
	wakeErr := errors.New("queue unavailable")
	wakes := 0
	provider.unpauser = functionUnpauserFunc(func(_ context.Context, shard string, scope queue.Scope) error {
		wakes++
		require.Equal(t, consts.DefaultQueueShardName, shard)
		require.Equal(t, queue.Scope{AccountID: fn.AccountID, EnvID: fn.EnvironmentID, FunctionID: fn.ID}, scope)
		if wakes == 1 {
			return wakeErr
		}
		return nil
	})
	resumed, err := provider.ResumeFunction(t.Context(), fn)
	require.ErrorIs(t, err, wakeErr)
	require.Equal(t, inngest.DeployedFunction{}, resumed)
	require.Equal(t, queue.PartitionPausedInfo{Paused: true}, provider.pauses.partitionPaused(t.Context(), fn.ID))
	require.Equal(t, fn.PausedAt, provider.pauses.get(fn.ID))
	resumed, err = provider.ResumeFunction(t.Context(), fn)
	require.NoError(t, err)
	fn.PausedAt = time.Time{}
	require.Equal(t, fn, resumed)
	require.Equal(t, 2, wakes)
}

func TestFunctionResumeStartsQueuedRun(t *testing.T) {
	for _, keyQueues := range []bool{false, true} {
		t.Run(fmt.Sprintf("key queues=%t", keyQueues), func(t *testing.T) {
			id, appID := uuid.New(), uuid.New()
			provider := NewFunctionProvider(&fakeFunctionStore{
				app: &cqrs.App{ID: appID, Name: "app"},
				fns: []*cqrs.Function{{ID: id, AppID: appID, Slug: "app-fn", Config: []byte(`{"name":"Function","slug":"fn"}`)}},
			}, nil, nil)
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			opts := []queue.QueueOpt{
				queue.WithRunMode(queue.QueueRunMode{Partition: true, ShadowPartition: keyQueues}),
				queue.WithNumWorkers(10),
				queue.WithPollTick(10 * time.Millisecond),
				queue.WithShadowPollTick(10 * time.Millisecond),
				queue.WithPartitionPausedGetter(provider.pauses.partitionPaused),
				queue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return keyQueues }),
			}
			shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), opts...)
			registry, err := queue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			q, err := queue.New(t.Context(), "resume-test", registry, opts...)
			require.NoError(t, err)
			provider.unpauser = q
			handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{
				Functions: provider, FunctionPauser: provider,
			}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
			require.NoError(t, err)
			request := func(action string, paused bool) {
				t.Helper()
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/"+action, nil))
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				var response pauseFunctionResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
				require.Equal(t, "fn", response.Data.ID)
				require.Equal(t, paused, response.Data.IsPaused)
			}
			request("pause", true)
			now := time.Now()
			runID := ulid.Make()
			_, err = shard.EnqueueItem(t.Context(), queue.QueueItem{
				FunctionID: id, WorkspaceID: consts.DevServerEnvID,
				Data: queue.Item{Kind: queue.KindStart, WorkspaceID: consts.DevServerEnvID, Identifier: state.Identifier{
					AccountID: consts.DevServerAccountID, WorkspaceID: consts.DevServerEnvID, WorkflowID: id, RunID: runID,
				}},
			}, now, queue.EnqueueOpts{})
			require.NoError(t, err)
			if keyQueues {
				parts, err := shard.PeekShadowPartitions(t.Context(), nil, true, 10, now.Add(time.Second))
				require.NoError(t, err)
				require.Len(t, parts, 1)
				later := now.Add(queue.ShadowPartitionRefillPausedRequeueExtension)
				require.NoError(t, shard.ShadowPartitionRequeue(t.Context(), parts[0], &later))
				parts, err = shard.PeekShadowPartitions(t.Context(), nil, true, 10, now.Add(time.Second))
				require.NoError(t, err)
				require.Empty(t, parts)
			} else {
				parts, err := shard.PartitionPeek(t.Context(), true, now.Add(time.Second), 10)
				require.NoError(t, err)
				require.Empty(t, parts)
			}
			request("resume", false)
			if keyQueues {
				parts, err := shard.PeekShadowPartitions(t.Context(), nil, true, 10, time.Now().Add(time.Second))
				require.NoError(t, err)
				require.Len(t, parts, 1)
				require.Equal(t, id.String(), parts[0].PartitionID)
			} else {
				parts, err := shard.PartitionPeek(t.Context(), true, time.Now().Add(time.Second), 10)
				require.NoError(t, err)
				require.Len(t, parts, 1)
				require.Equal(t, id, *parts[0].FunctionID)
			}
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			started := make(chan queue.Item, 1)
			go func() {
				done <- q.Run(ctx, func(_ context.Context, _ queue.RunInfo, item queue.Item) (queue.RunResult, error) {
					started <- item
					return queue.RunResult{}, nil
				})
			}()
			defer func() {
				cancel()
				select {
				case err := <-done:
					if err != nil {
						require.ErrorIs(t, err, context.Canceled)
					}
				case <-time.After(5 * time.Second):
					t.Error("queue did not stop")
				}
			}()
			select {
			case item := <-started:
				require.Equal(t, runID, item.Identifier.RunID)
				require.Equal(t, id, item.Identifier.WorkflowID)
				require.Equal(t, queue.KindStart, item.Kind)
			case <-time.After(5 * time.Second):
				t.Fatal("queued run did not start promptly after resume")
			}
		})
	}
}
