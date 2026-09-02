package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/oklog/ulid/v2"
)

type MetadataSpanAttrOpts func(cfg *MetadataSpanConfig)

type MetadataSpanConfig struct {
	Attrs *meta.SerializableAttrs

	// SyncListeners are notified synchronously, via OnMetadataEntry, right
	// after this span is created -- mirroring
	// execution.SyncLifecycleListener's dual-write hooks for run/step/
	// userland spans (see pkg/execution/dualwrite). Nil/empty is a no-op;
	// only a caller wiring DuckDB dual-write sets this today.
	SyncListeners []execution.SyncLifecycleListener
}

// WithMetadataSyncListeners registers listeners to notify synchronously
// after a metadata span is created -- see MetadataSpanConfig.SyncListeners.
func WithMetadataSyncListeners(ls ...execution.SyncLifecycleListener) MetadataSpanAttrOpts {
	return func(cfg *MetadataSpanConfig) {
		cfg.SyncListeners = append(cfg.SyncListeners, ls...)
	}
}

func CreateMetadataSpan(ctx context.Context, tracerProvider TracerProvider, parent *meta.SpanReference, location, pkgName string, stateMetadata *statev2.Metadata, spanMetadata metadata.Structured, scope metadata.Scope, opts ...MetadataSpanAttrOpts) (*meta.SpanReference, error) {
	values, err := spanMetadata.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	return CreateMetadataSpanFromValues(ctx, tracerProvider, parent, location, pkgName, stateMetadata, spanMetadata.Kind(), spanMetadata.Op(), values, scope, opts...)
}

// CreateMetadataSpanFromValues creates a metadata span from pre-serialized values,
// avoiding redundant serialization when the caller has already called Serialize.
func CreateMetadataSpanFromValues(ctx context.Context, tracerProvider TracerProvider, parent *meta.SpanReference, location, pkgName string, stateMetadata *statev2.Metadata, kind metadata.Kind, op metadata.Opcode, values metadata.Values, scope metadata.Scope, opts ...MetadataSpanAttrOpts) (*meta.SpanReference, error) {
	// Every metadata span, regardless of caller, passes through here — so
	// this is the single chokepoint to backfill EstimatedCost for
	// "inngest.ai" metadata that arrived without one (e.g. submitted
	// directly via inngest.metadata.update or the AddRunMetadata API,
	// bypassing the AIMetadata-producing extractors). A no-op when
	// EstimatedCost is already set.
	if kind == extractors.KindInngestAI {
		extractors.BackfillEstimatedCostInValues(values)
	}

	spanSize := values.Size()

	// Per-span size limit
	if spanSize > consts.MaxMetadataSpanSize {
		return nil, metadata.ErrMetadataSpanTooLarge
	}

	// Per-run cumulative size limit. Skip when stateMetadata is nil.
	//
	// TryAddMetadataSize atomically checks the limit and increments the
	// counter under a mutex, which is safe for concurrent access from
	// parallel step handlers (handleGeneratorGroup).
	if stateMetadata != nil {
		if !stateMetadata.Metrics.TryAddMetadataSize(spanSize, consts.MaxRunMetadataSize) {
			return nil, metadata.ErrRunMetadataSizeExceeded
		}
	}

	attrs := RawMetadataAttrs(kind, values, op)
	meta.AddAttr(attrs, meta.Attrs.MetadataScope, &scope)

	cfg := MetadataSpanConfig{
		Attrs: attrs,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	kindTag := kind.String()
	if kind.IsUser() {
		kindTag = fmt.Sprintf("%s*", metadata.KindPrefixUserland)
	}

	metrics.IncrMetadataSpansTotal(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"kind": kindTag,
		},
	})
	ref, err := tracerProvider.CreateSpan(
		ctx,
		meta.SpanNameMetadata,
		&CreateSpanOptions{
			Debug:      &SpanDebugData{Location: location},
			Parent:     parent,
			Metadata:   stateMetadata,
			Attributes: cfg.Attrs,

			// Set the dynamic_span_id from (parent, kind) so every
			// metadata emission of this kind under this parent aggregates together.
			DynamicSpanIDOverride: DeterministicSpanID(MetadataSpanIDSeed(parent.DynamicSpanID, kind)).String(),
		},
	)
	if err != nil {
		// Roll back the optimistic size increment on span creation failure.
		if stateMetadata != nil {
			stateMetadata.Metrics.RollbackMetadataSize(spanSize)
		}
		return nil, err
	}

	if len(cfg.SyncListeners) > 0 {
		if entry, ok := buildSyncMetadataEntry(cfg.Attrs, parent, stateMetadata, kind, scope, values); ok {
			for _, l := range cfg.SyncListeners {
				l.OnMetadataEntry(ctx, entry)
			}
		} else {
			logger.StdlibLogger(ctx).Error("tracing: metadata span sync dispatch missing run ID", "location", location, "kind", kind.String())
		}
	}

	return ref, nil
}

