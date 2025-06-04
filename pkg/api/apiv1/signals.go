package apiv1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/util"
)

type ReceiveSignalRequest struct {
	Signal string          `json:"signal"`
	Data   json.RawMessage `json:"data"`
}

type ReceiveSignalResponse struct {
	RunID string `json:"run_id"`
}

func (a router) receiveSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	data := ReceiveSignalRequest{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid signal data"))
		return
	}

	signalRes, err := util.WithRetry(
		ctx,
		"apiv1.receiveSignal",
		func(ctx context.Context) (*execution.ResumeSignalResult, error) {
			return a.opts.Executor.ResumeSignal(ctx, auth.WorkspaceID(), data.Signal, data.Data)
		},
		util.NewRetryConf(),
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Failed to receive signal"))
		return
	}

	if signalRes == nil || !signalRes.MatchedSignal {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "No signal found"))
		return
	}

	res := ReceiveSignalResponse{
		RunID: signalRes.RunID.String(),
	}

	_ = WriteResponse(w, res)
}
