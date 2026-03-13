package apiv1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

const (
	maxSpanTreeDepth  = 50
	maxSpanChildCount = 1000
)

// TraceSpanResponse is the API representation of a trace span.
type TraceSpanResponse struct {
	SpanID    string              `json:"span_id"`
	Name      string              `json:"name"`
	Status    string              `json:"status"`
	StepOp    string              `json:"step_op,omitempty"`
	Attempts  int                 `json:"attempts,omitempty"`
	QueuedAt  *time.Time          `json:"queued_at,omitempty"`
	StartedAt *time.Time          `json:"started_at,omitempty"`
	EndedAt   *time.Time          `json:"ended_at,omitempty"`
	Duration  *int64              `json:"duration,omitempty"`
	IsRoot    bool                `json:"is_root,omitempty"`
	OutputID  *string             `json:"output_id,omitempty"`
	StepInfo  map[string]any      `json:"step_info,omitempty"`
	Children  []TraceSpanResponse `json:"children,omitempty"`
	Truncated bool                `json:"truncated,omitempty"`
}

func (a router) getRunTrace(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/runs/{runID}/trace") {
		return
	}

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

	if a.opts.TraceReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No trace reader specified"))
		return
	}

	rootSpan, err := a.opts.TraceReader.GetRunSpanByRunID(ctx, runID, auth.AccountID(), auth.WorkspaceID())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load trace for run: %s", runID))
		return
	}
	if rootSpan == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "No trace found for run: %s", runID))
		return
	}

	resp := convertOtelSpan(rootSpan, 0)
	_ = WriteResponse(w, resp)
}

func convertOtelSpan(span *cqrs.OtelSpan, depth int) TraceSpanResponse {
	resp := TraceSpanResponse{
		SpanID:   span.SpanID,
		Name:     span.GetStepName(),
		Status:   span.Status.String(),
		Attempts: span.GetAttempts(),
		IsRoot:   span.GetIsRoot(),
		OutputID: span.GetOutputID(),
	}

	// Step operation
	if span.Attributes != nil && span.Attributes.StepOp != nil {
		resp.StepOp = span.Attributes.StepOp.String()
	}

	// Timestamps
	queuedAt := span.GetQueuedAtTime()
	resp.QueuedAt = &queuedAt

	if started := span.GetStartedAtTime(); started != nil {
		resp.StartedAt = started
	}
	if ended := span.GetEndedAtTime(); ended != nil {
		resp.EndedAt = ended
	}

	// Duration in milliseconds
	if resp.StartedAt != nil {
		end := time.Now()
		if resp.EndedAt != nil {
			end = *resp.EndedAt
		}
		ms := end.Sub(*resp.StartedAt).Milliseconds()
		resp.Duration = &ms
	}

	// Step info
	resp.StepInfo = buildStepInfo(span)

	// Children — enforce depth and width limits to prevent unbounded recursion
	if depth >= maxSpanTreeDepth {
		resp.Truncated = true
		return resp
	}

	childCount := 0
	for _, child := range span.Children {
		if child == nil || child.MarkedAsDropped {
			continue
		}
		if childCount >= maxSpanChildCount {
			resp.Truncated = true
			break
		}
		resp.Children = append(resp.Children, convertOtelSpan(child, depth+1))
		childCount++
	}

	return resp
}

func buildStepInfo(span *cqrs.OtelSpan) map[string]any {
	if span.Attributes == nil || span.Attributes.StepOp == nil {
		return nil
	}

	info := map[string]any{}
	attrs := span.Attributes

	switch *attrs.StepOp {
	case enums.OpcodeInvokeFunction:
		if attrs.StepInvokeFunctionID != nil {
			info["functionID"] = *attrs.StepInvokeFunctionID
		}
		if attrs.StepWaitExpiry != nil {
			info["timeout"] = attrs.StepWaitExpiry.Format(time.RFC3339)
		}
		if attrs.StepWaitExpired != nil {
			info["timedOut"] = *attrs.StepWaitExpired
		}
		if attrs.StepInvokeTriggerEventID != nil {
			info["triggeringEventID"] = attrs.StepInvokeTriggerEventID.String()
		}
	case enums.OpcodeSleep:
		if attrs.StepWaitExpiry != nil {
			info["sleepUntil"] = attrs.StepWaitExpiry.Format(time.RFC3339)
		}
	case enums.OpcodeWaitForEvent:
		if attrs.StepWaitForEventName != nil {
			info["eventName"] = *attrs.StepWaitForEventName
		}
		if attrs.StepWaitForEventIf != nil {
			info["expression"] = *attrs.StepWaitForEventIf
		}
		if attrs.StepWaitExpiry != nil {
			info["timeout"] = attrs.StepWaitExpiry.Format(time.RFC3339)
		}
		if attrs.StepWaitExpired != nil {
			info["timedOut"] = *attrs.StepWaitExpired
		}
	}

	if len(info) == 0 {
		return nil
	}
	return info
}
