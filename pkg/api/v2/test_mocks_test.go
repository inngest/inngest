package apiv2

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
)

var _ AppProvider = (*mockAppProvider)(nil)
var _ FunctionProvider = (*mockFunctionProvider)(nil)
var _ RunProvider = (*mockRunProvider)(nil)
var _ FunctionTraceReader = (*mockFunctionTraceReader)(nil)
var _ RateLimitProvider = (*mockRateLimitProvider)(nil)

type mockAppProvider struct {
	mock.Mock
}

func (m *mockAppProvider) GetApp(ctx context.Context, identifier string) (App, error) {
	args := m.Called(ctx, identifier)
	app, _ := args.Get(0).(App)
	return app, args.Error(1)
}

func (m *mockAppProvider) GetApps(ctx context.Context, opts GetAppsOpts) (*GetAppsResult, error) {
	args := m.Called(ctx, opts)
	result, _ := args.Get(0).(*GetAppsResult)
	return result, args.Error(1)
}

type mockFunctionProvider struct {
	mock.Mock
}

func (m *mockFunctionProvider) GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error) {
	args := m.Called(ctx, identifier)
	fn, _ := args.Get(0).(inngest.DeployedFunction)
	return fn, args.Error(1)
}

func (m *mockFunctionProvider) GetFunctionByApp(ctx context.Context, appID string, functionID string) (inngest.DeployedFunction, error) {
	args := m.Called(ctx, appID, functionID)
	fn, _ := args.Get(0).(inngest.DeployedFunction)
	return fn, args.Error(1)
}

func (m *mockFunctionProvider) GetFunctions(ctx context.Context, appID string, opts GetFunctionsOpts) (*GetFunctionsResult, error) {
	args := m.Called(ctx, appID, opts)
	result, _ := args.Get(0).(*GetFunctionsResult)
	return result, args.Error(1)
}

type mockRunProvider struct {
	mock.Mock
}

func (m *mockRunProvider) GetRun(ctx context.Context, runID ulid.ULID, opts GetRunOpts) (*cqrs.FunctionRun, error) {
	args := m.Called(ctx, runID, opts)
	run, _ := args.Get(0).(*cqrs.FunctionRun)
	return run, args.Error(1)
}

func (m *mockRunProvider) GetRuns(ctx context.Context, opts GetRunsOpts) (*GetRunsResult, error) {
	args := m.Called(ctx, opts)
	result, _ := args.Get(0).(*GetRunsResult)
	return result, args.Error(1)
}

func (m *mockRunProvider) Rerun(ctx context.Context, runID ulid.ULID, opts RerunOpts) (ulid.ULID, error) {
	args := m.Called(ctx, runID, opts)
	runID, _ = args.Get(0).(ulid.ULID)
	return runID, args.Error(1)
}

type mockFunctionTraceReader struct {
	mock.Mock
}

func (m *mockFunctionTraceReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	args := m.Called(ctx, runID)
	span, _ := args.Get(0).(*cqrs.OtelSpan)
	return span, args.Error(1)
}

func (m *mockFunctionTraceReader) GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	args := m.Called(ctx, id)
	output, _ := args.Get(0).(*cqrs.SpanOutput)
	return output, args.Error(1)
}

func (m *mockFunctionTraceReader) GetStepSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	args := m.Called(ctx, runID, stepID, accountID, workspaceID)
	span, _ := args.Get(0).(*cqrs.OtelSpan)
	return span, args.Error(1)
}

type mockRateLimitProvider struct {
	mock.Mock
}

func (m *mockRateLimitProvider) CheckRateLimit(ctx context.Context, method string) RateLimitResult {
	args := m.Called(ctx, method)
	result, _ := args.Get(0).(RateLimitResult)
	return result
}
