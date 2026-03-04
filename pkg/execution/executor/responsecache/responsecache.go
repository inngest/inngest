package responsecache

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/state"
)

// ResponseCache stores and retrieves DriverResponse results keyed by a unique
// execution identifier.  This enables crash recovery: if the executor crashes
// after receiving an SDK response but before processing it, the response can
// be loaded from cache on retry instead of re-sending the request.
type ResponseCache interface {
	// Get retrieves a cached DriverResponse for the given key.
	// Returns (nil, nil) when no cached response exists.
	Get(ctx context.Context, key string) (*state.DriverResponse, error)

	// Set stores a DriverResponse under the given key.  The implementation
	// is responsible for automatic expiration / cleanup.
	Set(ctx context.Context, key string, resp *state.DriverResponse) error

	// Close shuts down the cache and any background goroutines.
	Close() error
}