// buildSyncMetadataEntry assembles an execution.MetadataEntry for
// OnMetadataEntry from the same inputs CreateMetadataSpanFromValues already
// has -- tenant/run identity comes from stateMetadata when the caller
// provided one (the executor and AddRunMetadata always do), falling back to
// attrs (already resolved by the time this runs, regardless of whether the
// caller set them via stateMetadata or an option like addTenantIDs) for the
// one call site that has neither (pkg/api/apiv1/traces.go's
// commitSpanMetadata). ok is false when neither source yields a run ID,
// since a row with no run ID can't be joined onto anything.
func buildSyncMetadataEntry(attrs *meta.SerializableAttrs, parent *meta.SpanReference, stateMetadata *statev2.Metadata, kind metadata.Kind, scope metadata.Scope, values metadata.Values) (execution.MetadataEntry, bool) {
	entry := execution.MetadataEntry{
		Parent:    parent,
		Kind:      kind,
		Scope:     scope,
		Values:    values,
		CreatedAt: time.Now(),
	}

	if stateMetadata != nil {
		entry.AccountID = stateMetadata.ID.Tenant.AccountID
		entry.EnvID = stateMetadata.ID.Tenant.EnvID
		entry.AppID = stateMetadata.ID.Tenant.AppID
		entry.FunctionID = stateMetadata.ID.FunctionID
		entry.RunID = stateMetadata.ID.RunID
	} else {
		if v, ok := meta.GetAttr(attrs, meta.Attrs.AccountID); ok && v != nil {
			entry.AccountID = *v
		}
		if v, ok := meta.GetAttr(attrs, meta.Attrs.EnvID); ok && v != nil {
			entry.EnvID = *v
		}
		if v, ok := meta.GetAttr(attrs, meta.Attrs.AppID); ok && v != nil {
			entry.AppID = *v
		}
		if v, ok := meta.GetAttr(attrs, meta.Attrs.FunctionID); ok && v != nil {
			entry.FunctionID = *v
		}
		v, ok := meta.GetAttr(attrs, meta.Attrs.RunID)
		if !ok || v == nil || *v == (ulid.ULID{}) {
			return execution.MetadataEntry{}, false
		}
		entry.RunID = *v
	}

	// Step identity is carried on attrs regardless of whether stateMetadata
	// is set -- every caller that knows it's operating on a step adds it via
	// a MetadataSpanAttrOpts (e.g. the executor's createMetadataSpanOnParent,
	// apiv1/metadata.go's addTenantIDs), independently of tenant/run
	// identity's source above.
	if v, ok := meta.GetAttr(attrs, meta.Attrs.StepID); ok && v != nil {
		entry.StepID = *v
	}
	if v, ok := meta.GetAttr(attrs, meta.Attrs.StepUserlandIndex); ok && v != nil {
		entry.StepIndex = v
	}
	if v, ok := meta.GetAttr(attrs, meta.Attrs.StepAttempt); ok && v != nil {
		entry.StepAttempt = v
	}

	return entry, true
}

func RawMetadataAttrs(kind metadata.Kind, values metadata.Values, op metadata.Opcode) *meta.SerializableAttrs {
	rawAttrs := meta.NewAttrSet()

	meta.AddAttr(rawAttrs, meta.Attrs.MetadataKind, &kind)
	meta.AddAttr(rawAttrs, meta.Attrs.Metadata, &values)
	meta.AddAttr(rawAttrs, meta.Attrs.MetadataOp, &op)

	return rawAttrs
}

func MetadataAttrs(metadata metadata.Structured) (*meta.SerializableAttrs, error) {
	rawMetadata, err := metadata.Serialize()
	if err != nil {
		return nil, err
	}

	return RawMetadataAttrs(metadata.Kind(), rawMetadata, metadata.Op()), nil
}

func MetadataSpanIDSeed(parentID string, kind metadata.Kind) []byte {
	return fmt.Appendf(nil, "%s-metadata-%s", parentID, kind)
}
