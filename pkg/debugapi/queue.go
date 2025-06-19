package debugapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
)

func (a *debugAPI) partitionByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	partitionID := chi.URLParam(r, "id")

	// If the passed in value is not a valid UUID, it's then likely a system partition
	var queueName *string
	if _, err := uuid.Parse(partitionID); err != nil {
		queueName = &partitionID
	}

	fmt.Println("SHARD SELECTOR")
	shard, err := a.ShardSelector(ctx, consts.DevServerAccountID, queueName)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error"))
		return
	}

	fmt.Println("RETRIEVE PARTITION")
	qp, sqp, err := a.Queue.PartitionByID(ctx, shard, partitionID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		resp := DebugResponse{
			Error: fmt.Errorf("error retrieving partition: %w", err),
		}
		byt, err := json.Marshal(resp)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		w.Write(byt)
		return
	}

	partition := cqrs.QueuePartition{}
	if qp != nil {
		partition.ID = qp.ID
		partition.AccountID = qp.AccountID
		partition.EnvID = qp.EnvID
		partition.FunctionID = qp.FunctionID
	}

	if sqp != nil {
		partition.PauseRefill = sqp.PauseRefill
		partition.PauseEnqueue = sqp.PauseEnqueue
	}

	resp := DebugResponse{
		Data: partition,
	}
	fmt.Println("SUCCESS")
	byt, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(byt)
}
