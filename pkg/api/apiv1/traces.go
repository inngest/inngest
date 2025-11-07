package apiv1

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type TraceParent struct {
	TraceID trace.TraceID
	SpanID  trace.SpanID
}

type TraceRoot struct{}

func (a router) traces(w http.ResponseWriter, r *http.Request) {
	// Auth the app
	auth, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		respondError(w, r, http.StatusUnauthorized, "No auth found")
		return
	}

	ctx := context.Background()
	enabled, err := a.opts.TraceReader.OtelTracesEnabled(ctx, auth.AccountID())
	if err != nil {
		respondError(w, r, http.StatusUnauthorized, "Error checking OTel traces entitlement")
		return
	}
	if !enabled {
		respondError(w, r, http.StatusUnauthorized, "OTel traces are not enabled for this account")
		return
	}

	// Check that the trace ID is valid and accessible to the app.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "Error reading body")
		return
	}

	req := &collecttrace.ExportTraceServiceRequest{}
	isJSON := strings.Contains(r.Header.Get("Content-Type"), "json")
	if isJSON {
		err = protojson.Unmarshal(body, req)
	} else {
		err = proto.Unmarshal(body, req)
	}
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid payload")
		return
	}

	rejectedSpans := a.convertOTLPAndSend(r.Context(), auth, req)

	resp := &collecttrace.ExportTraceServiceResponse{}
	if rejectedSpans > 0 {
		resp.PartialSuccess = &collecttrace.ExportTracePartialSuccess{
			RejectedSpans: rejectedSpans,
		}
	}
	var respBytes []byte
	if isJSON {
		respBytes, _ = protojson.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
	} else {
		respBytes, _ = proto.Marshal(resp)
		w.Header().Set("Content-Type", "application/x-protobuf")
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write(respBytes)
		_ = gz.Close()
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func respondError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	isJSON := strings.Contains(r.Header.Get("Content-Type"), "json")
	status := &statuspb.Status{Message: msg}

	var data []byte
	if isJSON {
		data, _ = protojson.Marshal(status)
		w.Header().Set("Content-Type", "application/json")
	} else {
		data, _ = proto.Marshal(status)
		w.Header().Set("Content-Type", "application/x-protobuf")
	}

	w.WriteHeader(code)
	_, _ = w.Write(data)
}

func (a router) convertOTLPAndSend(ctx context.Context, auth apiv1auth.V1Auth, req *collecttrace.ExportTraceServiceRequest) int64 {
	var (
		errs atomic.Int64
		wg   sync.WaitGroup
	)

	l := logger.StdlibLogger(ctx).With(
		"account_id", auth.AccountID(),
		"workspace_id", auth.WorkspaceID(),
	)

	for _, rs := range req.ResourceSpans {
		res := convertResource(rs.Resource)

		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {

				wg.Add(1)

				go func() {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							l.Error("failed to commit span with", "panic", r)
							errs.Add(1)
						}
					}()

					err := a.commitSpan(ctx, auth, res, ss.Scope, s)
					if err != nil {
						l.Error("failed to commit span with", "error", err)
						errs.Add(1)
						return
					}
				}()
			}
		}
	}

	wg.Wait()

	return errs.Load()
}

