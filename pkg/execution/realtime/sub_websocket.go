package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
	"github.com/google/uuid"
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
func NewWebsocketSubscription(ctx context.Context, b Broadcaster, conn *websocket.Conn, topics []Topic) (Subscription, error) {
	sub := &SubscriptionWS{
		b:  b,
		id: uuid.New(),
		ws: conn,
	}

	// Handle reading of additional messages such as subscription requests from the WS
	go func() {
		if err := sub.poll(ctx); err != nil {
			logger.StdlibLogger(ctx).Warn(
				"error reading from rt ws conn",
				"error", err,
			)
		}
	}()

	if len(topics) > 0 {
		if err := b.Subscribe(ctx, sub, topics); err != nil {
			// TODO: Handle inability to subscribe.
		}
	}

	return sub, nil
}

// SubscriptionWS represents a websocket subscription
type SubscriptionWS struct {
	id uuid.UUID
	b  Broadcaster

	ws *websocket.Conn
}

func (s SubscriptionWS) ID() uuid.UUID {
	return s.id
}

func (s SubscriptionWS) Protocol() string {
	return "ws"
}

func (s SubscriptionWS) WriteMessage(m Message) error {
	byt, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.ws.Write(context.Background(), websocket.MessageText, byt)
}

func (s SubscriptionWS) SendKeepalive(m Message) error {
	// Ignore the keepalives and send a ping instead.
	return s.ws.Ping(context.Background())
}

func (s SubscriptionWS) Close() error {
	return s.ws.Close(websocket.CloseStatus(nil), string(MessageKindClosing))
}

func (s SubscriptionWS) poll(ctx context.Context) error {
	for {
		mt, byt, err := s.ws.Read(ctx)
		if err != nil {
			return err
		}

		if mt == websocket.MessageBinary {
			// We do not handle binary data in realtime connections.
			continue
		}

		// Unmarshal byt, handle subscribe and unsubscribe requests.
		msg := &Message{}
		if err := json.Unmarshal(byt, msg); err != nil {
			// Unknown message, ignore.
			logger.StdlibLogger(ctx).Warn(
				"unknown realtime ws message",
			)
			continue
		}

		switch msg.Kind {
		case MessageKindSubscribe:
			// Subscribe messages must always have a JWT as the data;
			// the JWT embeds the topics that will be subscribed to.
			jwt, ok := msg.Data.(string)
			if !ok {
				logger.StdlibLogger(ctx).Warn(
					"unknown subscribe jwt type",
					"type", fmt.Sprintf("%T", msg.Data),
				)
				continue
			}
			// TODO: Get token for topics.
			topics, err := TopicsFromJWT(ctx, []byte("TODO"), jwt)
			if err != nil {
				// TODO: Reply with unsuccessful subscribe msg
				continue
			}

			if err := s.b.Subscribe(ctx, s, topics); err != nil {
				// TODO: Reply with unsuccessful subscribe msg
				continue
			}

			// TODO: Reply with successful subscribe msg
			continue

		case MessageKindUnsubscribe:
			// TODO: Unsub from the given topics.
		}
	}
}
