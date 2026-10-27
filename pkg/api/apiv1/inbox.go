package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/execution/inbox"
	"github.com/inngest/inngest/pkg/publicerr"
)

type AppendInboxRequest struct {
	Data json.RawMessage `json:"data"`
}

type InboxResponse struct {
	Seq   int64 `json:"seq"`
	Head  int64 `json:"head"`
	Acked int64 `json:"acked"`
	// Delivered is true when the message went straight to a waiting
	// step.subscribe.
	Delivered bool `json:"delivered"`
}

// appendInbox queues a message for an inbox key and delivers it to a waiting
// step.subscribe if there is one.
func (a router) appendInbox(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(nil, 400, "Missing inbox key"))
		return
	}

	req := AppendInboxRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Data) == 0 {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid inbox message"))
		return
	}

	ws := auth.WorkspaceID()
	store := inbox.Default()
	seq := store.Append(ws, key, req.Data)

	delivered, err := inbox.Deliver(store, ws, key, func(signalID string, data json.RawMessage) (bool, error) {
		res, err := a.opts.Executor.ResumeSignal(ctx, ws, signalID, data)
		if err != nil || res == nil {
			return false, err
		}
		return res.MatchedSignal, nil
	})
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Failed to deliver inbox message"))
		return
	}

	_ = WriteResponse(w, InboxResponse{
		Seq:       seq,
		Head:      store.Head(ws, key),
		Acked:     store.Acked(ws, key),
		Delivered: delivered,
	})
}

// getInbox reports the head and acked cursors of an inbox key.
func (a router) getInbox(w http.ResponseWriter, r *http.Request) {
	auth, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	key := chi.URLParam(r, "key")
	ws := auth.WorkspaceID()
	store := inbox.Default()
	_ = WriteResponse(w, InboxResponse{Head: store.Head(ws, key), Acked: store.Acked(ws, key)})
}
