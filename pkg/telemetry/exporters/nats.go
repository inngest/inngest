package exporters

import (
	"context"
	"fmt"
	"sync"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub/broker"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	runv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type spanEvtType int

const (
	spanEvtTypeUnknown spanEvtType = iota
	spanEvtTypeEvent
	spanEvtTypeOutput
)

// NATS span exporter
type natsSpanExporter struct {
	subjects []string
	conn     *broker.NatsConnector
}

type NatsExporterOpts struct {
	// The subjects this exporter will be publishing the spans to
	Subjects []string
	// Comma delimited URLs of the NATS server to use
	URLs string
	// The path of the nkey file to be used for authentication
	NkeyFile string
	// The credentials file to be used for authentication
	CredsFile string
}

// NewNATSSpanExporter creates an otel compatible exporter that ships the spans to NATS
func NewNATSSpanExporter(ctx context.Context, opts *NatsExporterOpts) (trace.SpanExporter, error) {
	if opts == nil {
		return nil, fmt.Errorf("nats exporter setup options unavailable")
	}

	connOpts := []nats.Option{}
	// attempt to parse nkey file is the option was passed in
	if opts.NkeyFile != "" {
		auth, err := nats.NkeyOptionFromSeed(opts.NkeyFile)
		if err != nil {
			return nil, fmt.Errorf("error parsing nkey file for NATS: %w", err)
		}
		connOpts = append(connOpts, auth)
	}

	// Use chain credentials file for auth
	if opts.CredsFile != "" {
		auth := nats.UserCredentials(opts.CredsFile)
		connOpts = append(connOpts, auth)
	}

	conn, err := broker.NewNATSConnector(ctx, broker.NatsConnOpt{
		Name:      "run-span-exporter",
		URLS:      opts.URLs,
		JetStream: true,
		Opts:      connOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("error setting up nats: %w", err)
	}

	return &natsSpanExporter{
		subjects: opts.Subjects,
		conn:     conn,
	}, nil
}

func (e *natsSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	ctx = context.WithoutCancel(ctx)
	wg := sync.WaitGroup{}

	// Expect jetstream to be enabled
	js, err := e.conn.JSConn()
	if err != nil {
		return err
	}
	// publish to all subjects defined
	for _, subj := range e.subjects {
		for _, sp := range spans {
			wg.Add(1)

			go func(ctx context.Context, sub string, sp trace.ReadOnlySpan) {
				defer wg.Done()

				ts := sp.StartTime()
				dur := sp.EndTime().Sub(ts)
				scope := sp.InstrumentationScope().Name

				var psid *string
				if sp.Parent().HasSpanID() {
					sid := sp.Parent().SpanID().String()
					psid = &sid
				}

				id, status, kind, attr, err := e.parseSpanAttributes(sp.Attributes())
				if err != nil {
					logger.StdlibLogger(ctx).Error("error parsing span attributes",
						"error", err,
						"spanAttr", sp.Attributes(),
					)
				}

				links := make([]*runv2.SpanLink, len(sp.Links()))
				for i, spl := range sp.Links() {
					attrs := map[string]string{}
					for _, kv := range spl.Attributes {
						key := string(kv.Key)
						val := e.attributeValueAsString(kv.Value)
						attr[key] = val
					}

					links[i] = &runv2.SpanLink{
						TraceId:    spl.SpanContext.TraceID().String(),
						SpanId:     spl.SpanContext.SpanID().String(),
						TraceState: spl.SpanContext.TraceFlags().String(),
						Attributes: attrs,
					}
				}

				events, triggers, output, err := e.parseSpanEvents(sp.Events())
				if err != nil {
					logger.StdlibLogger(ctx).Error("error parsing span events",
						"error", err,
						"spanEvents", sp.Events(),
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)
				}

				span := &runv2.Span{
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
				}

				// serialize it into bytes
				byt, err := proto.Marshal(span)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error serializing span to protobuf",
						"error", err,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)
					return
				}

				// Use async publish to increase throughput
				fack, err := js.PublishAsync(sub, byt)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error on async publish to nats stream",
						"error", err,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)
					return
				}

				pstatus := "unknown"
				select {
				case <-fack.Ok():
					pstatus = "success"
				case err := <-fack.Err():
					pstatus = "error"

					logger.StdlibLogger(ctx).Error("error with async publish to nats stream",
						"error", err,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)
				}

				metrics.IncrSpanExportedCounter(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"subject": sub,
						"status":  pstatus,
					},
				})
			}(ctx, subj, sp)
		}
	}

	wg.Wait()
	return nil
}

func (e *natsSpanExporter) Shutdown(ctx context.Context) error {
	return e.conn.Shutdown(ctx)
}

// parseSpanAttributes iterates through the span attributes and extract out data that
func (e *natsSpanExporter) parseSpanAttributes(spanAttr []attribute.KeyValue) (*runv2.SpanIdentifier, runv2.SpanStatus, runv2.SpanStepOp, map[string]string, error) {
	id := &runv2.SpanIdentifier{}
	attr := map[string]string{}

	var (
		kind   runv2.SpanStepOp
		status runv2.SpanStatus
	)

	for _, kv := range spanAttr {
		if kv.Valid() {
			key := string(kv.Key)
			val := e.attributeValueAsString(kv.Value)

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
				kind = e.toProtoKind(val)
			case consts.OtelSysFunctionStatusCode:
				code := kv.Value.AsInt64()
				status = e.toProtoStatus(enums.RunCodeToStatus(code))
			}
			// TODO: move this into the default case so it doesn't record everything
			attr[key] = val
		}
	}

	return id, status, kind, attr, nil
}

func (e *natsSpanExporter) toProtoStatus(s enums.RunStatus) runv2.SpanStatus {
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

func (e *natsSpanExporter) toProtoKind(code string) runv2.SpanStepOp {
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

	default:
		return runv2.SpanStepOp_RUN
	}
}

// parseSpanEvents iterates through the otel span events and extract out
// embedded data
// - run triggers (events and crons)
// - output
func (e *natsSpanExporter) parseSpanEvents(spanEvents []trace.Event) ([]*runv2.SpanEvent, []*runv2.Trigger, []byte, error) {

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
				val := e.attributeValueAsString(kv.Value)

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

func (e *natsSpanExporter) attributeValueAsString(v attribute.Value) string {
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
