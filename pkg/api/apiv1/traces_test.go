package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

type traceTestAuth struct {
	accountID uuid.UUID
	envID     uuid.UUID
}

func (a traceTestAuth) AccountID() uuid.UUID {
	return a.accountID
}

func (a traceTestAuth) WorkspaceID() uuid.UUID {
	return a.envID
}

type traceTestFunctionReader struct {
	functions map[uuid.UUID]*cqrs.Function
}

func (r traceTestFunctionReader) GetFunctionByInternalUUID(_ context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	if fn, ok := r.functions[fnID]; ok {
		return fn, nil
	}
	return nil, errors.New("not found")
}

func (r traceTestFunctionReader) GetFunctionsByAppExternalID(context.Context, uuid.UUID, string) ([]*cqrs.Function, error) {
	return nil, errors.New("not implemented")
}

func (r traceTestFunctionReader) GetFunctionsByAppInternalID(context.Context, uuid.UUID) ([]*cqrs.Function, error) {
	return nil, errors.New("not implemented")
}

func (r traceTestFunctionReader) GetFunctionByExternalID(context.Context, uuid.UUID, string, string) (*cqrs.Function, error) {
	return nil, errors.New("not implemented")
}

func (r traceTestFunctionReader) GetActiveFunctionByAppAndSlug(context.Context, string, string) (*cqrs.Function, error) {
	return nil, errors.New("not implemented")
}

type traceTestTracerProvider struct {
	failSpanID string
	mu         sync.Mutex
	created    []string
}

func (p *traceTestTracerProvider) CreateSpan(_ context.Context, _ string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	spanID := opts.SpanID.String()
	if spanID == p.failSpanID {
		return nil, errors.New("create span failed")
	}

	p.mu.Lock()
	p.created = append(p.created, spanID)
	p.mu.Unlock()

	return &meta.SpanReference{
		TraceParent: "00-00000000000000000000000000000001-000000000001-01",
	}, nil
}

func (p *traceTestTracerProvider) CreateDroppableSpan(context.Context, string, *tracing.CreateSpanOptions) (*tracing.DroppableSpan, error) {
	return nil, errors.New("not implemented")
}

func (p *traceTestTracerProvider) UpdateSpan(context.Context, *tracing.UpdateSpanOptions) error {
	return nil
}

func TestConvertOTLPAndSendReturnsOnlyCommittedUsage(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()
	fnID := uuid.New()

	successSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 1}
	failedSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 2}
	successSpan := traceTestSpan(fnID, successSpanID)
	failedSpan := traceTestSpan(fnID, failedSpanID)

	tracer := &traceTestTracerProvider{
		failSpanID: trace.SpanID(failedSpanID).String(),
	}
	r := router{API: &API{opts: Opts{
		FunctionReader: traceTestFunctionReader{
			functions: map[uuid.UUID]*cqrs.Function{
				fnID: {
					ID:    fnID,
					EnvID: envID,
					AppID: appID,
				},
			},
		},
		TracerProvider: tracer,
	}}}

	accepted, rejected := r.convertOTLPAndSend(ctx, traceTestAuth{
		accountID: accountID,
		envID:     envID,
	}, traceTestTraceRequest(successSpan, failedSpan))

	require.Equal(t, int64(1), rejected)
	require.Len(t, accepted, 1)
	require.Equal(t, envID, accepted[0].WorkspaceID)
	require.Equal(t, appID, accepted[0].AppID)
	require.Equal(t, fnID, accepted[0].FunctionID)
	require.Equal(t, int64(proto.Size(successSpan)), accepted[0].Bytes)
	require.Equal(t, []string{trace.SpanID(successSpanID).String()}, tracer.created)
}

func traceTestTraceRequest(spans ...*tracev1.Span) *collecttrace.ExportTraceServiceRequest {
	return &collecttrace.ExportTraceServiceRequest{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name: "inngest",
						},
						Spans: spans,
					},
				},
			},
		},
	}
}

func traceTestSpan(fnID uuid.UUID, spanID []byte) *tracev1.Span {
	return &tracev1.Span{
		Name:   "inngest.execution",
		SpanId: spanID,
		Kind:   tracev1.Span_SPAN_KIND_INTERNAL,
		Attributes: []*commonv1.KeyValue{
			traceTestStringAttr("inngest.traceref", traceTestTraceRef()),
			traceTestStringAttr(consts.OtelAttrSDKRunID, ulid.Make().String()),
			traceTestStringAttr(consts.OtelSysFunctionID, fnID.String()),
		},
	}
}

func traceTestStringAttr(key, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		},
	}
}

func traceTestTraceRef() string {
	b, err := json.Marshal(meta.SpanReference{
		TraceParent: "00-00000000000000000000000000000001-000000000001-01",
	})
	if err != nil {
		panic(err)
	}
	return url.QueryEscape(string(b))
}

var _ apiv1auth.V1Auth = traceTestAuth{}
var _ cqrs.FunctionReader = traceTestFunctionReader{}
var _ tracing.TracerProvider = (*traceTestTracerProvider)(nil)
