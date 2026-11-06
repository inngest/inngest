package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestFunctionsResumeStartsQueuedRuns(t *testing.T) {
	for _, keyQueues := range []bool{false, true} {
		t.Run(fmt.Sprintf("key queues=%t", keyQueues), func(t *testing.T) {
			appID := uuid.New()
			ids := []uuid.UUID{uuid.New(), uuid.New()}
			provider := NewFunctionProvider(&fakeFunctionStore{
				app: &cqrs.App{ID: appID, Name: "app"},
				fns: []*cqrs.Function{
					{ID: ids[0], AppID: appID, Slug: "app-echo", Config: []byte(`{"name":"Echo","slug":"echo"}`)},
					{ID: ids[1], AppID: appID, Slug: "app-flaky", Config: []byte(`{"name":"Flaky","slug":"flaky"}`)},
				},
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
			q, err := queue.New(t.Context(), "bulk-resume-test", registry, opts...)
			require.NoError(t, err)
			provider.unpauser = q
			handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{
				Functions: provider, FunctionPauser: provider,
			}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
			require.NoError(t, err)
			requestedIDs := []string{"echo", "missing", "flaky"}
			body, err := json.Marshal(map[string]any{"functionIds": requestedIDs})
			require.NoError(t, err)
			request := func(action string, paused bool) {
				t.Helper()
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/"+action, strings.NewReader(string(body))))
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				var response struct {
					Data []struct {
						FunctionID string `json:"functionId"`
						Function   *struct {
							ID       string `json:"id"`
							IsPaused bool   `json:"isPaused"`
						} `json:"function"`
						Error *apiv2base.ErrorItem `json:"error"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
				require.Len(t, response.Data, 3)
				for i, item := range response.Data {
					require.Equal(t, requestedIDs[i], item.FunctionID)
					if i == 1 {
						require.Equal(t, &apiv2base.ErrorItem{Code: "not_found", Message: "Function not found"}, item.Error)
						require.Nil(t, item.Function)
						continue
					}
					require.Nil(t, item.Error)
					require.NotNil(t, item.Function)
					require.Equal(t, []string{"echo", "", "flaky"}[i], item.Function.ID)
					require.Equal(t, paused, item.Function.IsPaused)
				}
			}
			request("pause", true)
			first := []time.Time{provider.pauses.get(ids[0]), provider.pauses.get(ids[1])}
			request("pause", true)
			require.Equal(t, first, []time.Time{provider.pauses.get(ids[0]), provider.pauses.get(ids[1])})
			now := time.Now()
			runs := map[uuid.UUID]ulid.ULID{}
			for _, id := range ids {
				runs[id] = ulid.Make()
				_, err = shard.EnqueueItem(t.Context(), queue.QueueItem{
					FunctionID: id, WorkspaceID: consts.DevServerEnvID,
					Data: queue.Item{Kind: queue.KindStart, WorkspaceID: consts.DevServerEnvID, Identifier: state.Identifier{
						AccountID: consts.DevServerAccountID, WorkspaceID: consts.DevServerEnvID, WorkflowID: id, RunID: runs[id],
					}},
				}, now, queue.EnqueueOpts{})
				require.NoError(t, err)
			}
			if keyQueues {
				parts, err := shard.PeekShadowPartitions(t.Context(), nil, true, 10, now.Add(time.Second))
				require.NoError(t, err)
				require.Len(t, parts, 2)
				later := now.Add(queue.ShadowPartitionRefillPausedRequeueExtension)
				for _, part := range parts {
					require.NoError(t, shard.ShadowPartitionRequeue(t.Context(), part, &later))
				}
				parts, err = shard.PeekShadowPartitions(t.Context(), nil, true, 10, now.Add(time.Second))
				require.NoError(t, err)
				require.Empty(t, parts)
			} else {
				parts, err := shard.PartitionPeek(t.Context(), true, now.Add(time.Second), 10)
				require.NoError(t, err)
				require.Empty(t, parts)
			}
			request("resume", false)
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			started := make(chan queue.Item, 2)
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
			seen := map[uuid.UUID]ulid.ULID{}
			for range ids {
				select {
				case item := <-started:
					seen[item.Identifier.WorkflowID] = item.Identifier.RunID
					require.Equal(t, queue.KindStart, item.Kind)
				case <-time.After(5 * time.Second):
					t.Fatal("queued runs did not start promptly after bulk resume")
				}
			}
			require.Equal(t, runs, seen)
			request("resume", false)
		})
	}
}
