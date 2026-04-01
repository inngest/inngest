package tracing

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/constraintlifecycle"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

type constraintNotifier struct {
	tp TracerProvider
}

func NewConstraintNotifier(tp TracerProvider) constraintlifecycle.Notifier {
	return &constraintNotifier{tp: tp}
}

func (n *constraintNotifier) OnConstraintHit(ctx context.Context, constraint enums.QueueConstraint, itemMetadata map[string]any) error {
	spanRef := spanRefFromMetadata(itemMetadata)
	if spanRef == nil {
		return nil
	}
	return n.tp.UpdateSpan(ctx, &UpdateSpanOptions{
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.ConstraintHit, &constraint),
		),
		Debug:      &SpanDebugData{Location: "queue.Process.ConstraintHit"},
		TargetSpan: spanRef,
	})
}

func spanRefFromMetadata(metadata map[string]any) *meta.SpanReference {
	if metadata == nil {
		return nil
	}
	carrier, ok := metadata[meta.PropagationKey]
	if !ok {
		return nil
	}
	str, ok := carrier.(string)
	if !ok {
		return nil
	}
	var out meta.SpanReference
	if err := json.Unmarshal([]byte(str), &out); err != nil {
		return nil
	}
	return &out
}
