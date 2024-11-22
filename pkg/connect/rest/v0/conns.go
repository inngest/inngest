package v0

import (
	"encoding/json"
	"net/http"

	connpb "github.com/inngest/inngest/proto/gen/connect/v1"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/rest"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/publicerr"
)

// showConnections retrieves the list of connections from the gateway state
//
// Provides query params to further filter the returned data
//   - app_id
func (c *router) showConnections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		envID uuid.UUID
		appID *uuid.UUID
	)
	switch c.Dev {
	case true:
		envID = consts.DevServerEnvId

	case false:
		// Expect UUID
		param := chi.URLParam(r, "envID")
		id, err := uuid.Parse(param)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Error{
				Err:     err,
				Message: "invalid environment ID",
				Data: map[string]any{
					"envID": param,
				},
				Status: http.StatusBadRequest,
			})
			return
		}
		envID = id
	}

	// Check appID query param
	if param := r.URL.Query().Get("app_id"); param != "" {

		id, err := uuid.Parse(param)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Error{
				Err:     err,
				Message: "app_id is invalid UUID",
				Data:    map[string]any{"app_id": param},
				Status:  http.StatusBadRequest,
			})
			return
		}
		appID = &id
	}

	var (
		conns []*connpb.ConnMetadata
		err   error
	)
	switch {
	case appID != nil:
		if conns, err = c.ConnectManager.GetConnectionsByAppID(ctx, envID, *appID); err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Error{
				Err:     err,
				Message: err.Error(),
				Status:  http.StatusInternalServerError,
			})
			return
		}
	default:
		if conns, err = c.ConnectManager.GetConnectionsByEnvID(ctx, envID); err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Error{
				Err:     err,
				Message: err.Error(),
				Status:  http.StatusInternalServerError,
			})
			return
		}
	}

	// Respond
	if len(conns) == 0 {
		conns = []*connpb.ConnMetadata{}
	}

	reply := &rest.ShowConnsReply{
		Data: conns,
	}

	resp, err := json.Marshal(reply)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	_, _ = w.Write(resp)
}
