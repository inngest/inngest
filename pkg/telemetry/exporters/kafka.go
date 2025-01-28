package exporters

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/trace"
	"gocloud.dev/pubsub"
	"google.golang.org/protobuf/proto"
)

var (
	defaultMsgKey = "fn_id"
)

type kafkaSpanExporter struct {
	topic *pubsub.Topic
	key   string
}

type kafkaSpansExporterOpts struct {
	topic string
	key   string
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

func NewKafkaSpanExporter(ctx context.Context, opts ...KafkaSpansExporterOpts) (trace.SpanExporter, error) {
	conf := &kafkaSpansExporterOpts{}

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
		topic: topic,
		key:   conf.key,
	}, nil
}

func (e *kafkaSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	ctx = context.WithoutCancel(ctx)

	for _, sp := range spans {
		span, err := SpanToProto(ctx, sp)
		if err != nil {
			return err
		}

		byt, err := proto.Marshal(span)
		if err != nil {
			// TODO: log error
			return fmt.Errorf("error serializing span into binary: %w", err)
		}

		msg := &pubsub.Message{
			Metadata: map[string]string{},
			Body:     byt,
		}
		if e.key == defaultMsgKey {
			msg.Metadata[e.key] = span.GetId().GetFunctionId()
		}

		if err := e.topic.Send(ctx, &pubsub.Message{}); err != nil {
			// TODO: log error

			return fmt.Errorf("error publishing span to kafka: %w", err)
		}

		// TODO: return metrics for exported count
	}

	return nil
}

func (e *kafkaSpanExporter) Shutdown(ctx context.Context) error {
	return e.topic.Shutdown(ctx)
}
