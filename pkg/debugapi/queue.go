package debugapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
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
		_ = publicerr.WriteHTTP(
			w,
			publicerr.Wrapf(err, http.StatusBadRequest, "error finding shard: %s", err.Error()),
		)
		return
	}

	fmt.Println("RETRIEVE PARTITION")
	qp, sqp, err := a.Queue.PartitionByID(ctx, shard, partitionID)
	if err != nil {
		_ = publicerr.WriteHTTP(
			w,
			publicerr.Wrapf(err, http.StatusBadRequest, "error retrieving partition: %s", err.Error()),
		)
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
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, http.StatusBadRequest, "error marshaling response"))
		return
	}

	w.Write(byt)
}
