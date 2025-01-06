package realtime

import (
	"fmt"

	"github.com/google/uuid"
)

// SubscriptionWS represents a websocket subscription
type SubscriptionWS struct {
	id uuid.UUID
	// TODO
}

func (s SubscriptionWS) ID() uuid.UUID {
	return s.id
}

func (s SubscriptionWS) WriteMessage(m Message) error {
	return fmt.Errorf("not implemented")
}

func (s SubscriptionWS) Protocol() string {
	return "ws"
}

func (s SubscriptionWS) SendKeepalive() error {
	// TODO
	return fmt.Errorf("not implemented")
}

func (s SubscriptionWS) Close() error {
	// TODO
	return fmt.Errorf("not implemented")
}
