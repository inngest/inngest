package apiv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateCancellation(ctx context.Context, req *apiv2.CreateCancellationRequest) (*apiv2.CreateCancellationResponse, error) {
	if strings.TrimSpace(req.GetAppId()) == "" || strings.TrimSpace(req.GetFunctionId()) == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID and function ID are required")
	}
	for _, field := range []struct {
		name  string
		value *timestamppb.Timestamp
	}{{"from", req.From}, {"until", req.Until}} {
		if field.value == nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, field.name+" is required")
		}
		if err := field.value.CheckValid(); err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, field.name+" must be a valid timestamp")
		}
	}
	opts := CreateCancellationOpts{
		From:  req.From.AsTime(),
		Until: req.Until.AsTime(),
		If:    req.If,
		Name:  req.Name,
	}
	if opts.From.After(opts.Until) {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "from must not be after until")
	}
	if opts.Until.After(time.Now().Add(5 * time.Second)) {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "until cannot be in the future")
	}
	if opts.Name != nil {
		name := strings.TrimSpace(*opts.Name)
		if name == "" || utf8.RuneCountInString(name) > 255 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "name must be between 1 and 255 characters")
		}
		opts.Name = &name
	}

	if opts.If != nil {
		if utf8.RuneCountInString(*opts.If) > 1024 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "if cannot exceed 1024 characters")
		}
		if err := expressions.Validate(ctx, expressions.DefaultRestrictiveValidationPolicy(), *opts.If); err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, fmt.Sprintf("Invalid expression: %s", err))
		}
	}
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_CreateCancellation_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited, "API rate limit exceeded. The request was rejected and no cancellation was created.")
	}
	if s.functions == nil || s.cancellations == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Cancellation creation is not yet implemented")
	}
	fn, err := s.functions.GetFunctionByApp(ctx, req.AppId, req.FunctionId)
	if err != nil {
		return nil, s.getFunctionError(err)
	}
	if !fn.ArchivedAt.IsZero() && fn.ArchivedAt.Before(time.Now()) {
		return nil, s.getFunctionError(ErrFunctionNotFound)
	}
	c, err := s.cancellations.CreateCancellation(ctx, fn, opts)
	if errors.Is(err, ErrFunctionNotFound) {
		return nil, s.getFunctionError(err)
	}
	if err != nil {
		var compileError *expressions.CompileError
		if errors.As(err, &compileError) {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Invalid expression: "+compileError.Message())
		}
		logger.From(ctx).Error("unable to create cancellation", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to create cancellation")
	}
	_ = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "201"))
	return &apiv2.CreateCancellationResponse{
		Data: &apiv2.Cancellation{
			Id:         c.ID.String(),
			AppId:      req.AppId,
			FunctionId: publicFunctionID(req.AppId, req.FunctionId, fn),
			Name:       c.Name,
			From:       timestamppb.New(opts.From),
			Until:      timestamppb.New(opts.Until),
			If:         c.If,
			CreatedAt:  timestamppb.New(c.CreatedAt),
		},
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}
