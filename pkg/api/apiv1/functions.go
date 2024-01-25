package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/publicerr"
)

// GetAppFunctions retrieves functions for a given app name, as defined in the SDK.
func (a api) GetAppFunctions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	fns, err := a.opts.FunctionReader.GetFunctionsByAppExternalID(
		ctx,
		auth.WorkspaceID(),
		chi.URLParam(r, "appName"),
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	_ = json.NewEncoder(w).Encode(fns)
}
