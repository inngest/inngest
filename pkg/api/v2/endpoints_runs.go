package apiv2

import (
	"context"
	"net/http"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

func (s *Service) GetFunctionRun(ctx context.Context, req *apiv2.GetFunctionRunRequest) (*apiv2.GetFunctionRunResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}

	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function run is not yet implemented")
}

func (s *Service) GetFunctionTrace(ctx context.Context, req *apiv2.GetFunctionTraceRequest) (*apiv2.GetFunctionTraceResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}

	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function trace is not yet implemented")
}
