package v0

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/rest"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/publicerr"
)

// showConnections retrieves the list of connections from the gateway state
func (c *router) showConnectionsByEnv(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var envID uuid.UUID
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

	conns, err := c.ConnectManager.GetConnectionsByEnvID(ctx, envID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
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

func (c *router) showConnectionsByApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var envID uuid.UUID
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

	var appID uuid.UUID
	{
		param := chi.URLParam(r, "appID")
		id, err := uuid.Parse(param)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Error{
				Err:     err,
				Message: "invalid app ID",
				Data:    map[string]any{"appID": param},
				Status:  http.StatusBadRequest,
			})
			return
		}
		appID = id
	}

	conns, err := c.ConnectManager.GetConnectionsByAppID(ctx, envID, appID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	reply := &rest.ShowConnsReply{
		Data: conns,
	}

	resp, err := json.Marshal(reply)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(http.StatusInternalServerError, "error serializing response"))
		return
	}

	_, _ = w.Write(resp)
}
