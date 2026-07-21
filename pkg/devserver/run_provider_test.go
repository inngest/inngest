package devserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
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

func TestScoreMetadataLoaderReconstructsFinalizedRunMetadata(t *testing.T) {
	runID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTQ")
	eventID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTR")
	batchID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTS")
	originalRunID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTT")
	functionID := uuid.New()
	appID := uuid.New()
	startedAt := time.Date(2026, time.June, 18, 12, 0, 0, 0, time.UTC)

	loader := scoreMetadataLoader(&stubRunProviderDataReader{
		run: &cqrs.FunctionRun{
			RunID:           runID,
			RunStartedAt:    startedAt,
			FunctionID:      functionID,
			FunctionVersion: 7,
			EventID:         eventID,
			BatchID:         &batchID,
			OriginalRunID:   &originalRunID,
		},
		fn: &cqrs.Function{
			ID:    functionID,
			AppID: appID,
		},
	})

	md, err := loader(context.Background(), sv2.ID{
		RunID: runID,
		Tenant: sv2.Tenant{
			AccountID: consts.DevServerAccountID,
			EnvID:     consts.DevServerEnvID,
		},
	})
	require.NoError(t, err)
	require.Equal(t, runID, md.ID.RunID)
	require.Equal(t, functionID, md.ID.FunctionID)
	require.Equal(t, consts.DevServerAccountID, md.ID.Tenant.AccountID)
	require.Equal(t, consts.DevServerEnvID, md.ID.Tenant.EnvID)
	require.Equal(t, appID, md.ID.Tenant.AppID)
	require.Equal(t, 7, md.Config.FunctionVersion)
	require.Equal(t, startedAt, md.Config.StartedAt)
	require.Equal(t, []ulid.ULID{eventID}, md.Config.EventIDs)
	require.Equal(t, &batchID, md.Config.BatchID)
	require.Equal(t, &originalRunID, md.Config.OriginalRunID)
}

func TestScoreMetadataLoaderMapsMissingRowsToMetadataNotFound(t *testing.T) {
	loader := scoreMetadataLoader(&stubRunProviderDataReader{runErr: sql.ErrNoRows})

	md, err := loader(context.Background(), sv2.ID{
		RunID: ulid.Make(),
		Tenant: sv2.Tenant{
			AccountID: consts.DevServerAccountID,
			EnvID:     consts.DevServerEnvID,
		},
	})
	require.Nil(t, md)
	require.ErrorIs(t, err, sv2.ErrMetadataNotFound)
}

func TestRunProviderGetEventRunsUsesSharedRunList(t *testing.T) {
	appID := uuid.New()
	functionID := uuid.New()
	runID := ulid.Make()
	eventID := ulid.Make()
	cursor := "opaque-cursor"
	startedAt := time.Now().UTC()

	data := &stubRunProviderDataReader{
		listedRuns: []*cqrs.TraceRun{{
			RunID:        runID.String(),
			Cursor:       "next-cursor",
			AppID:        appID,
			AppName:      "inngest-ai",
			FunctionID:   functionID,
			FunctionSlug: "hello-world",
			FunctionName: "Hello",
			StartedAt:    startedAt,
			Status:       enums.RunStatusCompleted,
			TriggerIDs:   []string{eventID.String()},
		}},
	}
	provider := &runProvider{data: data}

	result, err := provider.GetRuns(t.Context(), apiv2.GetRunsOpts{
		EventID: eventID,
		Cursor:  cursor,
		Limit:   20,
	})
	require.NoError(t, err)
	require.Len(t, result.Runs, 1)
	require.NotNil(t, data.listOpts)
	assert.Equal(t, consts.DevServerAccountID, data.listOpts.Filter.AccountID)
	assert.Equal(t, consts.DevServerEnvID, data.listOpts.Filter.WorkspaceID)
	assert.Equal(t, []ulid.ULID{eventID}, data.listOpts.Filter.EventID)
	assert.Equal(t, cursor, data.listOpts.Cursor)
	assert.Equal(t, uint(21), data.listOpts.Items)
	assert.Equal(t, enums.TraceRunTimeQueuedAt, data.listOpts.Filter.TimeField)
	assert.Equal(t, enums.TraceRunOrderDesc, data.listOpts.Order[0].Direction)
	assert.Equal(t, "inngest-ai", result.Runs[0].AppID)
	assert.Equal(t, "hello-world", result.Runs[0].FunctionID)
	assert.Equal(t, "Hello", result.Runs[0].FunctionName)
	assert.Equal(t, eventID, result.Runs[0].EventID)
	assert.Equal(t, "next-cursor", result.Runs[0].Cursor)
}

type stubRunProviderDataReader struct {
	run        *cqrs.FunctionRun
	runErr     error
	fn         *cqrs.Function
	fnErr      error
	evt        *cqrs.Event
	evtErr     error
	listedRuns []*cqrs.TraceRun
	listOpts   *cqrs.GetTraceRunOpt
}

func (s *stubRunProviderDataReader) GetRuns(ctx context.Context, opts cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	s.listOpts = &opts
	return s.listedRuns, nil
}

func (s *stubRunProviderDataReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (s *stubRunProviderDataReader) GetFunctionRun(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	if s.runErr != nil {
		return nil, s.runErr
	}
	return s.run, nil
}

func (s *stubRunProviderDataReader) GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	if s.fnErr != nil {
		return nil, s.fnErr
	}
	return s.fn, nil
}

func (s *stubRunProviderDataReader) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error) {
	if s.evtErr != nil {
		return nil, s.evtErr
	}
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
