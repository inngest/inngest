package apiv2

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
)

var _ FunctionProvider = (*mockFunctionProvider)(nil)
var _ FunctionRunReader = (*mockFunctionRunReader)(nil)
var _ FunctionTraceReader = (*mockFunctionTraceReader)(nil)

type mockFunctionProvider struct {
	mock.Mock
}

func (m *mockFunctionProvider) GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error) {
	args := m.Called(ctx, identifier)
	fn, _ := args.Get(0).(inngest.DeployedFunction)
	return fn, args.Error(1)
}

type mockFunctionRunReader struct {
	mock.Mock
}

func (m *mockFunctionRunReader) GetFunctionRun(ctx context.Context, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	args := m.Called(ctx, runID)
	run, _ := args.Get(0).(*cqrs.FunctionRun)
	return run, args.Error(1)
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
