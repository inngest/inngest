package loader

import (
	"context"
	"errors"
	"testing"

	"github.com/graph-gophers/dataloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManyReturnsSingleError(t *testing.T) {
	expectedErr := errors.New("batch failed")
	keyCount := 0
	loader := dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		keyCount = len(keys)

		return []*dataloader.Result{
			{Error: expectedErr},
		}
	})

	results, err := LoadMany[string](
		context.Background(),
		loader,
		dataloader.NewKeysFromStrings([]string{"run"}),
	)

	require.ErrorIs(t, err, expectedErr)
	assert.Empty(t, results)
	assert.Equal(t, 1, keyCount)
}

func TestLoadManyReturnsFirstNonNilError(t *testing.T) {
	expectedErr := errors.New("second key failed")
	loader := dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		return []*dataloader.Result{
			{Data: "ok"},
			{Error: expectedErr},
		}
	})

	results, err := LoadMany[string](
		context.Background(),
		loader,
		dataloader.NewKeysFromStrings([]string{"ok", "failed"}),
	)

	require.ErrorIs(t, err, expectedErr)
	assert.Empty(t, results)
}
