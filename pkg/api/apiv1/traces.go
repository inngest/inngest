package apiv1

import (
	"compress/gzip"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/run"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
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

func (a router) traces(w http.ResponseWriter, r *http.Request) {
	// Auth the app
	auth, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		respondError(w, r, http.StatusUnauthorized, "No auth found")
		return
	}

	// Check that the trace ID is valid and accessible to the app.
	// TODO The quickest call we can do to CH to check that the trace ID
	// existing with the account and workspace IDs we have

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

	rejectedSpans := a.convertOTLPAndSend(auth, req)

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
	w.Write(data)
}

func (a router) convertOTLPAndSend(auth apiv1auth.V1Auth, req *collecttrace.ExportTraceServiceRequest) (rejectedSpans int64) {
	ctx := context.Background()

	for _, rs := range req.ResourceSpans {
		res := convertResource(rs.Resource)

		for _, ss := range rs.ScopeSpans {
			scope := ss.Scope.GetName()

			for _, s := range ss.Spans {
				// To be valid, each span must have an "inngest.traceparent"
				// attribute
				tp, err := getInngestTraceparent(s)
				if err != nil {
					// If we can't find the traceparent, we can't create a
					// span. So let's skip it.
					rejectedSpans++
					continue
				}

				// TODO This needs to change to channels and each traceRoot will
				// also check the trace ID against the authed account and
				// workspace. Just need to fetch the root traceRoot. Once we have
				// that, we also have all attributes we need like run ID,
				// which we should set all of here.
				//
				// For now we do this synchronously while testing
				traceRoot, err := a.opts.TraceReader.GetTraceRoot(ctx, cqrs.TraceRunIdentifier{
					AccountID:   auth.AccountID(),
					WorkspaceID: auth.WorkspaceID(),
					TraceID:     tp.TraceID.String(),
				})
				if err != nil {
					// If we can't find the trace ID, we can't create a
					// span. So let's skip it.
					rejectedSpans++
					continue
				}

				opts := []run.SpanOpt{
					run.WithTraceID(tp.TraceID),
					run.WithSpanID(trace.SpanID(s.SpanId)),
					run.WithName(s.Name),
					run.WithSpanKind(trace.SpanKind(s.Kind)),
					run.WithScope(scope),
					run.WithLinks(convertLinks(s.Links)...),
					run.WithTimestamp(time.Unix(0, int64(s.StartTimeUnixNano))),
				}

				attrs := convertAttributes(s.Attributes)

				// Add built-in inngest attributes to the span (run ID etc)
				for k, v := range traceRoot.SpanAttributes {
					if _, ok := copyableAttrs[k]; ok {
						attrs = append(attrs, attribute.KeyValue{
							Key:   attribute.Key(k),
							Value: attribute.StringValue(v),
						})
					}
				}

				// Always mark the span as userland
				attrs = append(attrs, attribute.KeyValue{
					Key:   attribute.Key(consts.OtelScopeUserland),
					Value: attribute.BoolValue(true),
				})

				opts = append(opts, run.WithSpanAttributes(attrs...))

				if scope == "inngest" && s.Name == "inngest.execution" {
					// This is the "root" span created by an SDK, so let's
					// set its parent to our span ID
					opts = append(opts, run.WithParentSpanID(trace.SpanID(tp.SpanID)))
				}

				if len(s.ParentSpanId) == 8 {
					opts = append(opts, run.WithParentSpanID(trace.SpanID(s.ParentSpanId)))
				}

				if res != nil {
					opts = append(opts, run.WithServiceName(resourceServiceName(res)))
				}

				_, span := run.NewSpan(ctx, opts...)

				for _, e := range convertEvents(s.Events) {
					span.AddEvent(e.Name, trace.WithTimestamp(e.Time), trace.WithAttributes(e.Attributes...))
				}

				if s.Status != nil {
					span.SetStatus(traceStatusCode(s.Status.Code), s.Status.Message)
				}

				span.End(trace.WithTimestamp(time.Unix(0, int64(s.EndTimeUnixNano))))
			}
		}
	}

	return
}

func getInngestTraceparent(s *tracev1.Span) (*TraceParent, error) {
	for _, kv := range s.Attributes {
		if kv.Key == "inngest.traceparent" {
			// This is the traceparent attribute, so we can use it to get the
			// trace ID and span ID
			parts := strings.Split(kv.GetValue().GetStringValue(), "-")
			if len(parts) < 3 {
				return nil, fmt.Errorf("Invalid traceparent header format")
			}

			traceIDStr := parts[1]
			if len(traceIDStr) != 32 {
				return nil, fmt.Errorf("Invalid trace ID length %d", len(traceIDStr))
			}
			var traceID trace.TraceID
			_, err := hex.Decode(traceID[:], []byte(traceIDStr))
			if err != nil {
				return nil, fmt.Errorf("Invalid trace ID hex string: %v", err)
			}

			spanIDStr := parts[2]
			if len(spanIDStr) != 16 {
				return nil, fmt.Errorf("Invalid span ID length %d", len(spanIDStr))
			}
			var spanID trace.SpanID
			_, err = hex.Decode(spanID[:], []byte(spanIDStr))
			if err != nil {
				return nil, fmt.Errorf("Invalid span ID hex string: %v", err)
			}

			return &TraceParent{
				TraceID: traceID,
				SpanID:  spanID,
			}, nil
		}
	}

	return nil, fmt.Errorf("No traceparent attribute found")
}

func convertAttributes(attrs []*commonv1.KeyValue) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
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

func convertEvents(evts []*tracev1.Span_Event) []tracesdk.Event {
	out := make([]tracesdk.Event, 0, len(evts))
	for _, e := range evts {
		out = append(out, tracesdk.Event{
			Name:       e.Name,
			Time:       time.Unix(0, int64(e.TimeUnixNano)),
			Attributes: convertAttributes(e.Attributes),
		})
	}
	return out
}

func convertLinks(links []*tracev1.Span_Link) []tracesdk.Link {
	out := make([]tracesdk.Link, 0, len(links))
	for _, l := range links {
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID(l.TraceId),
			SpanID:  trace.SpanID(l.SpanId),
		})
		out = append(out, tracesdk.Link{
			SpanContext: sc,
			Attributes:  convertAttributes(l.Attributes),
		})
	}
	return out
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

func traceStatusCode(code tracev1.Status_StatusCode) codes.Code {
	switch code {
	case tracev1.Status_STATUS_CODE_ERROR:
		return codes.Error
	case tracev1.Status_STATUS_CODE_OK:
		return codes.Ok
	case tracev1.Status_STATUS_CODE_UNSET:
		return codes.Unset
	default:
		return codes.Unset
	}
}

var copyableAttrs = map[string]struct{}{
	consts.OtelSysAccountID:       {},
	consts.OtelSysWorkspaceID:     {},
	consts.OtelSysAppID:           {},
	consts.OtelSysFunctionID:      {},
	consts.OtelSysFunctionSlug:    {},
	consts.OtelSysFunctionVersion: {},
	consts.OtelAttrSDKRunID:       {},
	consts.OtelSysStepGroupID:     {},
}
