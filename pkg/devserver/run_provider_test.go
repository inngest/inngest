package devserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRunProviderRerunSchedulesOriginalEvent(t *testing.T) {
	runID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T00")
	originalRunID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T01")
	newRunID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T02")
	eventID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T03")
	functionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	appID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	fnConfig, err := json.Marshal(inngest.Function{
		ID:   functionID,
		Name: "Test function",
		Slug: "test-function",
	})
	require.NoError(t, err)

	data := &stubRunProviderDataReader{
		run: &cqrs.FunctionRun{
			RunID:         runID,
			OriginalRunID: &originalRunID,
			FunctionID:    functionID,
			EventID:       eventID,
		},
		fn: &cqrs.Function{
			ID:     functionID,
			AppID:  appID,
			Slug:   "test-function",
			Config: fnConfig,
		},
		evt: &cqrs.Event{
			ID:        eventID,
			EventID:   "evt-1",
			EventName: "test/event",
			EventData: map[string]any{"ok": true},
		},
	}
	scheduler := &stubRunProviderScheduler{runID: newRunID}
	provider := &runProvider{data: data, scheduler: scheduler}

	result, err := provider.Rerun(context.Background(), runID, apiv2.RerunOpts{
		FromStep: &apiv2.RerunFromStep{
			StepID: "step-1",
			Input:  json.RawMessage(`[{"foo":"bar"}]`),
		},
	})

	require.NoError(t, err)
	require.Equal(t, newRunID, result)
	require.NotNil(t, scheduler.req)
	require.Equal(t, appID, scheduler.req.AppID)
	require.Equal(t, consts.DevServerAccountID, scheduler.req.AccountID)
	require.Equal(t, consts.DevServerEnvID, scheduler.req.WorkspaceID)
	require.True(t, scheduler.req.PreventRateLimit)
	require.Equal(t, originalRunID, *scheduler.req.OriginalRunID)
	require.Equal(t, "test-function", scheduler.req.Function.Slug)
	require.NotNil(t, scheduler.req.FromStep)
	require.Equal(t, "step-1", scheduler.req.FromStep.StepID)
	require.JSONEq(t, `[{"foo":"bar"}]`, string(scheduler.req.FromStep.Input))
	require.Len(t, scheduler.req.Events, 1)
	require.Equal(t, eventID, scheduler.req.Events[0].GetInternalID())
	require.Equal(t, "test/event", scheduler.req.Events[0].GetEvent().Name)
}

func TestRunProviderRerunUsesRunIDWhenOriginalRunIDIsMissing(t *testing.T) {
	runID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T00")
	newRunID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T02")
	eventID := ulid.MustParse("01HR3ZJ4Z4E0MZ6PRP7Z3A4T03")
	functionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	fnConfig, err := json.Marshal(inngest.Function{
		ID:   functionID,
		Name: "Test function",
		Slug: "test-function",
	})
	require.NoError(t, err)

	scheduler := &stubRunProviderScheduler{runID: newRunID}
	provider := &runProvider{
		data: &stubRunProviderDataReader{
			run: &cqrs.FunctionRun{
				RunID:      runID,
				FunctionID: functionID,
				EventID:    eventID,
			},
			fn: &cqrs.Function{
				ID:     functionID,
				AppID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				Config: fnConfig,
			},
			evt: &cqrs.Event{
				ID:        eventID,
				EventName: "test/event",
			},
		},
		scheduler: scheduler,
	}

	_, err = provider.Rerun(context.Background(), runID, apiv2.RerunOpts{})

	require.NoError(t, err)
	require.NotNil(t, scheduler.req)
	require.Equal(t, runID, *scheduler.req.OriginalRunID)
}

type stubRunProviderDataReader struct {
	run *cqrs.FunctionRun
	fn  *cqrs.Function
	evt *cqrs.Event
}

func (s *stubRunProviderDataReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (s *stubRunProviderDataReader) GetFunctionRun(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	return s.run, nil
}

func (s *stubRunProviderDataReader) GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	return s.fn, nil
}

func (s *stubRunProviderDataReader) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error) {
	return s.evt, nil
}

type stubRunProviderScheduler struct {
	runID ulid.ULID
	req   *execution.ScheduleRequest
}

func (s *stubRunProviderScheduler) Schedule(ctx context.Context, req execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	s.req = &req
	return &s.runID, nil, nil
}

var _ runProviderDataReader = (*stubRunProviderDataReader)(nil)
var _ apiv2.FunctionScheduler = (*stubRunProviderScheduler)(nil)
var _ event.TrackedEvent = (*cqrs.Event)(nil)
