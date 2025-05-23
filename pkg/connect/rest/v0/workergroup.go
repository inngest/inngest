package connectv0

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/rest"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/publicerr"
	"net/http"
)

func (cr *connectApiRouter) showWorkerGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var envID uuid.UUID
	switch cr.Dev {
	case true:
		envID = consts.DevServerEnvID

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

	group, err := cr.GroupManager.GetWorkerGroupByHash(ctx, envID, groupID)
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

	conns, err := cr.ConnectManager.GetConnectionsByGroupID(ctx, envID, groupID)
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

	reply := rest.ShowWorkerGroupReply{
		Data: &rest.WorkerGroup{
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
