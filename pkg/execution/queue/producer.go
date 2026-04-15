package queue

import (
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
)

type ProducerOpts struct {
	clock              clockwork.Clock
	queueKindMapping   map[string]string
	enableJobPromotion bool
	conditionalTracer  trace.ConditionalTracer
}

// QueueProducer implements the Producer interface, providing the ability to enqueue
// and requeue items.
type QueueProducer struct {
	clock              clockwork.Clock
	queueKindMapping   map[string]string
	enableJobPromotion bool
	ConditionalTracer  trace.ConditionalTracer

	shards ShardRegistry
}

func NewProducer(
	shards ShardRegistry,
	opts ProducerOpts,
) *QueueProducer {
	return &QueueProducer{
		clock:              opts.clock,
		queueKindMapping:   opts.queueKindMapping,
		enableJobPromotion: opts.enableJobPromotion,
		ConditionalTracer:  opts.conditionalTracer,
		shards:             shards,
	}
}

func (q *QueueProducer) Clock() clockwork.Clock {
	return q.clock
}
