package exporters

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/sourcegraph/conc/pool"
	"go.opentelemetry.io/otel/sdk/trace"
	"gocloud.dev/pubsub"
	"google.golang.org/protobuf/proto"
)

var (
	defaultMsgKey   = "fn_id"
	defaultPoolSize = 500
)

type kafkaSpanExporter struct {
	topic    *pubsub.Topic
	key      string
	poolSize int
}

type kafkaSpansExporterOpts struct {
	topic    string
	key      string
	poolSize int
}

type KafkaSpansExporterOpts func(k *kafkaSpansExporterOpts)

func WithKafkaExporterTopic(topic, key string) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.topic = topic

		if key != "" {
			k.key = key
		}
	}
}

func WithKafkaExporterPoolSize(size int) KafkaSpansExporterOpts {
	return func(k *kafkaSpansExporterOpts) {
		k.poolSize = size
	}
}

func NewKafkaSpanExporter(ctx context.Context, opts ...KafkaSpansExporterOpts) (trace.SpanExporter, error) {
	conf := &kafkaSpansExporterOpts{
		poolSize: defaultPoolSize,
	}

	for _, apply := range opts {
		apply(conf)
	}

	if conf.topic == "" {
		return nil, fmt.Errorf("no topic provided for span exporter")
	}

	if conf.key == "" {
		conf.key = defaultMsgKey
	}

	// construct topic URL
	topicURL := fmt.Sprintf("kafka://%s?key_name=%s", conf.topic, conf.key)

	// Open kafka topic with URL
	//
	// NOTE: the set of kafka brokers must be set in an environment variable KAFKA_BROKERS
	topic, err := pubsub.OpenTopic(ctx, topicURL)
	if err != nil {
		return nil, fmt.Errorf("error opening topic with kafka for span exporter: %w", err)
	}

	return &kafkaSpanExporter{
		topic:    topic,
		key:      conf.key,
		poolSize: conf.poolSize,
	}, nil
}

func (e *kafkaSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	ctx = context.WithoutCancel(ctx)

	l := logger.StdlibLogger(ctx)

	errp := pool.New().WithErrors().WithMaxGoroutines(e.poolSize)

	for _, sp := range spans {
		sp := sp

		errp.Go(func() error {
			span, err := SpanToProto(ctx, sp)
			if err != nil {
				l.Error("error converting span to proto", "err", err)
				return fmt.Errorf("error converting span to proto: %w", err)
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

				return fmt.Errorf("error serialzing span into binary: %w", err)
			}

			msg := &pubsub.Message{
				Metadata: map[string]string{},
				Body:     byt,
			}
			if e.key == defaultMsgKey {
				msg.Metadata[e.key] = id.GetFunctionId()
			}

			status := "success"
			if err := e.topic.Send(ctx, &pubsub.Message{}); err != nil {
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

			return nil
		})
	}

	return errp.Wait()
}

func (e *kafkaSpanExporter) Shutdown(ctx context.Context) error {
	return e.topic.Shutdown(ctx)
}
