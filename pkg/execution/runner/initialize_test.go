package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type scheduleErrExecutor struct {
	execution.Executor
	err error
}

func (e scheduleErrExecutor) Schedule(context.Context, execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	return nil, nil, e.err
}

func TestInitializeScheduleErrors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "skipped error is swallowed", err: executor.SkippedError{Reason: enums.SkipReasonAccountExecutionCapHit}},
		{name: "rate limited is swallowed", err: executor.ErrFunctionRateLimited},
		{name: "unknown error is returned", err: boom, wantErr: boom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := Initialize(context.Background(), InitOpts{
				appID: uuid.New(),
				fn:    inngest.Function{ID: uuid.New(), Slug: "fn"},
				evt:   event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, ulid.Make()),
				exec:  scheduleErrExecutor{err: tt.err},
			})
			require.Nil(t, md)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
