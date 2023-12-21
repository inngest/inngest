package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewGateway generates a new HTTP handler for a gateway
func NewGateway(o Opts) chi.Router {
	r := &Gateway{
		Router: chi.NewMux(),
		opts:   o,
	}
	r.setup()

	return r
}

// Opts represents the options for gateway API.
type Opts struct{}

type Gateway struct {
	chi.Router
	opts Opts
}

func (gw *Gateway) setup() {
	gw.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		r.Get("/health", gw.HealthCheck)
	})
}

func (gw *Gateway) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte("{ \"result\": \"ok\" }"))
}
