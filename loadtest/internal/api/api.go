// Package api exposes the REST surface the UI talks to. JSON only; no SSE,
// no GraphQL. The UI polls /runs/:id/live at 1–2s cadence for updates.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/runner"
	"github.com/inngest/inngest/loadtest/internal/shapes"
	"github.com/inngest/inngest/loadtest/internal/storage"
)

// Handler wires the REST routes onto an http.ServeMux.
type Handler struct {
	store   *storage.Store
	manager *runner.Manager
	hostID  string
}

// New constructs a Handler. hostID is baked into each run row for later
// multi-host attribution.
func New(store *storage.Store, mgr *runner.Manager, hostID string) *Handler {
	return &Handler{store: store, manager: mgr, hostID: hostID}
}

// Mount registers the API routes at /api/* on the given mux.
func (h *Handler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/defaults", h.getDefaults)
	mux.HandleFunc("GET /api/shapes", h.getShapes)
	mux.HandleFunc("POST /api/runs", h.postRun)
	mux.HandleFunc("GET /api/runs", h.listRuns)
	mux.HandleFunc("GET /api/runs/{id}", h.getRun)
	mux.HandleFunc("POST /api/runs/{id}/stop", h.stopRun)
	mux.HandleFunc("GET /api/runs/{id}/live", h.liveSamples)
	mux.HandleFunc("GET /api/runs/{id}/aggregates", h.aggregates)
	mux.HandleFunc("GET /api/runs/compare", h.compareRuns)
}

func (h *Handler) getDefaults(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, config.Defaults())
}

func (h *Handler) getShapes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"shapes": shapes.All()})
}

func (h *Handler) postRun(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}
	cfg, err := config.LoadJSON(body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}
	id, err := h.manager.StartRun(cfg, h.hostID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	runs, err := h.store.ListRuns(limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row, err := h.store.GetRun(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) stopRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.manager.StopRun(id) {
		writeErr(w, http.StatusNotFound, "run not active")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

func (h *Handler) liveSamples(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	after := int64(0)
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			after = n
		}
	}
	limit := 500
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			limit = n
		}
	}
	samples, err := h.store.ReadLiveSamples(id, after, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var maxTs int64 = after
	for _, s := range samples {
		if s.TSNanos > maxTs {
			maxTs = s.TSNanos
		}
	}
	// Merge in-memory counters (present while the run is active) with the
	// persisted summary (present after the run ends). If neither applies the
	// stats read as zero.
	var stats any
	if live := h.manager.LiveStats(id); live != nil {
		stats = live
	} else if row, err := h.store.GetRun(id); err == nil && row.Summary != nil {
		stats = row.Summary
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"samples": samples,
		"cursor":  maxTs,
		"stats":   stats,
	})
}

func (h *Handler) aggregates(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aggs, err := h.store.ReadAggregates(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, aggs)
}

func (h *Handler) compareRuns(w http.ResponseWriter, r *http.Request) {
	a := r.URL.Query().Get("a")
	b := r.URL.Query().Get("b")
	if a == "" || b == "" {
		writeErr(w, http.StatusBadRequest, "a and b query params required")
		return
	}
	aa, err := h.store.ReadAggregates(a)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	bb, err := h.store.ReadAggregates(b)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"a": aa, "b": bb})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
