package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/inngest/inngest/pkg/publicerr"
)

type ReceiveSignalRequest struct {
	Signal string          `json:"signal"`
	Data   json.RawMessage `json:"data"`
}

type ReceiveSignalResponse struct {
	MatchedSignal bool   `json:"matched_signal"`
	RunID         string `json:"run_id"`
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
		spew.Dump(err.Error())
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid signal data"))
		return
	}

	signalRes, err := a.opts.Executor.ReceiveSignal(ctx, auth.WorkspaceID(), data.Signal, data.Data)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Failed to receive signal"))
		return
	}

	if signalRes == nil || !signalRes.MatchedSignal {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "No signal found"))
		return
	}

	res := ReceiveSignalResponse{
		MatchedSignal: signalRes.MatchedSignal,
		RunID:         signalRes.RunID.String(),
	}

	_ = WriteResponse(w, res)
}
