package apiv2

import (
	"context"
	"net/http"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

func ValidateInvokeRequest(ctx context.Context, req *apiv2.InvokeFunctionRequest) error {
	if req.FunctionId == "" {
		return apiv2base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Function ID is required")
	}

	if req.Data == nil {
		return apiv2base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Input data is required")
	}

	// Validate mode parameter if provided
	if req.Mode != nil {
		if *req.Mode != "sync" && *req.Mode != "async" {
			return apiv2base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Mode must be either 'sync' or 'async'")
		}
	}
	return nil
}
