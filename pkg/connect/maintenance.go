package connect

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/publicerr"
	"net/http"
)

type GatewayMaintenanceActions interface {
	IsDraining() bool
	IsDrained() bool
	GetState() (*state.Gateway, error)
	DrainGateway() error
	ActivateGateway() error
}

type maintenanceApi struct {
	chi.Router
	GatewayMaintenance GatewayMaintenanceActions
}

func newMaintenanceApi(actions GatewayMaintenanceActions) *maintenanceApi {
	api := &maintenanceApi{
		Router:             chi.NewRouter(),
		GatewayMaintenance: actions,
	}
	api.setup()
	return api
}

func (m *maintenanceApi) setup() {
	m.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)
		r.Use(headers.ContentTypeJsonResponse())

		r.Get("/healthy", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("."))
		})
		r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
			if m.GatewayMaintenance.IsDraining() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		r.Get("/gateway", m.getGatewayState)

		r.Post("/drain", m.drainGateway)
		r.Get("/drained", func(w http.ResponseWriter, req *http.Request) {
			if m.GatewayMaintenance.IsDrained() {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusTooEarly)
		})

		r.Post("/activate", m.activateGateway)
	})
}

func (m *maintenanceApi) getGatewayState(w http.ResponseWriter, _ *http.Request) {
	state, err := m.GatewayMaintenance.GetState()
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

func (m *maintenanceApi) drainGateway(w http.ResponseWriter, _ *http.Request) {
	err := m.GatewayMaintenance.DrainGateway()
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

func (m *maintenanceApi) activateGateway(w http.ResponseWriter, _ *http.Request) {
	err := m.GatewayMaintenance.ActivateGateway()
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
