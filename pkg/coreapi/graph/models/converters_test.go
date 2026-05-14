package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
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

	t.Run("EndedAt is set only for terminal statuses", func(t *testing.T) {
		endedAt := time.Now().UTC()
		startedAt := endedAt.Add(-time.Minute)

		cases := []struct {
			name        string
			status      enums.RunStatus
			expectEnded bool
		}{
			{"running has no EndedAt", enums.RunStatusRunning, false},
			{"queued/scheduled has no EndedAt", enums.RunStatusScheduled, false},
			{"completed has EndedAt", enums.RunStatusCompleted, true},
			{"failed has EndedAt", enums.RunStatusFailed, true},
			{"cancelled has EndedAt", enums.RunStatusCancelled, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				run := &cqrs.TraceRun{
					RunID:      ulid.Make().String(),
					AppID:      uuid.New(),
					FunctionID: uuid.New(),
					Status:     tc.status,
					StartedAt:  startedAt,
					EndedAt:    endedAt,
				}
				got, err := MakeFunctionRunV2(run)
				require.NoError(t, err)
				require.NotNil(t, got)
				if tc.expectEnded {
					require.NotNil(t, got.EndedAt)
					assert.Equal(t, endedAt, *got.EndedAt)
				} else {
					assert.Nil(t, got.EndedAt)
				}
			})
		}
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
