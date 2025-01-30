package exporters

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"go.opentelemetry.io/otel/sdk/trace"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/batcher"
	"gocloud.dev/pubsub/kafkapubsub"
	"google.golang.org/protobuf/proto"
)

var (
	defaultMsgKey           = "fn_id"
	defaultHandlerBatchSize = 500
	defaultHandlerNum       = 500
	defaultMaxMsgBytes      = 1024 * 1024 * 30 // 30MB
)

type kafkaSpanExporter struct {
	topic    *pubsub.Topic
	key      string
	poolSize int
}

type kafkaSpansExporterOpts struct {
	addrs            []string
	topic            string
	key              string
	handlerNum       int
	handlerBatchSize int
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

func WithKafkaSendHandlerNum(n int) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.handlerNum = n
	}
}

func WithKafkaSendHandlerBatchSize(n int) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.handlerBatchSize = n
	}
}

func NewKafkaSpanExporter(ctx context.Context, opts ...KafkaSpansExporterOpts) (trace.SpanExporter, error) {
	conf := &kafkaSpansExporterOpts{
		handlerNum:       defaultHandlerNum,
		handlerBatchSize: defaultHandlerBatchSize,
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

	// Configure kafka connection options
	kconf := kafkapubsub.MinimalConfig()
	kconf.Producer.MaxMessageBytes = defaultMaxMsgBytes
	kconf.Producer.RequiredAcks = sarama.WaitForAll // Most durable
	kconf.Producer.Compression = sarama.CompressionZSTD
	kconf.Producer.CompressionLevel = 6

	kopts := kafkapubsub.TopicOptions{
		BatcherOptions: batcher.Options{
			MaxHandlers:  conf.handlerNum,
			MaxBatchSize: conf.handlerBatchSize,
		},
	}

	if conf.key != "" {
		conf.key = defaultMsgKey
		kopts.KeyName = defaultMsgKey
	}

	topic, err := kafkapubsub.OpenTopic(conf.addrs, kconf, conf.topic, &kopts)
	if err != nil {
		return nil, fmt.Errorf("error opening topic with kafka for span exporter: %w", err)
	}

	return &kafkaSpanExporter{
		topic: topic,
		key:   conf.key,
	}, nil
}

func (e *kafkaSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	ctx = context.WithoutCancel(ctx)

	l := logger.StdlibLogger(ctx)

	wg := sync.WaitGroup{}

	for _, sp := range spans {
		wg.Add(1)

		go func(ctx context.Context, sp trace.ReadOnlySpan) {
			defer wg.Done()

			span, err := SpanToProto(ctx, sp)
			if err != nil {
				l.Error("error converting span to proto", "err", err)
			}

			id := span.GetId()
			byt, err := proto.Marshal(span)
			if err != nil {
				l.Error("error serializing span into binary",
					"err", err,
					"acctID", id.AccountId,
					"wsID", id.EnvId,
					"fnID", id.FunctionId,
					"runID", id.RunId,
				)
			}

			msg := &pubsub.Message{
				Metadata: map[string]string{},
				Body:     byt,
			}
			switch e.key {
			case "account_id", "acct_id":
				msg.Metadata[e.key] = id.GetAccountId()
			case "workspace_id", "ws_id", "env_id":
				msg.Metadata[e.key] = id.GetEnvId()
			case "workflow_id", "wf_id", "function_id", "fn_id":
				msg.Metadata[e.key] = id.GetFunctionId()
			case "run_id":
				msg.Metadata[e.key] = id.GetRunId()
			}

			status := "success"
			if err := e.topic.Send(ctx, msg); err != nil {
				l.Error("error publishing span to kafka",
					"err", err,
					"acctID", id.AccountId,
					"wsID", id.EnvId,
					"fnID", id.FunctionId,
					"runID", id.RunId,
				)

				status = "error"
				// TODO: should attempt error handling or resending it
			}

			metrics.IncrSpanExportedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"producer": "kafka",
					"status":   status,
				},
			})

			return
		}(ctx, sp)
	}

	wg.Wait()
	return nil
}

func (e *kafkaSpanExporter) Shutdown(ctx context.Context) error {
	return e.topic.Shutdown(ctx)
}
