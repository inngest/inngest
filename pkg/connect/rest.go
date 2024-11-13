package connect

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/publicerr"
)

// showConnections retrieves the list of connections from the gateway state
func (c *connectGatewaySvc) showConnections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsID := uuid.New()
	conns, err := c.stateManager.GetConnections(ctx, wsID, GetConnOpts{})
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	resp, err := json.Marshal(conns)
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	w.Write(resp)
}
