package metadatawriter

import (
	"context"
	"errors"
	"fmt"

	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
)

var ErrLoadMetadata = errors.New("unable to load run metadata")
var ErrMissingMetadataItems = errors.New("missing metadata items")
var ErrMissingParentResolver = errors.New("missing metadata parent resolver")

// MissingStateLoader reconstructs run metadata when live execution state is gone.
// Callers can leave this nil to either reject missing state or, when
// AllowMissingState is true, use request-local metadata.
type MissingStateLoader func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error)

// ParentResolver maps loaded run metadata to the metadata span's parent and
// scope. API layers own target semantics; Writer owns persistence.
type ParentResolver func(md *statev2.Metadata) (*meta.SpanReference, metadata.Scope)

type Item struct {
	Metadata metadata.Update
	Parent   ParentResolver
}

func NewItems(updates []metadata.Update, parent ParentResolver) []Item {
	items := make([]Item, 0, len(updates))
	for _, update := range updates {
		items = append(items, Item{
			Metadata: update,
			Parent:   parent,
		})
	}
	return items
}

type Writer struct {
	State              statev2.RunService
	TracerProvider     tracing.TracerProvider
	MissingStateLoader MissingStateLoader
	PkgName            string

	// AllowMissingState preserves SDK metadata behavior for finalized runs
	// whose execution state has already been deleted. Missing state uses a
	// request-local metadata value so the write still enforces the cumulative size
	// limit within the request.
	AllowMissingState bool
}

type WriteRequest struct {
	ID       statev2.ID
	Items    []Item
	Location string
}

func (w Writer) Write(ctx context.Context, req WriteRequest) error {
	// Load run metadata to enforce the per-run cumulative size limit against
	// metadata that already exists in the run, not just this request.
	stateMetadata, loadedFromState, err := w.loadMetadata(ctx, req.ID)
	if err != nil {
		return err
	}
	statev2.InitConfig(&stateMetadata.Config)

	if len(req.Items) == 0 {
		return ErrMissingMetadataItems
	}
	for _, item := range req.Items {
		if item.Parent == nil {
			return ErrMissingParentResolver
		}
	}

	tracerProvider := w.TracerProvider
	if tracerProvider == nil {
		tracerProvider = tracing.NewNoopTracerProvider()
	}

	addTenantIDs := func(cfg *tracing.MetadataSpanConfig) {
		meta.AddAttr(cfg.Attrs, meta.Attrs.AccountID, util.ToPtr(stateMetadata.ID.Tenant.AccountID))
		meta.AddAttr(cfg.Attrs, meta.Attrs.EnvID, util.ToPtr(stateMetadata.ID.Tenant.EnvID))
		meta.AddAttr(cfg.Attrs, meta.Attrs.FunctionID, &stateMetadata.ID.FunctionID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.RunID, &stateMetadata.ID.RunID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.AppID, &stateMetadata.ID.Tenant.AppID)
	}

	for _, item := range req.Items {
		parentSpanRef, scope := item.Parent(stateMetadata)
		if _, err := tracing.CreateMetadataSpan(
			ctx,
			tracerProvider,
			parentSpanRef,
			req.Location,
			w.PkgName,
			stateMetadata,
			item.Metadata,
			scope,
			addTenantIDs,
		); err != nil {
			return err
		}
	}

	// Persist the cumulative metadata size delta back to the state store.
	// Only persist when we successfully loaded from state; reconstructed
	// metadata has no backing store to update.
	if delta := stateMetadata.Metrics.SwapMetadataSizeDelta(); loadedFromState && delta > 0 {
		if err := statev2.TryIncrementMetadataSize(ctx, w.State, stateMetadata.ID, delta); err != nil {
			logger.StdlibLogger(ctx).Error("failed to persist metadata size delta",
				"error", err,
				"run_id", stateMetadata.ID.RunID.String(),
				"delta", delta,
			)
		}
	}

	return nil
}

func (w Writer) loadMetadata(ctx context.Context, id statev2.ID) (*statev2.Metadata, bool, error) {
	var missingErr error
	if w.State != nil {
		md, err := w.State.LoadMetadata(ctx, id)
		if err == nil {
			return &md, true, nil
		}
		if !errors.Is(err, statev2.ErrRunNotFound) && !errors.Is(err, statev2.ErrMetadataNotFound) {
			return nil, false, fmt.Errorf("%w: %w", ErrLoadMetadata, err)
		}
		missingErr = err
	}

	if w.MissingStateLoader != nil {
		loaded, err := w.MissingStateLoader(ctx, id)
		if err != nil {
			return nil, false, err
		}
		if loaded != nil {
			return loaded, false, nil
		}
	}

	if !w.AllowMissingState {
		return nil, false, statev2.ErrMetadataNotFound
	}

	if missingErr != nil {
		logger.StdlibLogger(ctx).Warn("failed to load run metadata for size limit check, falling back to request-local limit",
			"error", missingErr,
			"run_id", id.RunID.String(),
		)
	}
	return &statev2.Metadata{ID: id}, false, nil
}
