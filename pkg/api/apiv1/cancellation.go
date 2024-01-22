package apiv1

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

func (a api) GetCancellations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	all, err := a.opts.CancellationReadWriter.Cancellations(ctx, auth.WorkspaceID())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Error listing cancellations"))
		return
	}

	_ = json.NewEncoder(w).Encode(all)

}

type CreateCancellationBody struct {
	// AppID is the client ID specified via the SDK in the app that defines the function.
	AppID string `json:"app_id"`
	// FunctionID is the function ID string specified in configuration via the SDK.
	FunctionID string     `json:"function_id"`
	From       *time.Time `json:"from"`
	To         time.Time  `json:"to"`
	If         *string    `json:"if,omitempty"`
}

func (a api) CreateCancellation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	opts := CreateCancellationBody{}
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid cancellation request"))
		return
	}

	fn, err := a.opts.FunctionReader.GetFunctionByExternalID(
		ctx,
		auth.WorkspaceID(),
		opts.AppID,
		opts.FunctionID,
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "Function not found"))
		return
	}

	// Create a new cancellation for the given function ID
	cancel := cqrs.Cancellation{
		ID:          ulid.MustNew(ulid.Now(), rand.Reader),
		WorkspaceID: auth.WorkspaceID(),
		FunctionID:  fn.ID,
		From:        opts.From,
		To:          opts.To,
		If:          opts.If,
	}
	if err := a.opts.CancellationReadWriter.CreateCancellation(ctx, cancel); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Error creating function"))
		return
	}

	_ = json.NewEncoder(w).Encode(cancel)
}
