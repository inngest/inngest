package models

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeFunctionRunV2(t *testing.T) {
	t.Run("nil input returns (nil, nil)", func(t *testing.T) {
		got, err := MakeFunctionRunV2(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("bad RunID returns an error", func(t *testing.T) {
		run := &cqrs.TraceRun{
			RunID: "not-a-ulid",
		}
		got, err := MakeFunctionRunV2(run)
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("invalid trigger ULIDs are filtered out", func(t *testing.T) {
		good := ulid.Make()
		run := &cqrs.TraceRun{
			RunID:      ulid.Make().String(),
			Status:     enums.RunStatusRunning,
			TriggerIDs: []string{good.String(), "not-a-ulid", ""},
		}
		got, err := MakeFunctionRunV2(run)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Len(t, got.TriggerIDs, 1)
		assert.Equal(t, good, got.TriggerIDs[0])
	})

	t.Run("BatchID's timestamp surfaces as BatchCreatedAt", func(t *testing.T) {
		want := time.UnixMilli(1700000000000)
		batchID := ulid.MustNew(uint64(want.UnixMilli()), ulid.DefaultEntropy())
		run := &cqrs.TraceRun{
			RunID:   ulid.Make().String(),
			Status:  enums.RunStatusRunning,
			BatchID: &batchID,
		}
		got, err := MakeFunctionRunV2(run)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotNil(t, got.BatchCreatedAt)
		assert.Equal(t, want.UnixMilli(), got.BatchCreatedAt.UnixMilli())
	})

	t.Run("zero StartedAt yields nil StartedAt", func(t *testing.T) {
		run := &cqrs.TraceRun{
			RunID:  ulid.Make().String(),
			Status: enums.RunStatusScheduled,
		}
		got, err := MakeFunctionRunV2(run)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Nil(t, got.StartedAt)
	})
}
