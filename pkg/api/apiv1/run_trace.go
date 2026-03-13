package apiv1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

const (
	maxSpanTreeDepth  = 50
	maxSpanChildCount = 1000
)

// TraceSpanResponse is the REST API representation of a trace span.
// It mirrors the GQL RunTraceSpan model but uses snake_case JSON keys
// and exposes only the fields relevant to REST consumers.
type TraceSpanResponse struct {
	SpanID       string               `json:"span_id"`
	TraceID      string               `json:"trace_id,omitempty"`
	Name         string               `json:"name"`
	Status       string               `json:"status"`
	StepOp       string               `json:"step_op,omitempty"`
	StepID       string               `json:"step_id,omitempty"`
	Attempts     *int                 `json:"attempts,omitempty"`
	QueuedAt     time.Time            `json:"queued_at"`
	StartedAt    *time.Time           `json:"started_at,omitempty"`
	EndedAt      *time.Time           `json:"ended_at,omitempty"`
	Duration     *int                 `json:"duration_ms,omitempty"`
	IsRoot       bool                 `json:"is_root,omitempty"`
	OutputID     *string              `json:"output_id,omitempty"`
	ParentSpanID *string              `json:"parent_span_id,omitempty"`
	StepInfo     map[string]any       `json:"step_info,omitempty"`
	Children     []*TraceSpanResponse `json:"children,omitempty"`
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
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(501, "No trace reader specified"))
		return
	}

	// Verify the run belongs to this workspace before loading the full span
	// tree. GetSpansByRunID does not scope by account/workspace at the SQL
	// layer, so we load the run and check its workspace explicitly.
	run, err := a.opts.TraceReader.GetRun(ctx, runID, auth.AccountID(), auth.WorkspaceID())
	if err != nil || run == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "Run not found: %s", runID))
		return
	}
	if run.WorkspaceID != auth.WorkspaceID() {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "Run not found: %s", runID))
		return
	}

	rootSpan, err := a.opts.TraceReader.GetSpansByRunID(ctx, runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load trace for run: %s", runID))
		return
	}
	if rootSpan == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "No trace found for run: %s", runID))
		return
	}

	// Use the same converter that GQL uses — single source of truth for
	// span tree presentation (discovery collapsing, attempt renaming,
	// status propagation, metadata, userland spans, etc.)
	gqlSpan, err := loader.ConvertOtelSpanToModel(ctx, rootSpan)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to convert trace for run: %s", runID))
		return
	}

	// Convert the GQL model to our REST response type (snake_case, curated fields)
	// and enforce tree size limits in a single pass.
	resp := convertSpanToResponse(gqlSpan, 0)

	_ = WriteResponse(w, resp)
}

// convertSpanToResponse maps a GQL RunTraceSpan to the REST TraceSpanResponse,
// enforcing depth and width limits during traversal.
func convertSpanToResponse(span *models.RunTraceSpan, depth int) *TraceSpanResponse {
	if span == nil {
		return nil
	}

	resp := &TraceSpanResponse{
		SpanID:       span.SpanID,
		TraceID:      span.TraceID,
		Name:         span.Name,
		Status:       string(span.Status),
		Attempts:     span.Attempts,
		QueuedAt:     span.QueuedAt,
		StartedAt:    span.StartedAt,
		EndedAt:      span.EndedAt,
		Duration:     span.Duration,
		IsRoot:       span.IsRoot,
		OutputID:     span.OutputID,
		ParentSpanID: span.ParentSpanID,
		StepInfo:     convertStepInfo(span.StepInfo),
	}

	if span.StepOp != nil {
		resp.StepOp = span.StepOp.String()
	}
	if span.StepID != nil {
		resp.StepID = *span.StepID
	}

	// Enforce depth and width limits to prevent unbounded response sizes.
	if depth >= maxSpanTreeDepth || len(span.ChildrenSpans) == 0 {
		return resp
	}

	children := span.ChildrenSpans
	if len(children) > maxSpanChildCount {
		children = children[:maxSpanChildCount]
	}

	resp.Children = make([]*TraceSpanResponse, 0, len(children))
	for _, child := range children {
		if converted := convertSpanToResponse(child, depth+1); converted != nil {
			resp.Children = append(resp.Children, converted)
		}
	}

	return resp
}

// convertStepInfo maps GQL StepInfo interface types to a snake_case map for REST serialization.
func convertStepInfo(info models.StepInfo) map[string]any {
	if info == nil {
		return nil
	}

	switch v := info.(type) {
	case models.InvokeStepInfo:
		m := map[string]any{
			"function_id":       v.FunctionID,
			"triggering_event_id": v.TriggeringEventID.String(),
			"timeout":           v.Timeout.Format(time.RFC3339),
		}
		if v.ReturnEventID != nil {
			m["return_event_id"] = v.ReturnEventID.String()
		}
		if v.RunID != nil {
			m["run_id"] = v.RunID.String()
		}
		if v.TimedOut != nil {
			m["timed_out"] = *v.TimedOut
		}
		return m
	case models.SleepStepInfo:
		return map[string]any{
			"sleep_until": v.SleepUntil.Format(time.RFC3339),
		}
	case models.WaitForEventStepInfo:
		m := map[string]any{
			"event_name": v.EventName,
			"timeout":    v.Timeout.Format(time.RFC3339),
		}
		if v.Expression != nil {
			m["expression"] = *v.Expression
		}
		if v.FoundEventID != nil {
			m["found_event_id"] = v.FoundEventID.String()
		}
		if v.TimedOut != nil {
			m["timed_out"] = *v.TimedOut
		}
		return m
	case models.WaitForSignalStepInfo:
		m := map[string]any{
			"signal":  v.Signal,
			"timeout": v.Timeout.Format(time.RFC3339),
		}
		if v.TimedOut != nil {
			m["timed_out"] = *v.TimedOut
		}
		return m
	case models.RunStepInfo:
		if v.Type != nil {
			return map[string]any{"type": *v.Type}
		}
		return nil
	default:
		return nil
	}
}
