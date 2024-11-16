package connect

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/publicerr"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type ShowConnsReply struct {
	Data []*connpb.ConnMetadata `json:"data"`
}

// showConnections retrieves the list of connections from the gateway state
func (c *connectGatewaySvc) showConnectionsByEnv(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var envID uuid.UUID
	switch c.dev {
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

	conns, err := c.stateManager.GetConnectionsByEnvID(ctx, envID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	reply := &ShowConnsReply{
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

	reply := &ShowConnsReply{
		Data: conns,
	}

	resp, err := json.Marshal(reply)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(http.StatusInternalServerError, "error serializing response"))
		return
	}

	_, _ = w.Write(resp)
}

type workerGroup struct {
	state.WorkerGroup

	Synced bool     `json:"synced"`
	Conns  []string `json:"conns"`
}

type ShowWorkerGroupReply struct {
	Data *workerGroup `json:"data"`
}

func (c *connectGatewaySvc) showWorkerGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var envID uuid.UUID
	switch c.dev {
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

	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(http.StatusBadRequest, "missing groupID"))
		return
	}

	group, err := c.stateManager.GetWorkerGroupByHash(ctx, envID, groupID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: "error retrieving worker group",
			Data: map[string]any{
				"envID":   envID,
				"groupID": groupID,
			},
			Status: http.StatusNotFound,
		})
		return
	}

	conns, err := c.stateManager.GetConnectionsByGroupID(ctx, envID, groupID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(
			err,
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	}

	connIDs := make([]string, len(conns))
	for i, conn := range conns {
		connIDs[i] = conn.Id
	}

	reply := ShowWorkerGroupReply{
		Data: &workerGroup{
			WorkerGroup: *group,
			Synced:      group.SyncID != nil,
			Conns:       connIDs,
		},
	}

	resp, err := json.Marshal(reply)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(http.StatusInternalServerError, "error serializing response"))
		return
	}

	_, _ = w.Write(resp)
}
