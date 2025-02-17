package exporters

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	runv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type spanEvtType int

const (
	spanEvtTypeUnknown spanEvtType = iota
	spanEvtTypeEvent
	spanEvtTypeOutput
)

func SpanToProto(ctx context.Context, sp trace.ReadOnlySpan) (*runv2.Span, error) {
	ts := sp.StartTime()
	dur := sp.EndTime().Sub(ts)
	scope := sp.InstrumentationScope().Name

	var psid *string
	if sp.Parent().HasSpanID() {
		sid := sp.Parent().SpanID().String()
		psid = &sid
	}

	id, status, kind, attr, err := parseSpanAttributes(sp.Attributes())
	if err != nil {
		logger.StdlibLogger(ctx).Error("error parsing span attributes",
			"error", err,
			"spanAttr", sp.Attributes(),
		)
		return nil, err
	}

	links := make([]*runv2.SpanLink, len(sp.Links()))
	for i, spl := range sp.Links() {
		attrs := map[string]string{}
		for _, kv := range spl.Attributes {
			key := string(kv.Key)
			val := attributeValueAsString(kv.Value)
			attr[key] = val
		}

		links[i] = &runv2.SpanLink{
			TraceId:    spl.SpanContext.TraceID().String(),
			SpanId:     spl.SpanContext.SpanID().String(),
			TraceState: spl.SpanContext.TraceFlags().String(),
			Attributes: attrs,
		}
	}

	events, triggers, output, err := parseSpanEvents(sp.Events())
	if err != nil {
		logger.StdlibLogger(ctx).Error("error parsing span events",
			"error", err,
			"spanEvents", sp.Events(),
			"acctID", id.AccountId,
			"wsID", id.EnvId,
			"wfID", id.FunctionId,
			"runID", id.RunId,
		)
		return nil, err
	}

	return &runv2.Span{
		Id: id,
		Ctx: &runv2.SpanContext{
			TraceId:      sp.SpanContext().TraceID().String(),
			ParentSpanId: psid,
			SpanId:       sp.SpanContext().SpanID().String(),
		},
		Name:       sp.Name(),
		Kind:       kind,
		Status:     status,
		StatusCode: sp.Status().Code.String(),
		Scope:      scope,
		Timestamp:  timestamppb.New(ts),
		DurationMs: dur.Milliseconds(),
		Attributes: attr,
		Triggers:   triggers,
		Output:     output,
		Events:     events,
		Links:      links,
	}, nil
}

// parseSpanAttributes iterates through the span attributes and extract out data that
func parseSpanAttributes(spanAttr []attribute.KeyValue) (*runv2.SpanIdentifier, runv2.SpanStatus, runv2.SpanStepOp, map[string]string, error) {
	id := &runv2.SpanIdentifier{}
	attr := map[string]string{}

	var (
		kind   runv2.SpanStepOp
		status runv2.SpanStatus
	)

	for _, kv := range spanAttr {
		if kv.Valid() {
			key := string(kv.Key)
			val := attributeValueAsString(kv.Value)

			switch key {
			case consts.OtelSysAccountID:
				id.AccountId = val
			case consts.OtelSysWorkspaceID:
				id.EnvId = val
			case consts.OtelSysAppID:
				id.AppId = val
			case consts.OtelSysFunctionID:
				id.FunctionId = val
			case consts.OtelAttrSDKRunID:
				id.RunId = val
			case consts.OtelSysStepOpcode:
				kind = toProtoKind(val)
			case consts.OtelSysFunctionStatusCode:
				code := kv.Value.AsInt64()
				status = toProtoStatus(enums.RunCodeToStatus(code))
			}
			// TODO: move this into the default case so it doesn't record everything
			attr[key] = val
		}
	}

	return id, status, kind, attr, nil
}

func toProtoStatus(s enums.RunStatus) runv2.SpanStatus {
	switch s {
	case enums.RunStatusScheduled:
		return runv2.SpanStatus_SCHEDULED
	case enums.RunStatusRunning:
		return runv2.SpanStatus_RUNNING
	case enums.RunStatusCompleted:
		return runv2.SpanStatus_COMPLETED
	case enums.RunStatusFailed, enums.RunStatusOverflowed:
		return runv2.SpanStatus_FAILED
	case enums.RunStatusCancelled:
		return runv2.SpanStatus_CANCELLED
	default:
		return runv2.SpanStatus_UNKNOWN
	}
}

func toProtoKind(code string) runv2.SpanStepOp {
	o, err := enums.OpcodeString(code)
	if err != nil {
		return runv2.SpanStepOp_NONE
	}

	switch o {
	case enums.OpcodeInvokeFunction:
		return runv2.SpanStepOp_INVOKE
	case enums.OpcodeWaitForEvent:
		return runv2.SpanStepOp_WAIT_FOR_EVENT
	case enums.OpcodeSleep:
		return runv2.SpanStepOp_SLEEP
	case enums.OpcodeStepRun, enums.OpcodeStep:
		return runv2.SpanStepOp_STEP
	case enums.OpcodeStepError:
		return runv2.SpanStepOp_STEP_ERROR
	case enums.OpcodeAIGateway:
		return runv2.SpanStepOp_AI_GATEWAY

	default:
		return runv2.SpanStepOp_RUN
	}
}

// parseSpanEvents iterates through the otel span events and extract out
// embedded data
// - run triggers (events and crons)
// - output
func parseSpanEvents(spanEvents []trace.Event) ([]*runv2.SpanEvent, []*runv2.Trigger, []byte, error) {
	events := []*runv2.SpanEvent{} // actual span events
	triggers := []*runv2.Trigger{}
	var output []byte

	for _, evt := range spanEvents {
		attr := map[string]string{}
		var evtID string

		// iterates over the list of attributes in this span event
		// and set the type.
		//
		// NOTE: event data and outputs should NEVER be in the same span event
		var typ spanEvtType
		for _, kv := range evt.Attributes {
			if kv.Valid() {
				key := string(kv.Key)
				val := attributeValueAsString(kv.Value)

				switch key {
				case consts.OtelSysEventData:
					typ = spanEvtTypeEvent
				case consts.OtelSysEventInternalID:
					typ = spanEvtTypeEvent
					evtID = kv.Value.AsString()
				case consts.OtelSysFunctionOutput, consts.OtelSysStepOutput:
					typ = spanEvtTypeOutput
				}
				// TODO: move this into the default case section
				attr[key] = val
			}
		}

		// update the relevant data based on type found
		switch typ {
		case spanEvtTypeEvent:
			triggers = append(triggers, &runv2.Trigger{
				InternalId: evtID,
				Body:       []byte(evt.Name),
			})
		case spanEvtTypeOutput:
			output = []byte(evt.Name)
		}

		// TODO: should be moved into the default case for switch
		events = append(events, &runv2.SpanEvent{
			Name:       evt.Name,
			Timestamp:  timestamppb.New(evt.Time),
			Attributes: attr,
		})
	}

	return events, triggers, output, nil
}

func attributeValueAsString(v attribute.Value) string {
	switch v.Type() {
	case attribute.BOOL:
		return fmt.Sprintf("%t", v.AsBool())
	case attribute.INT64:
		return fmt.Sprintf("%d", v.AsInt64())
	case attribute.STRING:
		return v.AsString()
	case attribute.FLOAT64:
		return fmt.Sprintf("%f", v.AsFloat64())
	default:
		logger.StdlibLogger(context.TODO()).Warn("not supported attribute value type",
			"value", v,
			"type", v.Type().String(),
		)
		return v.AsString()
	}
}
