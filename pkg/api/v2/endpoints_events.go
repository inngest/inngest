package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultEventsLimit    = 40
	maxEventsLimit        = cqrs.MaxEvents - 1
	maxEventsPayloadLimit = 20
)

func (s *Service) ListEvents(ctx context.Context, req *apiv2.ListEventsRequest) (*apiv2.ListEventsResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListEvents_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited, "API rate limit exceeded. No events were fetched.")
	}
	if s.events == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "List events is not yet implemented")
	}

	includePayload := false
	for _, value := range req.GetInclude() {
		include, err := ParseListEventsInclude(value)
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
		}
		includePayload = include == ListEventsIncludePayload
	}
	maxLimit := maxEventsLimit
	if includePayload {
		maxLimit = maxEventsPayloadLimit
	}
	limit := int(req.GetLimit())
	if limit == 0 {
		limit = min(defaultEventsLimit, maxLimit)
	}
	if limit < 1 || limit > maxLimit {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, fmt.Sprintf("Limit must be between 1 and %d", maxLimit))
	}
	if len(req.GetName()) > 20 {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "At most 20 event names are allowed")
	}
	for _, name := range req.GetName() {
		if len(name) == 0 || len(name) > 256 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Event names must be between 1 and 256 bytes")
		}
		if strings.ContainsRune(name, '\\') || strings.ContainsFunc(name, unicode.IsControl) {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Event names must not contain backslashes or control characters")
		}
	}
	opts := &cqrs.WorkspaceEventsOpts{
		Limit:                 limit + 1,
		Names:                 req.GetName(),
		IncludeInternalEvents: false,
		OmitEventData:         !includePayload,
	}
	if req.GetCursor() != "" {
		cursor, err := ulid.ParseStrict(req.GetCursor())
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Cursor must be a valid event cursor")
		}
		opts.Cursor = &cursor
	}
	from, err := optionalTimestamp(req.GetFrom(), "from")
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	if from != nil && from.IsZero() {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "from must not be the zero time")
	}
	until, err := optionalTimestamp(req.GetUntil(), "until")
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	if until != nil && until.IsZero() {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "until must not be the zero time")
	}
	if from != nil {
		opts.Oldest = *from
	}
	if until != nil {
		opts.Newest = *until
		if from == nil {
			opts.Oldest = until.Add(-time.Hour)
		}
	}
	if from != nil && until != nil && from.After(*until) {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "from must not be after until")
	}
	if err := opts.Validate(); err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	events, err := s.events.GetEvents(ctx, opts)
	if err != nil {
		logger.From(ctx).Error("unable to fetch events", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch events")
	}
	page := &apiv2.Page{Limit: int32(limit), HasMore: len(events) > limit}
	if page.HasMore {
		events = events[:limit]
		page.Cursor = new(events[len(events)-1].ID.String())
	}
	data := make([]*apiv2.Event, 0, len(events))
	for _, evt := range events {
		item, err := toAPIEvent(evt, includePayload)
		if err != nil {
			return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to build event response")
		}
		data = append(data, item)
	}
	return &apiv2.ListEventsResponse{
		Data:     data,
		Metadata: runsResponseMetadata(&opts.Oldest, &opts.Newest),
		Page:     page,
	}, nil
}

func (s *Service) GetEvent(ctx context.Context, req *apiv2.GetEventRequest) (*apiv2.GetEventResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetEvent_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited, "API rate limit exceeded. No event was fetched.")
	}
	if s.events == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get event is not yet implemented")
	}
	id, err := ulid.ParseStrict(req.GetEventId())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Event ID must be a valid ULID")
	}
	includePayload := false
	for _, value := range req.GetInclude() {
		include, err := ParseGetEventInclude(value)
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
		}
		includePayload = include == GetEventIncludePayload
	}
	evt, err := s.events.GetEvent(ctx, id)
	if errors.Is(err, ErrEventNotFound) {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Event not found")
	}
	if err != nil {
		logger.From(ctx).Error("unable to fetch event", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch event")
	}
	if strings.HasPrefix(evt.EventName, "inngest/") {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Event not found")
	}
	data, err := toAPIEvent(evt, includePayload)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to build event response")
	}
	return &apiv2.GetEventResponse{Data: data, Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()}}, nil
}

func toAPIEvent(evt *cqrs.Event, includePayload bool) (*apiv2.Event, error) {
	data := &apiv2.Event{
		Id:             evt.ID.String(),
		Name:           evt.EventName,
		ReceivedAt:     timestamppb.New(evt.ReceivedAt),
		IdempotencyKey: evt.EventID,
	}
	if evt.EventTS != 0 {
		data.OccurredAt = timestamppb.New(time.UnixMilli(evt.EventTS))
	}
	if evt.EventVersion != "" {
		data.Version = &evt.EventVersion
	}
	if evt.Source != "" || evt.SourceID != nil {
		data.Source = &apiv2.EventSource{}
		if evt.Source != "" {
			data.Source.Name = &evt.Source
		}
		if evt.SourceID != nil {
			data.Source.Id = new(evt.SourceID.String())
		}
	}
	if includePayload {
		payload, err := json.Marshal(evt.Event())
		if err != nil {
			return nil, err
		}
		data.Payload = &structpb.Struct{}
		if err := protojson.Unmarshal(payload, data.Payload); err != nil {
			return nil, err
		}
	}
	return data, nil
}
