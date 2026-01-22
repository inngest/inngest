package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/inngest/inngest/pkg/logger"
)

// NewWebsocketSubscription handles creating a new websocket subscription for a given
// http request.
//
// This requires a broadcaster, as the connection handles its own subscribe/unsubscribe
// flows to the broadcaster via incoming messages.
//
// The flow is as follows:
//
//   - An HTTP request is made to a realtime API, either with a JWT or a signing key as authentication
//   - The realtime API authenticates the incoming request and upgrades the connection to a websocket
//   - The API handler calls this function to instantiate a new Subscription, with any pre-registered
//     topics.
//   - The websocket subscriber listens for incoming messages which can subscribe and unsubscribe from
//     new topics at will (given a valid JWT in the websocket message, for subscription requests)
//
// This implements ReadWriteSubscription
func NewWebsocketSubscription(
	ctx context.Context,
	b Broadcaster,
	acctID, envID uuid.UUID,
	jwtSigningKey []byte,
	conn *websocket.Conn,
	topics []Topic,
) (*SubscriptionWS, error) {
	sub := &SubscriptionWS{
		b:             b,
		acctID:        acctID,
		envID:         envID,
		id:            uuid.New(),
		ws:            conn,
		jwtSigningKey: jwtSigningKey,
	}

	if b == nil {
		return nil, fmt.Errorf("Cannot make websocket connection without broadcaster")
	}
	err := b.Subscribe(ctx, sub, topics)
	return sub, err
}

// SubscriptionWS represents a websocket subscription
type SubscriptionWS struct {
	id uuid.UUID

	// acctID represents the authenticated account ID when initializing
	// the websocket connection
	acctID uuid.UUID
	// acctID represents the authenticated environment ID when initializing
	// the websocket connection
	envID uuid.UUID

	b Broadcaster

	jwtSigningKey []byte

	ws *websocket.Conn
}

func (s SubscriptionWS) ID() uuid.UUID {
	return s.id
}

func (s SubscriptionWS) Protocol() string {
	return "ws"
}

func (s SubscriptionWS) Write(b []byte) error {
	return s.ws.Write(context.Background(), websocket.MessageText, b)
}

func (s SubscriptionWS) WriteMessage(m Message) error {
	// Ensure that the data is valid JSON.  NOte that sometimes
	// m.Data is set as a raw string - eg. the channel ID.
	if !json.Valid(m.Data) {
		enc, err := json.Marshal(string(m.Data))
		if err != nil {
			return err
		}
		m.Data = enc
	}

	byt, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.ws.Write(context.Background(), websocket.MessageText, byt)
}

func (s SubscriptionWS) WriteChunk(c Chunk) error {
	byt, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.ws.Write(
		context.Background(),
		websocket.MessageText,
		byt,
	)
}

func (s SubscriptionWS) SendKeepalive(m Message) error {
	// Ignore the keepalives and send a ping instead.
	return s.ws.Ping(context.Background())
}

func (s SubscriptionWS) Close() error {
	return s.ws.Close(websocket.CloseStatus(nil), string(streamingtypes.MessageKindClosing))
}

func (s SubscriptionWS) Poll(ctx context.Context) error {
	for {
		mt, byt, err := s.ws.Read(ctx)
		if err != nil {
			status := websocket.CloseStatus(err)
			if status >= websocket.StatusNormalClosure && status <= websocket.StatusNoStatusRcvd {
				// safe to ignore tehse.
				return nil
			}
			return err
		}

		if mt == websocket.MessageBinary {
			// We do not handle binary data in realtime connections.
			continue
		}

		// Unmarshal byt, handle subscribe and unsubscribe requests.
		msg := &Message{}
		if err := json.Unmarshal(byt, msg); err != nil {
			// unknown message, ignore...
			logger.StdlibLogger(ctx).Warn("unknown realtime ws message")
			continue
		}

		switch msg.Kind {
		case streamingtypes.MessageKindSubscribe:
			// Subscribe messages must always have a JWT as the data;
			// the JWT embeds the topics that will be subscribed to.
			var jwt string
			if err := json.Unmarshal(msg.Data, &jwt); err != nil {
				logger.StdlibLogger(ctx).Warn(
					"unknown subscribe ws data type",
					"type", fmt.Sprintf("%T", msg.Data),
				)
				continue
			}

			topics, err := TopicsFromJWT(ctx, s.jwtSigningKey, jwt)
			if err != nil {
				// XXX: Reply with unsuccessful subscribe msg
				continue
			}

			if err := s.b.Subscribe(ctx, s, topics); err != nil {
				// XXX: Reply with unsuccessful subscribe msg
				continue
			}

			// XXX: Reply with successful subscribe msg
			continue
		case streamingtypes.MessageKindUnsubscribe:
			// Unsub from the given topics.  Assume that the unsubscribe data
			// is a list of topics.
			topics := []Topic{}
			if err := json.Unmarshal(msg.Data, &topics); err != nil {
				logger.StdlibLogger(ctx).Warn(
					"error unmarshalling unsubscribe data",
					"error", err,
				)
				continue
			}

			if err := s.b.Unsubscribe(ctx, s.id, topics); err != nil {
				// XXX: reply with error.
				continue
			}
		}
	}
}
