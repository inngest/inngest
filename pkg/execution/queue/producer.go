package queue

import (
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
)

// queueProducer implements the Producer interface, providing the ability to enqueue
// and requeue items.
type queueProducer struct {
	clock              clockwork.Clock
	queueKindMapping   map[string]string
	enableJobPromotion bool
	conditionalTracer  trace.ConditionalTracer

	shards ShardRegistry
}

type producerOpt func(*queueProducer)

type ProducerOpt = producerOpt

func WithProducerClock(clock clockwork.Clock) ProducerOpt {
	return withProducerClock(clock)
}

func withProducerClock(clock clockwork.Clock) producerOpt {
	return func(q *queueProducer) {
		q.clock = clock
	}
}

func WithProducerKindToQueueMapping(mapping map[string]string) ProducerOpt {
	return withProducerKindToQueueMapping(mapping)
}

func withProducerKindToQueueMapping(mapping map[string]string) producerOpt {
	return func(q *queueProducer) {
		q.queueKindMapping = mapping
	}
}

func WithProducerJobPromotion(enable bool) ProducerOpt {
	return withProducerJobPromotion(enable)
}

func withProducerJobPromotion(enable bool) producerOpt {
	return func(q *queueProducer) {
		q.enableJobPromotion = enable
	}
}

func WithProducerConditionalTracer(tracer trace.ConditionalTracer) ProducerOpt {
	return withProducerConditionalTracer(tracer)
}

func withProducerConditionalTracer(tracer trace.ConditionalTracer) producerOpt {
	return func(q *queueProducer) {
		q.conditionalTracer = tracer
	}
}

func NewProducer(
	shards ShardRegistry,
	opts ...ProducerOpt,
) Producer {
	q := &queueProducer{
		clock:             clockwork.NewRealClock(),
		conditionalTracer: trace.NoopConditionalTracer(),
		shards:            shards,
	}
	for _, opt := range opts {
		opt(q)
	}

	return q
}

func (q *queueProducer) Clock() clockwork.Clock {
	return q.clock
}
