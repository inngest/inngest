package loader

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadByUUID_DedupesUniqueKeysAndPreservesOrder pins the per-row dedup
// behavior that #4326 relies on: a list view rendering N runs that share
// fewer than N unique app/function IDs must invoke the underlying fetch
// once per unique ID, not once per row, and the loader must return results
// in the order of the input keys.
func TestLoadByUUID_DedupesUniqueKeysAndPreservesOrder(t *testing.T) {
	ctx := context.Background()

	idA := uuid.New()
	idB := uuid.New()
	idC := uuid.New()

	// Three unique IDs delivered as five keys with duplicates: A, B, A, C, B.
	keys := dataloader.NewKeysFromStrings([]string{
		idA.String(),
		idB.String(),
		idA.String(),
		idC.String(),
		idB.String(),
	})

	var calls atomic.Int32
	fetch := func(_ context.Context, id uuid.UUID) (*cqrs.App, error) {
		calls.Add(1)
		return &cqrs.App{ID: id}, nil
	}

	results := loadByUUID(ctx, keys, fetch)

	// Order preserved.
	require.Len(t, results, 5)
	expectedOrder := []uuid.UUID{idA, idB, idA, idC, idB}
	for i, r := range results {
		require.NoError(t, r.Error, "result %d unexpectedly errored", i)
		app, ok := r.Data.(*cqrs.App)
		require.True(t, ok, "result %d not a *cqrs.App", i)
		assert.Equal(t, expectedOrder[i], app.ID, "order mismatch at index %d", i)
	}

	// loadByUUID itself calls fetch per-key — dedup is provided by the
	// dataloader wrapper above it. Pin the per-key call count so we'd notice
	// if someone accidentally introduces extra round-trips inside the loop.
	assert.Equal(t, int32(5), calls.Load(),
		"loadByUUID should invoke fetch exactly once per input key")
}

// TestLoadByUUID_InvalidKeyErrorsPerRowWithoutAborting verifies that a single
// malformed UUID in the batch produces a per-row error and does not prevent
// the other valid keys from being fetched. Without this, one malformed cache
// key would 500-out an entire dashboard page.
func TestLoadByUUID_InvalidKeyErrorsPerRowWithoutAborting(t *testing.T) {
	ctx := context.Background()
	idA := uuid.New()
	idB := uuid.New()

	keys := dataloader.NewKeysFromStrings([]string{
		idA.String(),
		"not-a-uuid",
		idB.String(),
	})

	var calls atomic.Int32
	fetch := func(_ context.Context, id uuid.UUID) (*cqrs.App, error) {
		calls.Add(1)
		return &cqrs.App{ID: id}, nil
	}

	results := loadByUUID(ctx, keys, fetch)

	require.Len(t, results, 3)
	require.NoError(t, results[0].Error)
	require.Error(t, results[1].Error)
	assert.Contains(t, results[1].Error.Error(), "invalid uuid key")
	require.NoError(t, results[2].Error)

	// The invalid key must not consume a fetch call.
	assert.Equal(t, int32(2), calls.Load(),
		"loadByUUID must not call fetch for unparseable keys")
}

// TestLoadByUUID_PerKeyFetchErrorDoesNotAbortBatch pins that a fetch error
// on one key surfaces on that key's result without aborting the rest of the
// batch — important for a dashboard page that should render the runs it
// CAN resolve rather than 500-ing the whole list.
func TestLoadByUUID_PerKeyFetchErrorDoesNotAbortBatch(t *testing.T) {
	ctx := context.Background()
	idA := uuid.New()
	idB := uuid.New()
	idC := uuid.New()

	keys := dataloader.NewKeysFromStrings([]string{
		idA.String(),
		idB.String(),
		idC.String(),
	})

	notFound := errors.New("app not found")
	fetch := func(_ context.Context, id uuid.UUID) (*cqrs.App, error) {
		if id == idB {
			return nil, notFound
		}
		return &cqrs.App{ID: id}, nil
	}

	results := loadByUUID(ctx, keys, fetch)

	require.Len(t, results, 3)
	require.NoError(t, results[0].Error)
	assert.ErrorIs(t, results[1].Error, notFound)
	require.NoError(t, results[2].Error)
}