func (a router) commitSpan(ctx context.Context, auth apiv1auth.V1Auth, res *resource.Resource, scope *commonv1.InstrumentationScope, s *tracev1.Span) error {
	// To be valid, each span must have an "inngest.traceref" attribute
	tr, err := getInngestTraceRef(s)
	if err != nil {
		// If we can't find the traceref, we can't create a span. So let's
		// skip it.
		return fmt.Errorf("failed to get traceref: %w", err)
	}

	attrs := convertAttributes(s.Attributes)

	status := enums.StepStatusUnknown
	if s.Status != nil {
		switch s.Status.Code {
		case tracev1.Status_STATUS_CODE_ERROR:
			status = enums.StepStatusFailed
		case tracev1.Status_STATUS_CODE_OK:
			status = enums.StepStatusCompleted
		}
	}

	// Legacy, but try to pull the run ID out from the attributes using the
	// legacy key.
	var runID ulid.ULID
	for _, kv := range attrs {
		if kv.Key == consts.OtelAttrSDKRunID {
			runID, err = ulid.Parse(kv.Value.AsString())
			if err != nil {
				return fmt.Errorf("failed to parse run ID from attributes: %w", err)
			}

			break
		}
	}

	spanID := trace.SpanID(s.SpanId).String()
	spanKind := trace.SpanKind(s.Kind).String()
	resourceServiceName := resourceServiceName(res)
	isUserland := true

	run, err := a.opts.FunctionRunReader.GetFunctionRun(ctx, auth.AccountID(), auth.WorkspaceID(), runID)
	if err != nil {
		return fmt.Errorf("function run not found: %w", err)
	}
	functionID := run.FunctionID

	fn, err := a.opts.FunctionReader.GetFunctionByInternalUUID(ctx, functionID)
	if err != nil {
		return fmt.Errorf("function not found: %w", err)
	}
	if fn.IsArchived() {
		return fmt.Errorf("function is archived: %s", functionID)
	}

	ourAttrs := meta.NewAttrSet(
		meta.Attr(meta.Attrs.IsUserland, &isUserland),
		meta.Attr(meta.Attrs.UserlandSpanID, &spanID),
		meta.Attr(meta.Attrs.DynamicSpanID, &spanID),
		meta.Attr(meta.Attrs.UserlandName, &s.Name),
		meta.Attr(meta.Attrs.DynamicStatus, &status),
		meta.Attr(meta.Attrs.RunID, &runID),
		meta.Attr(meta.Attrs.UserlandKind, &spanKind),
		meta.Attr(meta.Attrs.UserlandServiceName, &resourceServiceName),
		meta.Attr(meta.Attrs.UserlandScopeName, &scope.Name),
		meta.Attr(meta.Attrs.UserlandScopeVersion, &scope.Version),
		meta.Attr(meta.Attrs.AppID, &fn.AppID),
		meta.Attr(meta.Attrs.FunctionID, &functionID),
	)

	// Add some additional attributes on top
	attrs = append(attrs, ourAttrs.Serialize()...)

	// By default, the parent span is the trace ref we found.
	parent := tr

	if (scope.Name != "inngest" || s.Name != "inngest.execution") && len(s.ParentSpanId) == 12 {
		// If this is not the "root" span created by an SDK, we need to listen to
		// the parent span ID that they have set so we can preserve whatever
		// lineage they're passing us.
		parent, err = tr.SetParentSpanID(trace.SpanID(s.ParentSpanId))
		if err != nil {
			return fmt.Errorf("failed to set parent span ID: %w", err)
		}
	}

	_, err = a.opts.TracerProvider.CreateSpan(context.Background(), meta.SpanNameUserland, &tracing.CreateSpanOptions{
		Debug:              &tracing.SpanDebugData{Location: "apiv1.traces.commitSpan"},
		StartTime:          time.Unix(0, int64(s.StartTimeUnixNano)),
		EndTime:            time.Unix(0, int64(s.EndTimeUnixNano)),
		Parent:             parent,
		RawOtelSpanOptions: []trace.SpanStartOption{trace.WithAttributes(attrs...)},
	})
	if err != nil {
		return fmt.Errorf("failed to create span: %w", err)
	}

	return nil
}

func getInngestTraceRef(s *tracev1.Span) (*meta.SpanReference, error) {
	for _, kv := range s.Attributes {
		if kv.Key != "inngest.traceref" {
			continue
		}

		sr := &meta.SpanReference{}

		traceRefStr, err := url.QueryUnescape(kv.GetValue().GetStringValue())
		if err != nil {
			return nil, fmt.Errorf("failed to unescape trace reference: %w", err)
		}

		err = json.Unmarshal([]byte(traceRefStr), sr)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal span reference: %w", err)
		}

		err = sr.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid span reference: %w", err)
		}

		return sr, nil
	}

	return nil, fmt.Errorf("no span reference found in attributes")
}

func convertAttributes(attrs []*commonv1.KeyValue) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
		// Filter out any attributes that have our prefixes
		if strings.HasPrefix(kv.Key, meta.AttrKeyPrefix) {
			continue
		}

		out = append(out, attribute.KeyValue{
			Key:   attribute.Key(kv.Key),
			Value: convertAnyValue(kv.Value),
		})
	}
	return out
}

func convertAnyValue(v *commonv1.AnyValue) attribute.Value {
	if v == nil {
		return attribute.StringValue("")
	}
	switch val := v.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return attribute.StringValue(val.StringValue)
	case *commonv1.AnyValue_IntValue:
		return attribute.Int64Value(val.IntValue)
	case *commonv1.AnyValue_DoubleValue:
		return attribute.Float64Value(val.DoubleValue)
	case *commonv1.AnyValue_BoolValue:
		return attribute.BoolValue(val.BoolValue)
	default:
		return attribute.StringValue("")
	}
}

func convertResource(res *resourcev1.Resource) *resource.Resource {
	if res == nil {
		return resource.Empty()
	}
	attrs := convertAttributes(res.Attributes)
	r, _ := resource.New(context.Background(), resource.WithAttributes(attrs...))
	return r
}

func resourceServiceName(res *resource.Resource) string {
	if res == nil {
		return ""
	}
	for _, attr := range res.Attributes() {
		if string(attr.Key) == "service.name" {
			return attr.Value.AsString()
		}
	}
	return ""
}
