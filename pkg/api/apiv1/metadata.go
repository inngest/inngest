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
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

func (a router) addRunMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
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

	switch {
	case data.Target.StepID == nil:
		switch {
		case data.Target.StepIndex != nil:
			err = errors.New("target.step_id must be defined if target.step_index is defined")
		case data.Target.StepAttempt != nil:
			err = errors.New("target.step_id must be defined if target.step_attempt is defined")
		case data.Target.SpanID != nil:
			err = errors.New("target.step_id must be defined if target.span_id is defined")
		}
	case data.Target.StepAttempt == nil && data.Target.SpanID != nil:
		err = errors.New("target.step_attempt must be defined if target.span_id is defined")
	}

	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid metadata target"))
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
	Target   RunMetadataTarget        `json:"target"`
	Metadata []meta.RawMetadataUpdate `json:"metadata"`
}

func (a router) AddRunMetadata(ctx context.Context, auth apiv1auth.V1Auth, runID ulid.ULID, req *AddRunMetadataRequest) error {
	rootSpan, err := a.opts.TraceReader.GetSpansByRunID(ctx, runID)
	if err != nil {
		return err // TODO: better error
	}

	parentSpan, err := a.getParentSpan(ctx, auth, runID, &req.Target)
	if err != nil {
		return err // TODO: better error
	}

	parentSpanRef := &meta.SpanReference{
		TraceParent: fmt.Sprintf("00-%s-%s-00", parentSpan.TraceID, parentSpan.SpanID),
	}

	commonAttrs := meta.NewAttrSet(
		meta.Attr(meta.Attrs.RunID, &runID),
		meta.Attr(meta.Attrs.FunctionID, &rootSpan.FunctionID),
		meta.Attr(meta.Attrs.AppID, &rootSpan.AppID),
	)

	for _, md := range req.Metadata {
		attrs, err := tracing.MetadataAttrs(md)
		if err != nil {
			return err
		}

		_, err = a.opts.TracerProvider.CreateSpan(
			ctx,
			meta.SpanNameMetadata,
			&tracing.CreateSpanOptions{
				Attributes: attrs.Merge(commonAttrs),
				Parent:     parentSpanRef,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a router) getParentSpan(ctx context.Context, auth apiv1auth.V1Auth, runID ulid.ULID, target *RunMetadataTarget) (*cqrs.OtelSpan, error) {
	var span *cqrs.OtelSpan
	var err error

	switch {
	case target.StepID == nil:
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
			span, err = a.opts.TraceReader.GetStepSpanByStepID(ctx, runID, stepID, auth.AccountID(), auth.WorkspaceID())
		} else {
			span, err = a.opts.TraceReader.GetExecutionSpanByStepIDAndAttempt(ctx, runID, stepID, *target.StepAttempt, auth.AccountID(), auth.WorkspaceID())
		}
	default:
		span, err = a.opts.TraceReader.GetSpanBySpanID(ctx, runID, *target.StepID, auth.AccountID(), auth.WorkspaceID())
	}

	switch {
	// TODO: specific err cases
	case err != nil:
		return nil, publicerr.Wrap(err, 404, "Unable to find metadata target")
	}

	return span, nil
}
