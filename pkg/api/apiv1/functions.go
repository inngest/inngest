package apiv1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
)

// GetAppFunctions retrieves functions for a given app name, as defined in the SDK.
func (a API) GetAppFunctions(ctx context.Context, appName string) ([]*cqrs.Function, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}

	fns, err := a.opts.FunctionReader.GetFunctionsByAppExternalID(
		ctx,
		auth.WorkspaceID(),
		appName,
	)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}

	return fns, nil
}

// GetAppFunctions is the route wrapper for the GetAppFunctions API handler.
func (a router) GetAppFunctions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fns, err := a.API.GetAppFunctions(ctx, chi.URLParam(r, "appName"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(fns)
}
