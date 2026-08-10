package apiv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var sendEventHTTPMethod, sendEventHTTPPath = sendEventRoute()

func sendEventRoute() (string, string) {
	for _, endpoint := range apiv2endpoint.Discover() {
		if endpoint.MethodName == "SendEvent" {
			return endpoint.HTTPMethod, endpoint.Path
		}
	}
	panic("SendEvent API route not found")
}

// SendEventBodyLimitMiddleware rejects large event bodies before they are read.
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
			if r.Method == sendEventHTTPMethod && path == sendEventHTTPPath {
				limited.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) SendEvent(ctx context.Context, req *apiv2.SendEventRequest) (*apiv2.SendEventResponse, error) {
	receivedAt := time.Now()
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_SendEvent_FullMethodName); result.Limited {
		return nil, s.base.NewError(
			http.StatusTooManyRequests,
			apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no event was sent.",
		)
	}
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
	if s.eventSender == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Send event is not yet implemented")
	}

	evt := event.Event{
		ID:        req.GetId(),
		Name:      req.Name,
		Data:      map[string]any{},
		Timestamp: req.GetTs(),
	}
	if req.Data != nil {
		evt.Data = req.Data.AsMap()
	}
	if req.User != nil {
		evt.User = req.User.AsMap()
	}
	if _, err := event.NormalizeInbound(ctx, &evt, receivedAt); err != nil {
		code := apiv2base.ErrorValidationError
		if errors.Is(err, event.ErrInternalEventName) {
			code = apiv2base.ErrorInvalidFieldFormat
		}
		return nil, s.base.NewError(http.StatusBadRequest, code, err.Error())
	}

	// Enforce this server's maximum event size. Cloud also applies the account's plan limit.
	if evt.Size() > s.maxEventSize {
		return nil, s.base.NewError(
			http.StatusBadRequest,
			apiv2base.ErrorValidationError,
			fmt.Sprintf("Event payload cannot exceed %d bytes", s.maxEventSize),
		)
	}

	eventID, err := s.eventSender(ctx, &evt)
	if err != nil {
		var httpErr interface{ HTTPStatus() int }
		if errors.As(err, &httpErr) {
			return nil, err
		}
		logger.From(ctx).Error("unable to publish event via API", "error", err, "event_name", evt.Name)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to send event")
	}

	return &apiv2.SendEventResponse{
		Data: &apiv2.SendEventData{EventId: eventID},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt: timestamppb.New(receivedAt),
		},
	}, nil
}
