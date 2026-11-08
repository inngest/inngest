package apiv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	minReplayNameLength = 3
	maxReplayNameLength = 64
	maxReplayFunctionID = 256
	maxReplayStatuses   = 5
	maxReplayClockSkew  = 5 * time.Second
)

func (s *Service) CreateReplay(ctx context.Context, req *apiv2.CreateReplayRequest) (*apiv2.CreateReplayResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_CreateReplay_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no replay was created.")
	}

	if s.replays == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Replay creation is not yet implemented")
	}

	opts, err := createReplayOpts(req)
	if err != nil {
		var validationErr *replayValidationError
		if errors.As(err, &validationErr) {
			return nil, s.base.NewError(http.StatusBadRequest, validationErr.code, validationErr.Error())
		}
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, err.Error())
	}

	result, err := s.replays.CreateReplay(ctx, opts)
	if err != nil {
		switch {
		case errors.Is(err, ErrFunctionNotFound):
			return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, fmt.Sprintf("Function %s not found", opts.FunctionID))
		case errors.Is(err, ErrFunctionPaused):
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, "Function is paused; unpause it before replaying")
		case errors.Is(err, ErrReplayInvalid):
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, "Replay request is invalid")
		}
		logger.From(ctx).Error("unable to create replay",
			"error", err,
			"function_id", opts.FunctionID,
			"from", opts.From,
			"until", opts.Until,
		)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to create replay")
	}
	if result.ID == uuid.Nil || result.CreatedAt.IsZero() {
		logger.From(ctx).Error("replay provider returned an invalid result",
			"replay_id", result.ID,
			"created_at", result.CreatedAt,
		)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to create replay")
	}

	_ = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "201"))
	statuses := make([]string, len(opts.Statuses))
	for i, status := range opts.Statuses {
		switch status {
		case enums.ReplayRunStatusAll:
			statuses[i] = "ALL"
		case enums.ReplayRunStatusCompleted:
			statuses[i] = "COMPLETED"
		case enums.ReplayRunStatusFailed:
			statuses[i] = "FAILED"
		case enums.ReplayRunStatusCancelled:
			statuses[i] = "CANCELLED"
		case enums.ReplayRunStatusSkippedPaused:
			statuses[i] = "SKIPPED_PAUSED"
		}
	}

	return &apiv2.CreateReplayResponse{
		Data: &apiv2.Replay{
			Id:         result.ID.String(),
			CreatedAt:  timestamppb.New(result.CreatedAt),
			Name:       opts.Name,
			FunctionId: opts.FunctionID,
			From:       timestamppb.New(opts.From),
			Until:      timestamppb.New(opts.Until),
			Statuses:   statuses,
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt: timestamppb.Now(),
		},
	}, nil
}

func createReplayOpts(req *apiv2.CreateReplayRequest) (CreateReplayOpts, error) {
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return CreateReplayOpts{}, missingReplayField("name is required")
	}
	nameLength := utf8.RuneCountInString(name)
	if nameLength < minReplayNameLength || nameLength > maxReplayNameLength {
		return CreateReplayOpts{}, invalidReplayField("name must be between %d and %d characters", minReplayNameLength, maxReplayNameLength)
	}

	appID := strings.TrimSpace(decodePathParam(req.GetAppId()))
	if appID == "" {
		return CreateReplayOpts{}, missingReplayField("appId is required")
	}
	functionID := strings.TrimSpace(decodePathParam(req.GetFunctionId()))
	if functionID == "" {
		return CreateReplayOpts{}, missingReplayField("functionId is required")
	}
	if len(functionID) > maxReplayFunctionID {
		return CreateReplayOpts{}, invalidReplayField("functionId cannot exceed %d bytes", maxReplayFunctionID)
	}

	from, err := requiredTimestamp(req.GetFrom(), "from")
	if err != nil {
		return CreateReplayOpts{}, err
	}
	until, err := requiredTimestamp(req.GetUntil(), "until")
	if err != nil {
		return CreateReplayOpts{}, err
	}
	now := time.Now().UTC()
	if until.After(now.Add(maxReplayClockSkew)) {
		return CreateReplayOpts{}, invalidReplayField("until cannot be in the future")
	}
	if until.After(now) {
		until = now
	}
	if from.After(until) {
		return CreateReplayOpts{}, invalidReplayField("from must not be after until")
	}

	statuses, err := replayStatuses(req.GetStatuses())
	if err != nil {
		return CreateReplayOpts{}, err
	}

	return CreateReplayOpts{
		Name:       name,
		AppID:      appID,
		FunctionID: functionID,
		From:       from,
		Until:      until,
		Statuses:   statuses,
	}, nil
}

func requiredTimestamp(ts *timestamppb.Timestamp, field string) (time.Time, error) {
	if ts == nil {
		return time.Time{}, missingReplayField("%s is required", field)
	}
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, invalidReplayField("%s must be a valid timestamp", field)
	}
	return ts.AsTime().UTC(), nil
}

func replayStatuses(values []string) ([]enums.ReplayRunStatus, error) {
	if len(values) == 0 {
		return nil, missingReplayField("at least one status is required")
	}
	if len(values) > maxReplayStatuses {
		return nil, invalidReplayField("at most %d statuses are allowed", maxReplayStatuses)
	}

	statuses := make([]enums.ReplayRunStatus, 0, len(values))
	seen := make(map[enums.ReplayRunStatus]struct{}, len(values))
	for _, value := range values {
		var status enums.ReplayRunStatus
		switch strings.ToUpper(strings.TrimSpace(value)) {
		case "ALL":
			status = enums.ReplayRunStatusAll
		case "COMPLETED":
			status = enums.ReplayRunStatusCompleted
		case "FAILED":
			status = enums.ReplayRunStatusFailed
		case "CANCELLED":
			status = enums.ReplayRunStatusCancelled
		case "SKIPPED_PAUSED":
			status = enums.ReplayRunStatusSkippedPaused
		default:
			return nil, invalidReplayField("status must be one of ALL, COMPLETED, FAILED, CANCELLED, or SKIPPED_PAUSED")
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		statuses = append(statuses, status)
	}
	if len(statuses) > 1 {
		if _, ok := seen[enums.ReplayRunStatusAll]; ok {
			return nil, invalidReplayField("ALL cannot be combined with another status")
		}
	}
	return statuses, nil
}

type replayValidationError struct {
	code    string
	message string
}

func (e *replayValidationError) Error() string {
	return e.message
}

func missingReplayField(format string, args ...any) error {
	return &replayValidationError{
		code:    apiv2base.ErrorMissingField,
		message: fmt.Sprintf(format, args...),
	}
}

func invalidReplayField(format string, args ...any) error {
	return &replayValidationError{
		code:    apiv2base.ErrorInvalidFieldFormat,
		message: fmt.Sprintf(format, args...),
	}
}
