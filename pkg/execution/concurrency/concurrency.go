package concurrency

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/queue"
)

var (
	ErrAtConcurrencyLimit = fmt.Errorf("at concurrency limit")
	ErrKeyNotFound        = fmt.Errorf("concurrency key not found")
)

// ConcurrencyService defines a concurrency service which tracks concurrency by function IDs.  As
// items are claimed.  This service lives outside of the state store and queue, allowing transactionality
// at a higher level.
type ConcurrencyService interface {
	// Add inserts a new item into the concurrency service.  This
	// will count towards concurrency limits until the specified timeout or until the key is
	// removed via a call to Done().
	//
	// Add must return an ErrAtConcurrencyLimit if there is no capacity for the given
	// function ID/key.
	Add(ctx context.Context, functionID uuid.UUID, qi queue.Item) error

	// Done removes the given key from concurrency limits for a function.
	//
	// It is expected that Done is manually called outside of the state store and queue; the
	// implementer must decide how to complete items and remove them from concurrency limits.
	Done(ctx context.Context, functionID uuid.UUID, qi queue.Item) error

	// Check returns whether there's capacity within concurrency limits for a given
	// function ID.
	//
	// When processing items it's safer to call Add, which should atomically check concurrency
	// limits and only add the item if there's capacity.  This is not intended for use when
	// adding items to the concurrency service, otherwise race conditions may cause us to exceed
	// concurrency limits.
	Check(ctx context.Context, functionID uuid.UUID, limit int) error
}
