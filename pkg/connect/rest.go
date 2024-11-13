package connect

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/publicerr"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

// showConnections retrieves the list of connections from the gateway state
func (c *connectGatewaySvc) showConnections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsID := uuid.New()
	conns, err := c.stateManager.GetConnections(ctx, wsID, GetConnOpts{})
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
