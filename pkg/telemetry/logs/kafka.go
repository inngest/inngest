package logs

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/sdk/log"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"google.golang.org/protobuf/proto"
)

const (
	pkgName = "logs.telemetry.inngest"
)

var (
	defaultMaxProduceMB = 30 // 30MB
)

type kafkaLogsExporter struct {
	client *kgo.Client
}

type kafkaLogsExporterOpts struct {
	addrs           []string
	topic           string
	key             string
	autoCreateTopic bool
	scramAuth       *scram.Auth
	maxProduceMB    int
}

type KafkaLogsExporterOpts func(k *kafkaLogsExporterOpts)

func WithKafkaExporterBrokers(addrs []string) KafkaLogsExporterOpts {
	return func(k *kafkaLogsExporterOpts) {
		k.addrs = addrs
	}
}

func WithKafkaExporterTopic(topic, key string) KafkaLogsExporterOpts {
	return func(k *kafkaLogsExporterOpts) {
		k.topic = topic

		if key != "" {
			k.key = key
		}
	}
}

func WithKafkaExporterAutoCreateTopic() KafkaLogsExporterOpts {
	return func(k *kafkaLogsExporterOpts) {
		k.autoCreateTopic = true
	}
}

func WithKafkaExporterScramAuth(user, pass string) KafkaLogsExporterOpts {
	return func(k *kafkaLogsExporterOpts) {
		k.scramAuth = &scram.Auth{
			User: user,
			Pass: pass,
		}
	}
}

func WithKafkaExporterMaxProduceMB(size int) KafkaLogsExporterOpts {
	return func(k *kafkaLogsExporterOpts) {
		k.maxProduceMB = size
	}
}

func NewKafkaLogExporter(ctx context.Context, opts ...KafkaLogsExporterOpts) (log.Exporter, error) {
	conf := &kafkaLogsExporterOpts{
		maxProduceMB: defaultMaxProduceMB,
	}

	for _, apply := range opts {
		apply(conf)
	}

	if len(conf.addrs) == 0 {
		return nil, fmt.Errorf("no kafka broker addresses provided")
	}

	if conf.topic == "" {
		return nil, fmt.Errorf("no topic provided for log exporter")
	}

	kclopts := []kgo.Opt{
		kgo.SeedBrokers(conf.addrs...),
		kgo.DefaultProduceTopic(conf.topic),
		kgo.RequiredAcks(kgo.AllISRAcks()), // Most durable with some perf hits
		kgo.ProducerBatchMaxBytes(int32(conf.maxProduceMB * 1024 * 1024)),
		// Increment metrics on data loss detection
		kgo.ProducerOnDataLossDetected(func(topic string, partition int32) {
			// record data loss when happened.
			metrics.IncrLogExportDataLoss(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"producer":  "kafka",
					"topic":     topic,
					"partition": partition,
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

	return &kafkaLogsExporter{
		client: cl,
	}, nil
}

func (e *kafkaLogsExporter) Export(ctx context.Context, records []log.Record) error {
	ctx = context.WithoutCancel(ctx)

	l := logger.StdlibLogger(ctx)

	// Transform to Protobuf
	// Logs data model: https://opentelemetry.io/docs/specs/otel/logs/data-model/

	wg := sync.WaitGroup{}
	for _, rec := range records {
		wg.Add(1)

		if rec.ObservedTimestamp().IsZero() {
			rec.SetObservedTimestamp(time.Now())
		}

		transformed := LogRecord(rec)

		byt, err := proto.Marshal(transformed)
		if err != nil {
			l.Error("error serializing log record into binary",
				"err", err,
			)
		}

		ts := rec.Timestamp()
		if ts.IsZero() {
			ts = rec.ObservedTimestamp()
		}

		// set a near-random key to ensure uniform distribution
		// across Kafka partitions
		key := ts.String()

		rec := &kgo.Record{
			Key:   []byte(key),
			Value: byt,
		}

		e.client.Produce(ctx, rec, func(r *kgo.Record, err error) {
			defer wg.Done()

			status := "success"
			if err != nil {
				l.Error("error on producing log record",
					"error", err,
				)
				status = "error"
			}

			metrics.IncrLogRecordExportedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"producer": "kafka",
					"status":   status,
				},
			})
		})

	}

	wg.Wait()
	return nil
}

func (e *kafkaLogsExporter) ForceFlush(ctx context.Context) error {
	return e.client.Flush(ctx)
}

func (e *kafkaLogsExporter) Shutdown(ctx context.Context) error {
	e.client.Close()
	return nil
}

type noopExporter struct{}

func (n noopExporter) Export(ctx context.Context, records []log.Record) error {
	// no-op
	return nil
}

func (n noopExporter) Shutdown(ctx context.Context) error {
	// no-op
	return nil
}

func (n noopExporter) ForceFlush(ctx context.Context) error {
	// no-op
	return nil
}

func NewNoopKafkaExporter() log.Exporter {
	return noopExporter{}
}
