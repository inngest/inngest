package apiv2

import (
	"context"
	"net/http"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

func validateInvokeRequest(ctx context.Context, req *apiv2.InvokeFunctionRequest) error {
	if req.FunctionId == "" {
		return apiv2base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Function ID is required")
	}

	if req.Data == nil {
		return apiv2base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Input data is required")
	}

	return nil
}
