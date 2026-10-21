package devserver

import (
	"encoding/json"
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
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestCreateCancellationPersistsAndQueues(t *testing.T) {
	ctx := t.Context()
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	id, appID := uuid.New(), uuid.New()
	functions := NewFunctionProvider(&fakeFunctionStore{
		app: &cqrs.App{
			ID:   appID,
			Name: "app",
		},
		fns: []*cqrs.Function{{
			ID:     id,
			AppID:  appID,
			Slug:   "app-fn",
			Config: []byte(`{"name":"Function","slug":"fn"}`),
		}},
	})
	functions.pauses.pause(id)
	shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), queue.WithPartitionPausedGetter(functions.pauses.partitionPaused))
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	producer := queue.NewProducer(registry)
	reader := cancellation.NewRedisReader(rc, "")
	checker := cancellation.NewChecker(reader)
	handler, err := apiv2.NewHTTPHandler(ctx, apiv2.ServiceOptions{
		Functions:     functions,
		Cancellations: NewCancellationProvider(cancellation.NewRedisWriter(rc, producer, "")),
	}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)
	from := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	until := time.Now().Truncate(time.Millisecond)
	runID := ulid.MustNew(ulid.Timestamp(from), nil)
	require.NoError(t, producer.Enqueue(ctx, queue.Item{
		Kind:        queue.KindStart,
		WorkspaceID: consts.DevServerEnvID,
		Identifier: state.Identifier{
			AccountID:   consts.DevServerAccountID,
			WorkspaceID: consts.DevServerEnvID,
			AppID:       appID,
			WorkflowID:  id,
			RunID:       runID,
		},
	}, from, queue.EnqueueOpts{}))
	body, err := json.Marshal(map[string]any{"from": from, "until": until, "if": "event.data.cancel == true", "name": "cleanup"})
	require.NoError(t, err)
	var ids []ulid.ULID
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/cancellations", strings.NewReader(string(body))))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var response struct {
			Data struct {
				ID, AppID, FunctionID, Name string
				If                          string
				From, Until, CreatedAt      time.Time
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		cancelID, err := ulid.Parse(response.Data.ID)
		require.NoError(t, err)
		ids = append(ids, cancelID)
		require.Equal(t, "app", response.Data.AppID)
		require.Equal(t, "fn", response.Data.FunctionID)
		require.Equal(t, "cleanup", response.Data.Name)
		require.Equal(t, "event.data.cancel == true", response.Data.If)
		require.Equal(t, from.UTC(), response.Data.From)
		require.Equal(t, until.UTC(), response.Data.Until)
	}
	require.NotEqual(t, ids[0], ids[1])
	records, err := reader.ReadAt(ctx, consts.DevServerEnvID, id, from)
	require.NoError(t, err)
	require.Len(t, records, 2)
	for _, c := range records {
		require.Contains(t, ids, c.ID)
		require.Equal(t, cqrs.Cancellation{
			ID:            c.ID,
			CreatedAt:     c.CreatedAt,
			AccountID:     consts.DevServerAccountID,
			WorkspaceID:   consts.DevServerEnvID,
			AppID:         appID,
			FunctionID:    id,
			FunctionSlug:  "app-fn",
			StartedAfter:  new(from.UTC()),
			StartedBefore: until.UTC(),
			If:            new("event.data.cancel == true"),
			Name:          new("cleanup"),
			Kind:          enums.CancellationKindBulkRun,
			Type:          enums.CancellationTypeAPI,
		}, c)
	}
	for _, tc := range []struct {
		name   string
		at     time.Time
		cancel bool
		match  bool
	}{
		{"paused start", from, true, true}, {"until inclusive", until, true, true}, {"filtered", from, false, false},
		{"too early", from.Add(-time.Millisecond), true, false}, {"too late", until.Add(time.Millisecond), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := checker.IsCancelled(ctx, consts.DevServerEnvID, id, ulid.MustNew(ulid.Timestamp(tc.at), nil), map[string]any{"data": map[string]any{"cancel": tc.cancel}})
			require.NoError(t, err)
			if tc.match {
				require.NotNil(t, c)
				require.Contains(t, ids, c.ID)
			} else {
				require.Nil(t, c)
			}
		})
	}
	jobs, err := shard.ItemsByPartition(ctx, queue.Scope{
		IsSystem:  true,
		AccountID: consts.DevServerAccountID,
		EnvID:     consts.DevServerEnvID,
	}, queue.KindCancel, time.Time{}, time.Now().Add(time.Second))
	require.NoError(t, err)
	var queued []ulid.ULID
	for job := range jobs {
		require.Equal(t, queue.KindCancel, job.Data.Kind)
		require.Equal(t, id, job.Data.Identifier.WorkflowID)
		data, err := json.Marshal(job.Data.Payload)
		require.NoError(t, err)
		var c cqrs.Cancellation
		require.NoError(t, json.Unmarshal(data, &c))
		queued = append(queued, c.ID)
		require.Contains(t, records, c)
	}
	require.ElementsMatch(t, ids, queued)
	require.True(t, functions.pauses.partitionPaused(ctx, id).Paused)
}

func TestCreateCancellationUnknownFunction(t *testing.T) {
	functions := NewFunctionProvider(&fakeFunctionStore{app: &cqrs.App{
		ID:   uuid.New(),
		Name: "app",
	}})
	handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{
		Functions:     functions,
		Cancellations: NewCancellationProvider(nil),
	}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)
	for _, app := range []string{"app", "unknown"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/"+app+"/functions/unknown/cancellations", strings.NewReader(`{"from":"2026-01-01T00:00:00Z","until":"2026-01-01T01:00:00Z"}`)))
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"errors":[{"code":"not_found","message":"Function not found"}]}`, rec.Body.String())
	}
}
