package devserver

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestNewFunctionProvider(t *testing.T) {
	ctx := context.Background()
	fnID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	appID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	store := &fakeFunctionStore{
		fns: []*cqrs.Function{
			{
				ID:     fnID,
				AppID:  appID,
				Slug:   "app-test-fn",
				Config: []byte(`{"name":"Test function","slug":"test-fn"}`),
			},
		},
		app: &cqrs.App{
			ID:   appID,
			Name: "app",
		},
	}

	provider := NewFunctionProvider(store)

	fn, err := provider.GetFunction(ctx, fnID.String())

	require.NoError(t, err)
	require.Equal(t, fnID, fn.ID)
	require.Equal(t, "app-test-fn", fn.Slug)
	require.Equal(t, appID, fn.AppID)
	require.Equal(t, "app", fn.AppName)
	require.Equal(t, consts.DevServerAccountID, fn.AccountID)
	require.Equal(t, consts.DevServerEnvID, fn.EnvironmentID)
	require.Equal(t, "Test function", fn.Function.Name)
	require.Equal(t, "test-fn", fn.Function.Slug)
}

func TestNewFunctionProviderErrors(t *testing.T) {
	t.Run("reader error", func(t *testing.T) {
		provider := NewFunctionProvider(&fakeFunctionStore{err: errors.New("read failed")})

		fn, err := provider.GetFunction(context.Background(), "missing")

		require.ErrorContains(t, err, "read failed")
		require.Empty(t, fn.ID)
	})

	t.Run("not found", func(t *testing.T) {
		provider := NewFunctionProvider(&fakeFunctionStore{})

		fn, err := provider.GetFunction(context.Background(), "missing")

		require.ErrorContains(t, err, "function not found")
		require.Empty(t, fn.ID)
	})

	t.Run("invalid function config", func(t *testing.T) {
		provider := NewFunctionProvider(&fakeFunctionStore{
			fns: []*cqrs.Function{
				{
					ID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Slug:   "bad-fn",
					Config: []byte(`not-json`),
				},
			},
		})

		fn, err := provider.GetFunction(context.Background(), "bad-fn")

		require.Error(t, err)
		require.Empty(t, fn.ID)
	})
}

func TestFunctionRunReader(t *testing.T) {
	runID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cy")
	run := &cqrs.FunctionRun{RunID: runID}
	reader := &fakeFunctionRunReader{run: run}

	result, err := NewFunctionRunReader(reader).GetFunctionRun(context.Background(), runID)

	require.NoError(t, err)
	require.Equal(t, run, result)
	require.Equal(t, consts.DevServerAccountID, reader.accountID)
	require.Equal(t, consts.DevServerEnvID, reader.workspaceID)
	require.Equal(t, runID, reader.runID)
}

func TestFunctionTraceReader(t *testing.T) {
	runID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cy")
	spanID := cqrs.SpanIdentifier{SpanID: "span"}
	root := &cqrs.OtelSpan{RunID: runID}
	output := &cqrs.SpanOutput{Data: []byte(`{"ok":true}`)}
	reader := &fakeTraceReader{root: root, output: output}
	traceReader := NewFunctionTraceReader(reader)

	result, err := traceReader.GetSpansByRunID(context.Background(), runID)
	require.NoError(t, err)
	require.Equal(t, root, result)
	require.Equal(t, runID, reader.runID)

	spanOutput, err := traceReader.GetSpanOutput(context.Background(), spanID)
	require.NoError(t, err)
	require.Equal(t, output, spanOutput)
	require.Equal(t, spanID, reader.spanID)
}

type fakeFunctionStore struct {
	fns []*cqrs.Function
	app *cqrs.App
	err error
}

func (f *fakeFunctionStore) GetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	return f.fns, f.err
}

func (f *fakeFunctionStore) GetApps(ctx context.Context, envID uuid.UUID, filter *cqrs.FilterAppParam) ([]*cqrs.App, error) {
	return nil, nil
}

func (f *fakeFunctionStore) GetAppByChecksum(ctx context.Context, envID uuid.UUID, checksum string) (*cqrs.App, error) {
	return nil, nil
}

func (f *fakeFunctionStore) GetAppByURL(ctx context.Context, envID uuid.UUID, url string) (*cqrs.App, error) {
	return nil, nil
}

func (f *fakeFunctionStore) GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*cqrs.App, error) {
	return nil, nil
}

func (f *fakeFunctionStore) GetAllApps(ctx context.Context, envID uuid.UUID) ([]*cqrs.App, error) {
	return nil, nil
}

func (f *fakeFunctionStore) GetAppByID(ctx context.Context, id uuid.UUID) (*cqrs.App, error) {
	if f.app == nil || f.app.ID != id {
		return nil, errors.New("app not found")
	}
	return f.app, nil
}

type fakeFunctionRunReader struct {
	run         *cqrs.FunctionRun
	accountID   uuid.UUID
	workspaceID uuid.UUID
	runID       ulid.ULID
}

func (f *fakeFunctionRunReader) GetFunctionRunsFromEvents(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, eventIDs []ulid.ULID) ([]*cqrs.FunctionRun, error) {
	return nil, nil
}

func (f *fakeFunctionRunReader) GetFunctionRun(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	f.accountID = accountID
	f.workspaceID = workspaceID
	f.runID = runID
	return f.run, nil
}

type fakeTraceReader struct {
	root   *cqrs.OtelSpan
	output *cqrs.SpanOutput
	runID  ulid.ULID
	spanID cqrs.SpanIdentifier
}

func (f *fakeTraceReader) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetTraceRunsCount(ctx context.Context, opt cqrs.GetTraceRunOpt) (int, error) {
	return 0, nil
}

func (f *fakeTraceReader) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetTraceRunsByRunIDs(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID]*cqrs.TraceRun, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetTraceSpansByRun(ctx context.Context, id cqrs.TraceRunIdentifier) ([]*cqrs.Span, error) {
	return nil, nil
}

func (f *fakeTraceReader) LegacyGetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetSpanStack(ctx context.Context, id cqrs.SpanIdentifier) ([]string, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	f.runID = runID
	return f.root, nil
}

func (f *fakeTraceReader) GetSpansByDebugRunID(ctx context.Context, debugRunID ulid.ULID) ([]*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetSpansByDebugSessionID(ctx context.Context, debugSessionID ulid.ULID) ([][]*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	f.spanID = id
	return f.output, nil
}

func (f *fakeTraceReader) GetRunSpanByRunID(ctx context.Context, runID ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetStepSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetExecutionSpanByStepIDAndAttempt(ctx context.Context, runID ulid.ULID, stepID string, attempt int, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetLatestExecutionSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetSpanBySpanID(ctx context.Context, runID ulid.ULID, spanID string, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, nil
}

func (f *fakeTraceReader) OtelTracesEnabled(ctx context.Context, accountID uuid.UUID) (bool, error) {
	return true, nil
}

func (f *fakeTraceReader) GetEventRuns(ctx context.Context, eventID ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) ([]*cqrs.FunctionRun, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetRun(ctx context.Context, runID ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.FunctionRun, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetEvent(ctx context.Context, id ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.Event, error) {
	return nil, nil
}

func (f *fakeTraceReader) GetEvents(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	return nil, nil
}
