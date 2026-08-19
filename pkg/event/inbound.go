package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
)

// ErrInternalEventName means an event name uses Inngest's reserved prefix.
var ErrInternalEventName = errors.New("event name is reserved for internal use")

// NormalizeInbound removes internal data, fills defaults, and validates an inbound event.
func NormalizeInbound(ctx context.Context, evt *Event, receivedAt time.Time) (SessionsMetrics, error) {
	if evt.IsInternal() {
		return SessionsMetrics{}, fmt.Errorf("%w: %s", ErrInternalEventName, evt.Name)
	}

	delete(evt.Data, consts.InngestEventDataPrefix)
	if evt.Timestamp == 0 {
		evt.Timestamp = receivedAt.UnixMilli()
	}
	if evt.User == nil {
		evt.User = map[string]any{}
	}

	sessionsMetrics := evt.Meta.ResolveSessions()
	if err := evt.Validate(ctx); err != nil {
		return sessionsMetrics, err
	}

	return sessionsMetrics, nil
}
