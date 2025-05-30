package testapi

import (
	"encoding/json"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/redis/rueidis"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type TestAPI struct {
	chi.Router

	QueueShardSelector redis_state.ShardSelector
	Queue              queue.Queue
	Executor           execution.Executor
	StateManager       statev2.RunService
}

type Options struct {
	QueueShardSelector redis_state.ShardSelector
	Queue              queue.Queue
	Executor           execution.Executor
	StateManager       statev2.RunService
}

func ShouldEnable() bool {
	return util.InTestMode()
}

func New(o Options) http.Handler {
	test := TestAPI{
		Router:             chi.NewRouter(),
		QueueShardSelector: o.QueueShardSelector,
		Queue:              o.Queue,
		Executor:           o.Executor,
		StateManager:       o.StateManager,
	}

	test.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
		_, _ = writer.Write([]byte("OK"))
	})

	test.Post("/function/pause", test.PauseFunction)
	test.Post("/function/runs/cancel", test.CancelFunctionRun)

	test.Get("/queue/function-queue-size", test.GetQueueSize)

	test.Get("/queue/active-counter", test.GetQueueActiveCounter)

	return test
}

func (t *TestAPI) GetQueueSize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountId := r.FormValue("accountId")
	fnId := r.FormValue("fnId")

	parsedAccountId, err := uuid.Parse(accountId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid accountId"))
		return
	}

	parsedFnId, err := uuid.Parse(fnId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid fnId"))
		return
	}

	shard, err := t.QueueShardSelector(ctx, parsedAccountId, nil)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	rc := shard.RedisClient.Client()

	count, err := rc.Do(ctx, rc.B().Zcard().Key(shard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, parsedFnId.String(), "")).Build()).ToInt64()
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	marshaled, err := json.Marshal(map[string]any{
		"count": count,
	})
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	_, _ = w.Write(marshaled)
}

type TestActiveCounters struct {
	ActiveAccount      int `json:"activeAccount"`
	ActiveFunction     int `json:"activeFunction"`
	ActiveRunsAccount  int `json:"activeRunsAccount"`
	ActiveRunsFunction int `json:"activeRunsFunction"`
}

func (t *TestAPI) GetQueueActiveCounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountId := r.FormValue("accountId")
	fnId := r.FormValue("fnId")

	parsedAccountId, err := uuid.Parse(accountId)
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to parse account ID", "err", err)
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid accountId"))
		return
	}

	parsedFnId, err := uuid.Parse(fnId)
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to parse fn ID", "err", err)
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid fnId"))
		return
	}

	shard, err := t.QueueShardSelector(ctx, parsedAccountId, nil)
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to select queue shard", "err", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	rc := shard.RedisClient.Client()

	activeAccount, err := rc.Do(ctx, rc.B().Get().Key(shard.RedisClient.KeyGenerator().ActiveCounter("account", parsedAccountId.String())).Build()).AsInt64()
	if err != nil && !rueidis.IsRedisNil(err) {
		logger.StdlibLogger(ctx).Error("failed to retrieve active count for account", "err", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	activePartition, err := rc.Do(ctx, rc.B().Get().Key(shard.RedisClient.KeyGenerator().ActiveCounter("p", parsedFnId.String())).Build()).AsInt64()
	if err != nil && !rueidis.IsRedisNil(err) {
		logger.StdlibLogger(ctx).Error("failed to retrieve active count for function", "err", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	activeRunsAccount, err := rc.Do(ctx, rc.B().Get().Key(shard.RedisClient.KeyGenerator().ActiveRunsCounter("account", parsedAccountId.String())).Build()).AsInt64()
	if err != nil && !rueidis.IsRedisNil(err) {
		logger.StdlibLogger(ctx).Error("failed to retrieve active run count for account", "err", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	activeRunsPartition, err := rc.Do(ctx, rc.B().Scard().Key(shard.RedisClient.KeyGenerator().ActivePartitionRunsIndex(parsedFnId.String())).Build()).AsInt64()
	if err != nil && !rueidis.IsRedisNil(err) {
		logger.StdlibLogger(ctx).Error("failed to retrieve active run count for function", "err", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	marshaled, err := json.Marshal(TestActiveCounters{
		ActiveAccount:      int(activeAccount),
		ActiveFunction:     int(activePartition),
		ActiveRunsAccount:  int(activeRunsAccount),
		ActiveRunsFunction: int(activeRunsPartition),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to marshal active counters response", "err", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	_, _ = w.Write(marshaled)
}

func (t *TestAPI) PauseFunction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountId := r.FormValue("accountId")
	fnId := r.FormValue("fnId")

	parsedAccountId, err := uuid.Parse(accountId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid accountId"))
		return
	}

	parsedFnId, err := uuid.Parse(fnId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid fnId"))
		return
	}

	err = t.Queue.SetFunctionPaused(ctx, parsedAccountId, parsedFnId, true)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
	}

	w.WriteHeader(200)
	_, _ = w.Write([]byte("OK"))
}

func (t *TestAPI) CancelFunctionRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountId := r.FormValue("accountId")
	fnId := r.FormValue("fnId")
	runId := r.FormValue("runId")

	parsedAccountId, err := uuid.Parse(accountId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid accountId"))
		return
	}

	parsedFnId, err := uuid.Parse(fnId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid fnId"))
		return
	}

	parsedRunId, err := ulid.Parse(runId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid runId"))
		return
	}

	err = t.Queue.SetFunctionPaused(ctx, parsedAccountId, parsedFnId, true)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
	}

	md, err := t.StateManager.LoadMetadata(ctx, statev2.ID{
		RunID:      parsedRunId,
		FunctionID: parsedFnId,
		Tenant: statev2.Tenant{
			AccountID: parsedAccountId,
		},
	})
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
	}

	err = t.Executor.Cancel(ctx, md.ID, execution.CancelRequest{})
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
	}

	w.WriteHeader(200)
	_, _ = w.Write([]byte("OK"))
}
