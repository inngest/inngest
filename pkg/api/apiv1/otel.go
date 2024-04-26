package apiv1

import (
	"fmt"
	"io"
	"net/http"

	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/publicerr"
	ptrace "go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	pbContentType   = "application/x-protobuf"
	jsonContentType = "application/json"
)

func (a router) OTLPTrace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Err:     err,
			Message: err.Error(),
		})
	}

	var encoder ptrace.Unmarshaler

	cnt := r.Header.Get("Content-Type")
	switch cnt {
	case pbContentType:
		encoder = &ptrace.ProtoUnmarshaler{}
	case jsonContentType:
		encoder = &ptrace.JSONUnmarshaler{}
	default:
		log.From(ctx).Error().Str("content-type", cnt).Msg("unknown content type for traces")
		err = fmt.Errorf("unable to handle unknown content type for traces: %s", cnt)
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Err:     err,
			Message: err.Error(),
		})
		return
	}

	traces, err := encoder.UnmarshalTraces(body)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Err:     err,
			Message: err.Error(),
		})
		return
	}
	log.From(ctx).Trace().Int("len", traces.SpanCount()).Msg("recording otel trace spans")

	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rs := traces.ResourceSpans().At(i)
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			for k := 0; k < ss.Spans().Len(); k++ {
				// span := ss.Spans().At(k)
				// TODO: construct the data to be inserted into the DB
				// fmt.Printf("Span: %#v\n", span)
			}
		}
	}
}
