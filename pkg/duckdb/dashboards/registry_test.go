package dashboards

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestGetBoundsConcurrency covers Get's slot limit: w/ every slot taken, a
// call gives up once its deadline passes instead of queueing forever.
func TestGetBoundsConcurrency(t *testing.T) {
	origTimeout := getTimeout
	t.Cleanup(func() { getTimeout = origTimeout })
	getTimeout = 50 * time.Millisecond

	for range maxConcurrentGets {
		getSlots <- struct{}{}
	}
	t.Cleanup(func() {
		for range maxConcurrentGets {
			<-getSlots
		}
	})

	start := time.Now()
	// The db is never reached: Get times out waiting for a slot first.
	_, err := Get(context.Background(), nil, Scope{AccountID: uuid.New(), EnvID: uuid.New()}, KeySessionKeys)
	require.ErrorContains(t, err, "too many dashboard queries")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
}
