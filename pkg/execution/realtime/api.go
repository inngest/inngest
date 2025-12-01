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
	"github.com/golang-jwt/jwt/v5"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
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
		r.Get("/realtime/stream", a.GetSSE)
		r.Post("/realtime/token", a.PostCreateJWT)
	})

	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		// Note that we use use realtime auth middleware and check for publishing claims manually.
		// This also allows us to use the original auth middleware and use signing keys for publishing.
		r.Use(realtimeAuthMW(a.opts.JWTSecret, a.opts.AuthMiddleware))

		r.Post("/realtime/publish", a.PostPublish)
	})
}

func (a *api) PostCreateJWT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

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

func (a *api) GetSSE(w http.ResponseWriter, r *http.Request) {
}

func (a *api) GetWebsocketUpgrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// NOTE: Here we always use the realtime auth finder, which attempts to auth
	// realtime connections via single-use JWTs, falling back to other auth methods
	// as necessary.
	auth, err := realtimeAuth(ctx)
	if err != nil {
		w.Header().Set("content-type", "application/json")
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
		return
	}

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // We don't care about verifying the origin.
	})
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
	// This is only usable:
	// - Within an Inngest function
	// - Or if the JWT given has "publish" permissions via specific claims in the JWT.

	// If we have authed using JWT claims, ensure it has the publish claim.  This publishing JWT
	claims, err := realtimeAuth(r.Context())
	if err == nil && !claims.Publish {
		// We have claims, but not for publishing. Error out.
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
		return
	}
	if claims == nil {
		// We have no claims, so attempt to auth using API keys.
		auth, err := a.opts.AuthFinder(r.Context())
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
			return
		}

		// In  this case, we've authed using signing keys.  Create a set of JWT claims and assign
		// this to the request so that we can use standard claims when publishing streams.  This
		// gives us a single place to load account IDs and envs for the current request.
		claims = &JWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: auth.AccountID().String(),
			},
			Env:     auth.WorkspaceID(),
			Publish: true,
		}
		r = r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
	}

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
	msg.Kind = streamingtypes.MessageKindData
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

	msg.Kind = streamingtypes.MessageKindDataStreamStart
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
		msg.Kind = streamingtypes.MessageKindDataStreamEnd
		a.opts.Broadcaster.Publish(ctx, msg)
	}(msg)

	// Read the body in chunks, up to X size.
	for range consts.MaxStreamingChunks {
		buf := make([]byte, consts.StreamingChunkSize)
		n, err := r.Body.Read(buf)

		if n > 0 {
			// Spit this chunk out!
			a.opts.Broadcaster.PublishChunk(
				ctx,
				msg,
				streamingtypes.ChunkFromMessage(
					msg,
					string(buf[:n]),
				),
			)
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
	auth, err := realtimeAuth(r.Context())
	if err != nil || auth == nil || !auth.Publish {
		return Message{}, err
	}

	msg := Message{
		Channel:   r.URL.Query().Get("channel"),
		Topic:     r.URL.Query().Get("topic"),
		EnvID:     auth.Env,
		CreatedAt: time.Now(),
	}
	if runID := r.URL.Query().Get("run_id"); runID != "" {
		msg.RunID, _ = ulid.Parse(runID)
	}
	if metadata := r.URL.Query().Get("metadata"); metadata != "" {
		// Don't bother to JSON deocde, just pass this straight through.
		msg.Metadata = json.RawMessage(metadata)
	}

	return msg, nil
}
