package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/inngest/inngest/pkg/publicerr"
)

type SignalData struct {
	Data json.RawMessage `json:"data"`
}

func (a router) receiveSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	auth.WorkspaceID()

	signal := r.FormValue("signal")
	if signal == "" {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Missing signal query parameter"))
		return
	}

	data := SignalData{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		spew.Dump(err.Error())
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid signal data"))
		return
	}

	res, err := a.opts.Executor.ReceiveSignal(ctx, auth.WorkspaceID(), signal, data.Data)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Failed to receive signal"))
		return
	}

	// TODO Not raw res
	_ = WriteResponse(w, res)
}
