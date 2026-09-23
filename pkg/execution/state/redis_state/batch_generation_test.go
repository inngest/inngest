package redis_state

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestBatchGenerationNamespacesKeysWithoutChangingLegacyKeys(t *testing.T) {
	r := miniredis.RunT(t)
	client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	defer client.Close()

	keys := NewBatchClient(client, "queue").KeyGenerator()
	functionID, batchID := uuid.New(), ulid.Make()
	legacy := keys.Batch(context.Background(), functionID, batchID)
	generated := keys.Batch(WithBatchGeneration(context.Background(), "01JGENERATION"), functionID, batchID)

	require.Equal(t, "{queue:"+functionID.String()+"}:batches:"+batchID.String(), legacy)
	require.Contains(t, generated, ":batchgen:01JGENERATION")
	require.NotEqual(t, legacy, generated)
	require.NotEqual(t,
		keys.BatchPointer(context.Background(), functionID),
		keys.BatchPointer(WithBatchGeneration(context.Background(), "01JGENERATION"), functionID),
	)
}
