package apiv1

import (
	"net/http"

	"github.com/inngest/inngest/pkg/publicerr"
)

func (a router) promScrape(w http.ResponseWriter, _ *http.Request) {
	_ = publicerr.WriteHTTP(w, publicerr.Errorf(
		http.StatusNotImplemented,
		"not implemented",
	))
}
