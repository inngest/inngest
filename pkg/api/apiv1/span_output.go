package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
)

// SpanOutputResponse is the API representation of a span's output.
type SpanOutputResponse struct {
	Input json.RawMessage    `json:"input,omitempty"`
	Data  json.RawMessage    `json:"data,omitempty"`
	Error *StepErrorResponse `json:"error,omitempty"`
}

// StepErrorResponse is the API representation of a step error.
type StepErrorResponse struct {
	Message string `json:"message"`
	Name    string `json:"name,omitempty"`
	Stack   string `json:"stack,omitempty"`
}

func (a router) getSpanOutput(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/traces/span-output/{outputID}") {
		return
	}

	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	if a.opts.TraceReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No trace reader specified"))
		return
	}

	outputID := chi.URLParam(r, "outputID")

	id := &cqrs.SpanIdentifier{}
	if err := id.Decode(outputID); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid output ID"))
		return
	}

	if id.WorkspaceID != auth.WorkspaceID() {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "Span output not found"))
		return
	}

	// Use preview or legacy path based on the identifier
	var spanData *cqrs.SpanOutput
	if id.Preview == nil || !*id.Preview {
		spanData, err = a.opts.TraceReader.LegacyGetSpanOutput(ctx, *id)
	} else {
		spanData, err = a.opts.TraceReader.GetSpanOutput(ctx, *id)
	}
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load span output"))
		return
	}
	if spanData == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "Span output not found"))
		return
	}

	resp := SpanOutputResponse{}

	if spanData.IsError {
		var stepErr StepErrorResponse
		if err := json.Unmarshal(spanData.Data, &stepErr); err != nil {
			// If unmarshal fails, put raw data as the stack trace
			stepErr.Stack = string(spanData.Data)
		}
		if stepErr.Message == "" {
			stepErr.Stack = string(spanData.Data)
		}
		resp.Error = &stepErr
	} else {
		resp.Data = spanData.Data
	}

	if len(spanData.Input) > 0 {
		resp.Input = spanData.Input
	}

	_ = WriteResponse(w, resp)
}
