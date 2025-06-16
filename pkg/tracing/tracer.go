package tracing

import (
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider defines the interface for tracing providers.
type TracerProvider interface {
	CreateSpan(name string, opts *CreateSpanOptions) (*meta.SpanReference, error)
	CreateDroppableSpan(name string, opts *CreateSpanOptions) (*DroppableSpan, error)
	UpdateSpan(opts *UpdateSpanOptions) error
}

type DroppableSpan struct {
	span trace.Span
	Ref  *meta.SpanReference
}

type CreateSpanOptions struct {
	Carriers    []map[string]any
	FollowsFrom *meta.SpanReference
	Location    string
	Metadata    *statev2.Metadata
	Parent      *meta.SpanReference
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
}

type UpdateSpanOptions struct {
	Carrier     map[string]string
	EndTime     time.Time
	Location    string
	Metadata    *statev2.Metadata
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
	Status      enums.StepStatus
	TargetSpan  *meta.SpanReference
}
