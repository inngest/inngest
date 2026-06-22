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

type ProducerOpt func(*queueProducer)

func WithProducerClock(clock clockwork.Clock) ProducerOpt {
	return func(q *queueProducer) {
		q.clock = clock
	}
}

func WithProducerKindToQueueMapping(mapping map[string]string) ProducerOpt {
	return func(q *queueProducer) {
		q.queueKindMapping = mapping
	}
}

func WithProducerJobPromotion(enable bool) ProducerOpt {
	return func(q *queueProducer) {
		q.enableJobPromotion = enable
	}
}

func WithProducerConditionalTracer(tracer trace.ConditionalTracer) ProducerOpt {
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
