package apiv1

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type MetadataOpts struct {
	Flag AllowMetadataFlag

	SpanExtractor metadata.SpanExtractor
}

type AllowMetadataFlag func(ctx context.Context, accountID uuid.UUID) bool

func (am AllowMetadataFlag) Enabled(ctx context.Context, accountID uuid.UUID) bool {
	if am == nil {
		return false
	}

	return am(ctx, accountID)
}

func (a router) addRunMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	if !a.opts.MetadataOpts.Flag.Enabled(ctx, auth.AccountID()) {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(403, "Metadata is not enabled for this account"))
		return
	}

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	data := AddRunMetadataRequest{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid metadata target"))
		return
	}

	if data.Target.StepID == nil {
		switch {
		case data.Target.StepIndex != nil:
			err = errors.New("target.step_id must be defined if target.step_index is defined")
		case data.Target.StepAttempt != nil:
			err = errors.New("target.step_id must be defined if target.step_attempt is defined")
		case data.Target.SpanID != nil:
			err = errors.New("target.step_id must be defined if target.span_id is defined")
		}
	}

	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid metadata target"))
		return
	}

	err = a.AddRunMetadata(ctx, auth, runID, &data)
	switch {
	case errors.Is(err, metadata.ErrMetadataSpanTooLarge):
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 413, "Metadata span exceeds maximum size of 64KB"))
		return
	case errors.Is(err, metadata.ErrRunMetadataSizeExceeded):
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 413, "Cumulative metadata size exceeds limit"))
		return
	case err != nil:
		_ = publicerr.WriteHTTP(w, err)
		return
	}

}

type RunMetadataTarget struct {
	StepID *string `json:"step_id"`
	// StepIndex == nil is equivalent to StepIndex == 0
	StepIndex *int `json:"step_index"`
	// When StepAttempt == -1 (legacy) or nil, select the last attempt
	StepAttempt *int    `json:"step_attempt"`
	SpanID      *string `json:"span_id"`
}

type AddRunMetadataRequest struct {
	Target   RunMetadataTarget `json:"target"`
	Metadata []metadata.Update `json:"metadata"`
}

