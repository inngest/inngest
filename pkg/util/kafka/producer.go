package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer interface {
	fmt.Stringer

	Produce(ctx context.Context, r *Record) error
}

type franzProducer struct {
	client *kgo.Client
}

func (f *franzProducer) String() string {
	return "franz-go"
}

// Produce implements Producer.
func (f *franzProducer) Produce(ctx context.Context, r *Record) error {
	errChan := make(chan error)

	kgoRecord := r.ToKgoRecord()

	// Attempt to produce, this will fail immediately if buffer is full
	// This call will not block if buffer is full!
	f.client.TryProduce(ctx, kgoRecord, func(r *kgo.Record, err error) {
		if err == nil {
			close(errChan)
			return
		}

		// This may fail for many reasons
		// - Buffer is full (cannot add more)
		// - Record in same buffer reached max retries
		// - Max retries reached for current record
		errChan <- err
	})

	// Record buffer size gauge
	metrics.GaugeKafkaProducerBufferSize(ctx, f.client.BufferedProduceRecords(), metrics.GaugeOpt{
		PkgName: "kafka",
		Tags: map[string]any{
			"producer_name": f.String(),
			"topic":         r.Topic,
		},
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err, open := <-errChan:
		if !open {
			return nil
		}
		return err
	}
}

func NewProducer(client *kgo.Client) (Producer, error) {
	// Max retries must be set to ensure we do not infinitely retry publishing in case of unavailable partition leader.
	// franz-go stores retries as int64 internally
	retries, ok := client.OptValue(kgo.RecordRetries).(int64)
	if !ok || retries == 0 {
		return nil, fmt.Errorf("must configure retries on franz-go client")
	}

	return &franzProducer{
		client: client,
	}, nil
}

type fallbackProducer struct {
	producers []Producer
}

// String implements Producer.
func (f *fallbackProducer) String() string {
	return "fallback"
}

// Produce implements Producer.
func (f *fallbackProducer) Produce(ctx context.Context, r *Record) error {
	l := logger.StdlibLogger(ctx)

	for i, p := range f.producers {
		start := time.Now()

		// Attempt to produce record
		err := p.Produce(ctx, r)

		status := "success"
		if err != nil {
			status = "error"
		}

		// Record duration metric
		metrics.HistogramKafkaProducerDuration(ctx, time.Since(start), metrics.HistogramOpt{
			PkgName: "kafka",
			Tags: map[string]any{
				"producer_name": p.String(),
				"attempt":       i,
				"status":        status,
				"topic":         r.Topic,
			},
		})

		if err != nil {
			// In case error is context timeout, return immediately
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}

			l.Error("failed to produce", "err", err, "producer", p.String())

			// If this is the last producer, return error
			if i == len(f.producers)-1 {
				return err
			}

			// Otherwise, continue with next producer
			continue
		}

		// Successfully produced, exit early
		return nil
	}

	return fmt.Errorf("not produced")
}

func NewFallbackProducer(producers ...Producer) Producer {
	return &fallbackProducer{
		producers: producers,
	}
}
