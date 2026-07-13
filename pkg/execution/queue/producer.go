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

func withProducerClock(clock clockwork.Clock) producerOpt {
	return func(q *queueProducer) {
		q.clock = clock
	}
}

func withProducerKindToQueueMapping(mapping map[string]string) producerOpt {
	return func(q *queueProducer) {
		q.queueKindMapping = mapping
	}
}

func withProducerJobPromotion(enable bool) producerOpt {
	return func(q *queueProducer) {
		q.enableJobPromotion = enable
	}
}

func withProducerConditionalTracer(tracer trace.ConditionalTracer) producerOpt {
	return func(q *queueProducer) {
		q.conditionalTracer = tracer
	}
}

func newProducer(
	shards ShardRegistry,
	opts ...producerOpt,
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
