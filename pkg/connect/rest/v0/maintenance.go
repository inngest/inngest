package v0

import (
	"encoding/json"
	"github.com/inngest/inngest/pkg/publicerr"
	"net/http"
)

func (c *router) getGatewayState(w http.ResponseWriter, _ *http.Request) {
	state, err := c.GatewayMaintenance.GetState()
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	data, err := json.Marshal(state)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	_, _ = w.Write(data)
}

func (c *router) drainGateway(w http.ResponseWriter, _ *http.Request) {
	err := c.GatewayMaintenance.DrainGateway()
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	_, _ = w.Write([]byte("{\"status\":\"ok\"}"))
}

func (c *router) activateGateway(w http.ResponseWriter, _ *http.Request) {
	err := c.GatewayMaintenance.ActivateGateway()
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Err:     err,
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	_, _ = w.Write([]byte("{\"status\":\"ok\"}"))
}
