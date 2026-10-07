package batch

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/tests/execution/queue/helper"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// Cluster mode matters here: the batch scripts access keys outside KEYS and
// rely on hash tags to keep them in one slot.
func TestBatchValkeyCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []struct {
		Name       string
		ValkeyOpts []helper.ValkeyOption
	}{
		{
			Name: "Valkey 9 cluster",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyV9Image),
				helper.WithValkeyCluster(true),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			container, err := helper.StartValkey(t, tc.ValkeyOpts...)
			require.NoError(t, err)
			t.Cleanup(func() { _ = container.Terminate(ctx) })

			client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, true)
			require.NoError(t, err)
			t.Cleanup(func() { client.Close() })

			bc := redis_state.NewBatchClient(client, redis_state.QueueDefaultKey)
			bm := batch.NewRedisBatchManager(bc, nil, batch.WithoutBuffer())

			newFn := func(maxSize int) inngest.Function {
				return inngest.Function{
					ID:         uuid.New(),
					EventBatch: &inngest.EventBatchConfig{MaxSize: maxSize, Timeout: "60s"},
				}
			}
			newItem := func(fn inngest.Function, data map[string]any) batch.BatchItem {
				return batch.BatchItem{
					FunctionID: fn.ID,
					EventID:    ulid.Make(),
					Event:      event.Event{Name: "test/event", Data: data},
				}
			}

			t.Run("append", func(t *testing.T) {
				fn := newFn(3)
				first := newItem(fn, nil)

				res, err := bm.Append(ctx, first, fn)
				require.NoError(t, err)
				require.Equal(t, enums.BatchNew, res.Status)

				// A retried first event still reports new so the batch gets scheduled.
				res, err = bm.Append(ctx, first, fn)
				require.NoError(t, err)
				require.Equal(t, enums.BatchNew, res.Status)

				res, err = bm.Append(ctx, newItem(fn, nil), fn)
				require.NoError(t, err)
				require.Equal(t, enums.BatchAppend, res.Status)

				res, err = bm.Append(ctx, newItem(fn, nil), fn)
				require.NoError(t, err)
				require.Equal(t, enums.BatchFull, res.Status)

				items, err := bm.RetrieveItems(ctx, fn.ID, ulid.MustParse(res.BatchID))
				require.NoError(t, err)
				require.Len(t, items, 3)
				require.Equal(t, first.EventID, items[0].EventID)
			})

			t.Run("maxsize", func(t *testing.T) {
				sized := batch.NewRedisBatchManager(bc, nil, batch.WithoutBuffer(), batch.WithRedisBatchSizeLimit(1024))
				fn := newFn(100)

				res, err := sized.Append(ctx, newItem(fn, nil), fn)
				require.NoError(t, err)
				require.Equal(t, enums.BatchNew, res.Status)

				res, err = sized.Append(ctx, newItem(fn, map[string]any{"pad": strings.Repeat("x", 2048)}), fn)
				require.NoError(t, err)
				require.Equal(t, enums.BatchMaxSize, res.Status)
			})

			t.Run("bulk append", func(t *testing.T) {
				fn := newFn(3)
				items := make([]batch.BatchItem, 5)
				for i := range items {
					items[i] = newItem(fn, nil)
				}

				res, err := bm.BulkAppend(ctx, items, fn)
				require.NoError(t, err)
				require.Equal(t, "overflow", res.Status)
				require.Equal(t, 5, res.Committed)
				require.Equal(t, 2, res.OverflowCount)

				overflow, err := bm.RetrieveItems(ctx, fn.ID, ulid.MustParse(res.NextBatchID))
				require.NoError(t, err)
				require.Len(t, overflow, 2)

				res, err = bm.BulkAppend(ctx, items, fn)
				require.NoError(t, err)
				require.Equal(t, "itemexists", res.Status)
				require.Equal(t, 5, res.Duplicates)
			})

			t.Run("start", func(t *testing.T) {
				fn := newFn(10)
				res, err := bm.Append(ctx, newItem(fn, nil), fn)
				require.NoError(t, err)
				batchID := ulid.MustParse(res.BatchID)

				status, err := bm.StartExecution(ctx, fn.ID, batchID, res.BatchPointerKey)
				require.NoError(t, err)
				require.Equal(t, enums.BatchStatusReady.String(), status)

				status, err = bm.StartExecution(ctx, fn.ID, batchID, res.BatchPointerKey)
				require.NoError(t, err)
				require.Equal(t, enums.BatchStatusStarted.String(), status)

				status, err = bm.StartExecution(ctx, fn.ID, ulid.Make(), res.BatchPointerKey)
				require.NoError(t, err)
				require.Equal(t, enums.BatchStatusAbsent.String(), status)
			})

			t.Run("keyed delete", func(t *testing.T) {
				fn := newFn(10)
				fn.EventBatch.Key = new("event.data.tenant")

				_, err := bm.Append(ctx, newItem(fn, map[string]any{"tenant": "a"}), fn)
				require.NoError(t, err)
				_, err = bm.Append(ctx, newItem(fn, map[string]any{"tenant": "b"}), fn)
				require.NoError(t, err)

				info, err := bm.GetBatchInfo(ctx, fn.ID, "a")
				require.NoError(t, err)
				require.Len(t, info.Items, 1)
				require.Equal(t, enums.BatchStatusPending.String(), info.Status)

				deleted, err := bm.DeleteBatch(ctx, fn.ID, "a")
				require.NoError(t, err)
				require.True(t, deleted.Deleted)

				items, err := bm.RetrieveItems(ctx, fn.ID, ulid.MustParse(info.BatchID))
				require.NoError(t, err)
				require.Empty(t, items)

				info, err = bm.GetBatchInfo(ctx, fn.ID, "b")
				require.NoError(t, err)
				require.Len(t, info.Items, 1)
			})
		})
	}
}
