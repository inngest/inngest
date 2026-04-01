package constraintlifecycle

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
)

// Notifier is called when queue items hit constraints during processing.
type Notifier interface {
	// OnConstraintHit is called when a queue item hits a constraint.
	// itemMetadata is the queue item's metadata map used to extract span references.
	OnConstraintHit(ctx context.Context, constraint enums.QueueConstraint, itemMetadata map[string]any) error
}
