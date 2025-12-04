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
	}

	for _, md := range data.Metadata {
		if err := md.Validate(); err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid metadata"))
			return
		}
	}

	err = a.AddRunMetadata(ctx, auth, runID, &data)
	switch {
	// TODO: better cases for specific errors
	case err != nil:
		_ = publicerr.WriteHTTP(w, err)
		return
	}

}

type RunMetadataTarget struct {
	StepID *string `json:"step_id"`
	// StepIndex == nil is equivalent to StepIndex == 0
	StepIndex *int `json:"step_index"`
	// When StepAttempt == -1, select the last attempt
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
		if err := md.Validate(); err != nil {
			return publicerr.Wrap(err, 400, "Invalid metadata")
		}

		// TODO: validate that specific kinds are allowed to be set by the user and check account-level metadata
		// limits.
		_, err := tracing.CreateMetadataSpan(
			ctx,
			a.opts.TracerProvider,
			parentSpanRef,
			"router.AddRunMetadata",
			pkgName,
			nil,
			md,
			scope,
			addTenantIDs,
		)
		if err != nil {
			return err
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

		if target.StepAttempt == nil {
			scope = enums.MetadataScopeStep
			span, err = a.opts.TraceReader.GetStepSpanByStepID(ctx, runID, stepID, auth.AccountID(), auth.WorkspaceID())
		} else if *target.StepAttempt < 0 {
			scope = enums.MetadataScopeStepAttempt
			span, err = a.opts.TraceReader.GetLatestExecutionSpanByStepID(ctx, runID, stepID, auth.AccountID(), auth.WorkspaceID())
		} else {
			scope = enums.MetadataScopeStepAttempt
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
