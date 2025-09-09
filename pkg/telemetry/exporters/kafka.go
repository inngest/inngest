package exporters

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/proto"
)

var (
	defaultMsgKey       = "run_id"
	defaultMaxProduceMB = 30 // 30MB
)

type kafkaSpanExporter struct {
	client *kgo.Client
	key    string
}

type kafkaSpansExporterOpts struct {
	addrs           []string
	topic           string
	key             string
	autoCreateTopic bool
	scramAuth       *scram.Auth
	maxProduceMB    int
}

type KafkaSpansExporterOpts func(k *kafkaSpansExporterOpts)

func WithKafkaExporterBrokers(addrs []string) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.addrs = addrs
	}
}

func WithKafkaExporterTopic(topic, key string) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.topic = topic

		if key != "" {
			k.key = key
		}
	}
}

func WithKafkaExporterAutoCreateTopic() KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.autoCreateTopic = true
	}
}

func WithKafkaExporterScramAuth(user, pass string) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.scramAuth = &scram.Auth{
			User: user,
			Pass: pass,
		}
	}
}

func WithKafkaExporterMaxProduceMB(size int) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.maxProduceMB = size
	}
}

func NewKafkaSpanExporter(ctx context.Context, opts ...KafkaSpansExporterOpts) (trace.SpanExporter, error) {
	conf := &kafkaSpansExporterOpts{
		maxProduceMB: defaultMaxProduceMB,
		key:          defaultMsgKey,
	}

	for _, apply := range opts {
		apply(conf)
	}

	if len(conf.addrs) == 0 {
		return nil, fmt.Errorf("not kafka broker addresses provided")
	}

	if conf.topic == "" {
		return nil, fmt.Errorf("no topic provided for span exporter")
	}

	kclopts := []kgo.Opt{
		kgo.SeedBrokers(conf.addrs...),
		kgo.DefaultProduceTopic(conf.topic),
		kgo.RequiredAcks(kgo.AllISRAcks()), // Most durable with some perf hits

		kgo.ProducerBatchMaxBytes(int32(conf.maxProduceMB * 1024 * 1024)),
		// Increment metrics on data loss detection
		kgo.ProducerOnDataLossDetected(func(topic string, partition int32) {
			// record data loss when happened.
			metrics.IncrSpanExportDataLoss(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"producer":      "kafka",
					"topic":         topic,
					"partition":     partition,
					"trace_version": "v1",
				},
			})
		}),
	}

	if conf.autoCreateTopic {
		kclopts = append(kclopts, kgo.AllowAutoTopicCreation())
	}

	if conf.scramAuth != nil {
		kclopts = append(kclopts, kgo.SASL(conf.scramAuth.AsSha512Mechanism()))
	}

	cl, err := kgo.NewClient(kclopts...)
	if err != nil {
		return nil, fmt.Errorf("error initializing kafka client: %w", err)
	}
	if err := cl.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error establishing connection to kafka: %w", err)
	}

	return &kafkaSpanExporter{
		client: cl,
		key:    conf.key,
	}, nil
}

func (e *kafkaSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	ctx = context.WithoutCancel(ctx)

	l := logger.StdlibLogger(ctx)

	wg := sync.WaitGroup{}
	for _, sp := range spans {
		wg.Add(1)

		// TODO: skip publishing if delete tag is set
		// for _, attr := range sp.Attributes() {
		// 	// don't bother sending to Kafka if it's gonna be deleted anyways
		// 	if attr.Key == consts.OtelSysStepDelete && attr.Value.AsBool() {
		// 		wg.Done()
		// 		continue
		// 	}
		// }

		span, err := SpanToProto(ctx, sp)
		if err != nil {
			l.Error("error converting span to proto", "err", err)
			wg.Done()
			continue
		}

		id := span.GetId()
		byt, err := proto.Marshal(span)
		if err != nil {
			l.Error("error serializing span into binary",
				"err", err,
				"account_id", id.AccountId,
				"env_id", id.EnvId,
				"fn_id", id.FunctionId,
				"run_id", id.RunId,
			)
		}

		rec := &kgo.Record{Value: byt}
		switch e.key {
		case "account_id", "acct_id":
			rec.Key = []byte(id.GetAccountId())
		case "workspace_id", "ws_id", "env_id":
			rec.Key = []byte(id.GetEnvId())
		case "workflow_id", "wf_id", "function_id", "fn_id":
			rec.Key = []byte(id.GetFunctionId())
		case "run_id":
			switch {
			case id.GetRunId() != "":
				rec.Key = []byte(id.GetRunId())
			case id.GetFunctionId() != "":
				l.Warn("missing run_id, falling back to function_id", "span", sp)
				rec.Key = []byte(id.GetFunctionId())
			case id.GetEnvId() != "":
				l.Warn("missing run_id, falling back to env_id", "span", sp)
				rec.Key = []byte(id.GetEnvId())
			case id.GetAccountId() != "":
				l.Warn("missing run_id, falling back to acct_id", "span", sp)
				rec.Key = []byte(id.GetAccountId())
			default:
				l.Error("missing run_id, no other identifier to fallback to", "span", sp)
			}
		}

		e.client.Produce(ctx, rec, func(r *kgo.Record, err error) {
			defer wg.Done()

			status := "success"
			if err != nil {
				l.Error("error on producing span",
					"error", err,
					"account_id", id.AccountId,
					"env_id", id.EnvId,
					"fn_id", id.FunctionId,
					"run_id", id.RunId,
				)
				status = "error"

				if strings.Contains(err.Error(), consts.KafkaMsgTooLargeError) {
					batchSize := len(span.Events)
					evtPayloadSizes := make([]int, len(span.Events))
					for i, evt := range span.Events {
						evtPayloadSizes[i] = len(evt.Name)
					}
					l.Error("error on producing span MESSAGE_TOO_LARGE", "error", err, "span proto size", proto.Size(span), "marhsalled span proto size", len(byt), "span.output size", len(span.Output), "batchSize", batchSize, "evt payload sizes:", evtPayloadSizes)
				}
			}

			metrics.IncrSpanExportedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"producer":      "kafka",
					"status":        status,
					"trace_version": "v1",
				},
			})
		})

	}

	wg.Wait()
	return nil
}

func (e *kafkaSpanExporter) Shutdown(ctx context.Context) error {
	e.client.Close()
	return nil
}