func (a router) AddRunMetadata(ctx context.Context, auth apiv1auth.V1Auth, runID ulid.ULID, req *AddRunMetadataRequest) error {
	parentSpan, scope, err := a.getParentSpan(ctx, auth, runID, &req.Target)
	if err != nil {
		return err
	}

	// Load run metadata to enforce the per-run cumulative size limit against
	// metadata that already exists in the run, not just this request.
	stateID := statev2.ID{
		RunID:      parentSpan.RunID,
		FunctionID: parentSpan.FunctionID,
		Tenant: statev2.Tenant{
			AppID:     parentSpan.AppID,
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}

	for _, md := range req.Metadata {
		if err := md.ValidateAllowedForScope(scope); err != nil {
			return publicerr.Wrap(err, 400, "Invalid metadata")
		}
	}

	var stateMetadata *statev2.Metadata
	loadedFromState := false
	if a.opts.State != nil {
		md, err := a.opts.State.LoadMetadata(ctx, stateID)
		if errors.Is(err, statev2.ErrRunNotFound) || errors.Is(err, statev2.ErrMetadataNotFound) {
			logger.StdlibLogger(ctx).Warn("failed to load run metadata for size limit check, falling back to request-local limit",
				"error", err,
				"run_id", runID.String(),
			)
		} else if err != nil {
			return publicerr.Wrap(err, 500, "Unable to load run metadata")
		} else {
			stateMetadata = &md
			loadedFromState = true
		}
	}

	// When state metadata is unavailable (e.g. stale runs evicted from the
	// state store), use a zero-value Metadata as a request-local fallback so
	// that CreateMetadataSpan still enforces the per-run cumulative size
	// limit within this request. This prevents unbounded metadata writes to
	// old runs.
	if stateMetadata == nil {
		stateMetadata = &statev2.Metadata{ID: stateID}
	}
	statev2.InitConfig(&stateMetadata.Config)

	parentSpanRef := &meta.SpanReference{
		TraceParent:            fmt.Sprintf("00-%s-%s-00", parentSpan.TraceID, parentSpan.SpanID),
		DynamicSpanID:          parentSpan.SpanID,
		DynamicSpanTraceParent: fmt.Sprintf("00-%s-%s-00", parentSpan.TraceID, parentSpan.SpanID),
	}

	addTenantIDs := func(cfg *tracing.MetadataSpanConfig) {
		meta.AddAttr(cfg.Attrs, meta.Attrs.AccountID, util.ToPtr(auth.AccountID()))
		meta.AddAttr(cfg.Attrs, meta.Attrs.EnvID, util.ToPtr(auth.WorkspaceID()))
		meta.AddAttr(cfg.Attrs, meta.Attrs.FunctionID, &parentSpan.FunctionID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.RunID, &parentSpan.RunID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.AppID, &parentSpan.AppID)
	}

	for _, md := range req.Metadata {
		_, err = tracing.CreateMetadataSpan(
			ctx,
			a.opts.TracerProvider,
			parentSpanRef,
			"router.AddRunMetadata",
			pkgName,
			stateMetadata,
			md,
			scope,
			addTenantIDs,
		)
		if err != nil {
			return err
		}
	}

	// Persist the cumulative metadata size delta back to the state store.
	// Only persist when we successfully loaded from state; the fallback
	// Metadata is request-local and has no backing store to update.
	if loadedFromState {
		if delta := stateMetadata.Metrics.SwapMetadataSizeDelta(); delta > 0 {
			if err := statev2.TryIncrementMetadataSize(ctx, a.opts.State, stateID, delta); err != nil {
				logger.StdlibLogger(ctx).Error("failed to persist metadata size delta",
					"error", err,
					"run_id", runID.String(),
					"delta", delta,
				)
			}
		}
	}

	return nil
}

func (a router) getParentSpan(ctx context.Context, auth apiv1auth.V1Auth, runID ulid.ULID, target *RunMetadataTarget) (*cqrs.OtelSpan, metadata.Scope, error) {
	var scope metadata.Scope
	var span *cqrs.OtelSpan
	var err error

	switch {
	case target.StepID == nil:
		scope = enums.MetadataScopeRun
		span, err = a.opts.TraceReader.GetRunSpanByRunID(ctx, runID, auth.AccountID(), auth.WorkspaceID())
	case target.StepAttempt == nil || target.SpanID == nil:
		var stepID string
		if target.StepIndex == nil || *target.StepIndex == 0 {
			sum := sha1.Sum([]byte(*target.StepID))
			stepID = hex.EncodeToString(sum[:])
		} else {

			sum := sha1.Sum(fmt.Appendf(nil, "%s:%d", *target.StepID, *target.StepIndex))
			stepID = hex.EncodeToString(sum[:])
		}

		if target.StepAttempt == nil || *target.StepAttempt < 0 {
			scope = enums.MetadataScopeStep
			span, err = a.opts.TraceReader.GetLatestExecutionSpanByStepID(ctx, runID, stepID, auth.AccountID(), auth.WorkspaceID())
		} else {
			scope = enums.MetadataScopeStep
			span, err = a.opts.TraceReader.GetExecutionSpanByStepIDAndAttempt(ctx, runID, stepID, *target.StepAttempt, auth.AccountID(), auth.WorkspaceID())
		}
	default:
		scope = enums.MetadataScopeExtendedTrace
		// TODO: require that this is a extended trace span
		span, err = a.opts.TraceReader.GetSpanBySpanID(ctx, runID, *target.SpanID, auth.AccountID(), auth.WorkspaceID())
	}

	switch {
	// TODO: specific err cases
	case err != nil:
		return nil, 0, publicerr.Wrap(err, 404, "Unable to find metadata target")
	}

	return span, scope, nil
}
