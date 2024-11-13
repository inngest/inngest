package connect

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/publicerr"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

// showConnections retrieves the list of connections from the gateway state
func (c *connectGatewaySvc) showConnectionsByEnv(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var wsID uuid.UUID
	{
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
		wsID = id
	}

	conns, err := c.stateManager.GetConnectionsByEnvID(ctx, wsID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	reply := &connpb.ShowConnsReply{
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

func (c *connectGatewaySvc) showConnectionsByApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	conns, err := c.stateManager.GetConnectionsByAppID(ctx, appID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	reply := &connpb.ShowConnsReply{
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
