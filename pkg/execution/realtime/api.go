package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/util"
)

type APIOpts struct {
	JWTSecret []byte
	// Broadcaster allows connections to subscribe to topics, picking up events from
	// the system and forwarding them on.
	Broadcaster Broadcaster
	// AuthMiddleware authenticates the incoming API request.
	AuthMiddleware func(http.Handler) http.Handler
	// AuthFinder authenticates the given request, returning the env and account IDs.
	AuthFinder apiv1auth.AuthFinder
}

func NewAPI(o APIOpts) http.Handler {
	if o.AuthFinder == nil {
		o.AuthFinder = apiv1auth.NilAuthFinder
	}

	// Create the HTTP implementation, which wraps the handler.  We do ths to code
	// share and split the HTTP concerns from the actual logic, eg. to share to GQL.
	impl := &api{
		Router: chi.NewRouter(),
		opts:   o,
	}

	impl.setup()

	return impl
}

type api struct {
	chi.Router
	opts APIOpts
}

func (a *api) setup() {
	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		// NOTE: We always use the realtime auth middleware which wraps the standard
		// auth middleware with JWT validation
		r.Use(realtimeAuthMW(a.opts.JWTSecret, a.opts.AuthMiddleware))

		r.Get("/realtime/connect", a.GetWebsocketUpgrade)
		r.Post("/realtime/token", a.PostCreateJWT)
	})

	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		// This can ONLY use the AuthMiddleware as we MUST use a signing key for
		// authentication for publishing to streams.
		if a.opts.AuthMiddleware != nil {
			r.Use(a.opts.AuthMiddleware)
		}

		r.Post("/realtime/publish", a.PostPublish)
	})
}

func (a *api) PostCreateJWT(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("content-type", "application/json")

	// This only uses the given auth finder, which does not accept JWT claims.
	auth, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
		return
	}

	// We expect the user to post a list of topics that they're interested in.
	topics := []Topic{}
	if err := json.NewDecoder(r.Body).Decode(&topics); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid request: must provide a list of topics"))
		return
	}

	// Set the env ID from the authentication context.
	for n := range topics {
		topics[n].EnvID = auth.WorkspaceID()
	}

	jwt, err := NewJWT(
		r.Context(),
		a.opts.JWTSecret,
		auth.AccountID(),
		auth.WorkspaceID(),
		topics,
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Error creating JWT.  Please try again"))
		return
	}

	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"jwt": jwt,
	})
}

func (a *api) GetWebsocketUpgrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// NOTE: Here we always use the realtime auth finder, which attempts to auth
	// realtime connections via single-use JWTs, falling back to other auth methods
	// as necessary.
	auth, err := realtimeAuth(ctx)
	if err != nil {
		w.Header().Add("content-type", "application/json")
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
		return
	}

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		w.WriteHeader(400)
		logger.StdlibLogger(ctx).Error("error upgrading ws connection", "error", err)
		// NOTE: This already responds with an error.
		return
	}

	logger.StdlibLogger(ctx).Info(
		"new realtime connection",
		"acct_id", auth.AccountID(),
		"env_id", auth.Env,
		"topics", auth.Topics,
	)

	sub, err := NewWebsocketSubscription(
		ctx,
		a.opts.Broadcaster,
		auth.AccountID(),
		auth.WorkspaceID(),
		a.opts.JWTSecret,
		ws,
		auth.Topics,
	)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error creating websocket subscription", "error", err)
		ws.Close(websocket.StatusAbnormalClosure, "error subscribing to topics")
		return
	}

	// Handle reading of additional messages such as subscription requests from the WS
	pollCtx := context.Background()
	if err := sub.Poll(pollCtx); err != nil {
		logger.StdlibLogger(ctx).Warn(
			"error reading from rt ws conn",
			"error", err,
		)
	}

	_ = ws.CloseNow()
}

func (a *api) PostPublish(w http.ResponseWriter, r *http.Request) {
	// Allow publishing of arbitrary data using the environment signing
	// key as the auth token.
	//
	// This is only usable within an Inngest function.

	// NOTE: If the content type is of "text/stream", this creates a new
	// stream to buffer messages to subscribers in 1KB chunks
	if r.Header.Get("content-type") == "text/stream" {
		// Read body in goroutine, publishing a stream to subscribers
		// until the message is done.
		a.publishStream(w, r)
		return
	}

	msg, err := a.getStreamMessage(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// This msg is arbitrary data.
	msg.Kind = MessageKindData
	// Read all data from the request body.
	byt, err := io.ReadAll(io.LimitReader(r.Body, consts.MaxStreamingMessageSizeBytes))
	_ = r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Is byt valid JSON?  If so, we don't want to double-encode it.
	if json.Valid(byt) {
		msg.Data = byt
	} else {
		msg.Data, err = json.Marshal(string(byt))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err := msg.Validate(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	a.opts.Broadcaster.Publish(r.Context(), msg)
}

// publishStream handles publishing a stream of data sent to Inngest over seconds
// to minutes.
func (a *api) publishStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	msg, err := a.getStreamMessage(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	msg.Kind = MessageKindDataStreamStart
	// We must create a new random stream ID for the data stream, allowing
	// all published chunks to be associated with each other.
	sID := util.XXHash(time.Now())
	msg.Data = []byte(sID)

	if err := msg.Validate(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Publish the stream start message
	a.opts.Broadcaster.Publish(ctx, msg)

	// And always publish a stream end.
	defer r.Body.Close()
	defer func(msg Message) {
		msg.Kind = MessageKindDataStreamEnd
		a.opts.Broadcaster.Publish(ctx, msg)
	}(msg)

	// Read the body in chunks, up to  chui
	for i := 0; i < consts.MaxStreamingChunks; i++ {
		buf := make([]byte, consts.StreamingChunkSize)
		n, err := r.Body.Read(buf)

		if n > 0 {
			// Spit this chunk out!
			a.opts.Broadcaster.PublishStream(ctx, msg, string(buf[:n]))
		}

		if errors.Is(err, io.EOF) {
			// Read it all
			break
		}
		if err != nil {
			// Some other error; log and respond with an error message.
			logger.StdlibLogger(ctx).Warn(
				"error reading streaming publish",
				"error", err,
				"stream_message", msg,
			)
			return
		}
	}
}

func (a *api) getStreamMessage(r *http.Request) (Message, error) {
	auth, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		return Message{}, err
	}

	msg := Message{
		Channel:    r.URL.Query().Get("channel"),
		TopicNames: []string{r.URL.Query().Get("topic")},
		EnvID:      auth.WorkspaceID(),
		CreatedAt:  time.Now(),
	}
	return msg, nil
}
