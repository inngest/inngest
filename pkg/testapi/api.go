package testapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/logger"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type TestAPI struct {
	chi.Router

	options Options
}

type Options struct {
	QueueShards        queue.ShardRegistry
	Queue              queue.Queue
	Executor           execution.Executor
	StateManager       statev2.RunService
	ResetAll           func()
	PauseFunction      func(id uuid.UUID)
	UnpauseFunction    func(id uuid.UUID)
	Hub                *Hub
}

func ShouldEnable() bool {
	return util.InTestMode()
}

func New(o Options) http.Handler {
	test := TestAPI{
		Router:  chi.NewRouter(),
		options: o,
	}

	test.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
		_, _ = writer.Write([]byte("OK"))
	})

	test.Post("/function/pause", test.PauseFunction)
	test.Post("/function/runs/cancel", test.CancelFunctionRun)

	test.Get("/queue/function-queue-size", test.GetQueueSize)

	test.Post("/reset", test.Reset)

	test.Get("/events", test.streamEvents)

	return test
}

func (t *TestAPI) Reset(w http.ResponseWriter, r *http.Request) {
	t.options.ResetAll()
	logger.StdlibLogger(r.Context()).Info("Reset all data stores")
	_, _ = w.Write([]byte("ok"))
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

	shard, err := t.options.QueueShards.Resolve(ctx, parsedAccountId, nil)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	count, err := shard.PartitionSize(ctx, parsedFnId.String(), time.Now().Add(2*365*24*time.Hour))
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

func (t *TestAPI) PauseFunction(w http.ResponseWriter, r *http.Request) {
	fnId := r.FormValue("fnId")

	parsedFnId, err := uuid.Parse(fnId)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Invalid fnId"))
		return
	}

	t.options.PauseFunction(parsedFnId)

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

	t.options.PauseFunction(parsedFnId)

	md, err := t.options.StateManager.LoadMetadata(ctx, statev2.ID{
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

	err = t.options.Executor.Cancel(ctx, md.ID, execution.CancelRequest{})
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal server error"))
	}

	w.WriteHeader(200)
	_, _ = w.Write([]byte("OK"))
}
