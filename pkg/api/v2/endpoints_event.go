package apiv2

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SendEventBodyLimitMiddleware limits event bodies before grpc-gateway decodes them.
func SendEventBodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		limited := apiv2base.MaxRequestBodyBytesMiddleware(maxBytes)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if after, ok := strings.CutPrefix(path, "/api/v2"); ok {
				path = after
			} else if after, ok := strings.CutPrefix(path, "/v2"); ok {
				path = after
			}
			if r.Method == http.MethodPost && path == "/events" {
				limited.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) SendEvent(ctx context.Context, req *apiv2.SendEventRequest) (*apiv2.SendEventResponse, error) {
	receivedAt := time.Now()
	if req == nil || req.Name == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Event name is required")
	}
	if len(req.Name) > consts.MaxEventNameLength {
		return nil, s.base.NewError(
			http.StatusBadRequest,
			apiv2base.ErrorInvalidFieldFormat,
			fmt.Sprintf("Event name cannot exceed %d characters", consts.MaxEventNameLength),
		)
	}
	if strings.HasPrefix(strings.ToLower(req.Name), event.InternalNamePrefix) {
		return nil, s.base.NewError(
			http.StatusBadRequest,
			apiv2base.ErrorInvalidFieldFormat,
			fmt.Sprintf("Event name is reserved for internal use: %s", req.Name),
		)
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_SendEvent_FullMethodName); result.Limited {
		return nil, s.base.NewError(
			http.StatusTooManyRequests,
			apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no event was sent.",
		)
	}

	if s.eventPublisher == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Send event is not yet implemented")
	}

	publishContext, err := s.eventContext(ctx)
	if err != nil {
		return nil, err
	}

	internalID := ulid.MustNew(ulid.Timestamp(receivedAt), rand.Reader)
	evt := event.Event{
		ID:        req.GetId(),
		Name:      req.Name,
		Data:      map[string]any{},
		Timestamp: receivedAt.UnixMilli(),
		User:      map[string]any{},
	}
	if req.Data != nil {
		evt.Data = req.Data.AsMap()
	}
	if req.User != nil {
		evt.User = req.User.AsMap()
	}
	if req.GetTs() != 0 {
		evt.Timestamp = req.GetTs()
	}
	if evt.ID == "" {
		evt.ID = internalID.String()
	}
	delete(evt.Data, consts.InngestEventDataPrefix)

	if err := evt.Validate(ctx); err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorValidationError, err.Error())
	}

	maxSize := publishContext.MaxSizeBytes
	if maxSize <= 0 {
		maxSize = consts.AbsoluteMaxEventSize
	}
	if evt.Size() > maxSize {
		return nil, s.base.NewError(
			http.StatusBadRequest,
			apiv2base.ErrorValidationError,
			fmt.Sprintf("Event payload cannot exceed %d bytes", maxSize),
		)
	}

	tracked := event.BaseTrackedEvent{
		ID:          internalID,
		AccountID:   publishContext.AccountID,
		WorkspaceID: publishContext.WorkspaceID,
		Event:       evt,
		ReceivedAt:  &receivedAt,
	}
	if err := s.eventPublisher.Publish(context.WithoutCancel(ctx), tracked); err != nil {
		logger.From(ctx).Error("unable to publish event via API", "error", err, "event_name", evt.Name)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to send event")
	}

	return &apiv2.SendEventResponse{
		Data: &apiv2.SendEventData{EventId: internalID.String()},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt: timestamppb.New(receivedAt),
		},
	}, nil
}
