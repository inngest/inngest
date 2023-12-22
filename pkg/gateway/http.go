package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest/pkg/inngest/log"
)

func (gw *Gateway) Invoke(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	if err := handleHTTPInvoke(w, r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
		return
	}
}

func handleHTTPInvoke(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	req := httpdriver.Request{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	log.From(ctx).
		Info().
		Str("url", r.URL.String()).
		Msg("handling SDK gateeay request")

	resp, err := httpdriver.DoRequest(ctx, httpdriver.DefaultClient, req)
	if err != nil {
		log.From(r.Context()).Error().
			Err(err).
			Str("url", req.URL.String()).
			Msg("error handling sdk gateway request")
	}
	if resp != nil {
		log.From(r.Context()).Error().
			Err(err).
			Str("url", req.URL.String()).
			Msg("handled gateway request")

		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return fmt.Errorf("Error decoding SDK response: %w", err)
		}
		return nil
	}

	return fmt.Errorf("Error communcating with SDK: %w", err)
}
