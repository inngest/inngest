package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestHandlePausesIteratorCompletion(t *testing.T) {
	readErr := errors.New("pause store unavailable")
	for _, tc := range []struct {
		name  string
		count int
		err   error
		want  error
	}{
		{name: "empty"},
		{name: "populated", count: 2},
		{name: "legacy completion", count: 1, err: context.Canceled},
		{name: "iterator failure", count: 1, err: readErr, want: readErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &executor{
				log:   logger.VoidLogger(),
				queue: &stubQueue{},
				smv2:  &stubRunService{},
				pm: &iterationPauseManager{iter: &completionPauseIterator{
					count: tc.count, err: tc.err,
				}},
			}
			result, err := e.HandlePauses(context.Background(), event.InternalEvent{
				Event: event.Event{Name: "test/event"},
			})
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.want)
			}
			require.Equal(t, execution.HandlePauseResult{int32(tc.count), 0}, result)
		})
	}
}

type iterationPauseManager struct {
	pauses.Manager
	iter state.PauseIterator
}

func (m *iterationPauseManager) BufferLen(context.Context, pauses.Index) (int64, error) {
	return int64(m.iter.Count()), nil
}

func (m *iterationPauseManager) Aggregated(context.Context, pauses.Index, int64) (bool, error) {
	return false, nil
}

func (m *iterationPauseManager) PausesSince(context.Context, pauses.Index, time.Time) (state.PauseIterator, error) {
	return m.iter, nil
}

type completionPauseIterator struct {
	count int
	index int
	err   error
}

func (i *completionPauseIterator) Count() int   { return i.count }
func (i *completionPauseIterator) Index() int64 { return int64(i.index) }
func (i *completionPauseIterator) Error() error { return i.err }
func (i *completionPauseIterator) Next(context.Context) bool {
	if i.index >= i.count {
		return false
	}
	i.index++
	return true
}
func (i *completionPauseIterator) Val(context.Context) *state.Pause {
	expression := "false"
	return &state.Pause{Expression: &expression}
}
