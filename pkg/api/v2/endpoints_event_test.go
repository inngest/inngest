package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type testEventPublisher struct {
	event event.TrackedEvent
	err   error
}

func (p *testEventPublisher) Publish(_ context.Context, evt event.TrackedEvent) error {
	p.event = evt
	return p.err
}

type testEventRateLimiter struct {
	limited bool
	method  string
}

func (l *testEventRateLimiter) CheckRateLimit(_ context.Context, method string) RateLimitResult {
	l.method = method
	return RateLimitResult{Limited: l.limited}
}

func TestService_SendEvent(t *testing.T) {
	accountID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	workspaceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	timestamp := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC).UnixMilli()
	data, err := structpb.NewStruct(map[string]any{
		"message":  "hello",
		"_inngest": map[string]any{"invoke": true},
	})
	require.NoError(t, err)
	user, err := structpb.NewStruct(map[string]any{"id": "user-1"})
	require.NoError(t, err)
	publisher := &testEventPublisher{}
	limiter := &testEventRateLimiter{}
	service := NewService(ServiceOptions{
		EventPublisher: publisher,
		EventContext: func(context.Context) (EventPublishContext, error) {
			return EventPublishContext{
				AccountID:    accountID,
				WorkspaceID:  workspaceID,
				MaxSizeBytes: 1024,
			}, nil
		},
		RateLimitProvider: limiter,
	})

	resp, err := service.SendEvent(context.Background(), &apiv2.SendEventRequest{
		Name: "app/user.created",
		Data: data,
		User: user,
		Id:   stringPointer("event-idempotency-key"),
		Ts:   &timestamp,
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Metadata.GetFetchedAt())
	require.Equal(t, apiv2.V2_SendEvent_FullMethodName, limiter.method)
	require.NotNil(t, publisher.event)
	require.Equal(t, resp.Data.EventId, publisher.event.GetInternalID().String())
	require.Equal(t, accountID, publisher.event.GetAccountID())
	require.Equal(t, workspaceID, publisher.event.GetWorkspaceID())
	require.Equal(t, "event-idempotency-key", publisher.event.GetEvent().ID)
	require.Equal(t, "app/user.created", publisher.event.GetEvent().Name)
	require.Equal(t, timestamp, publisher.event.GetEvent().Timestamp)
	require.Equal(t, map[string]any{"message": "hello"}, publisher.event.GetEvent().Data)
	require.Equal(t, map[string]any{"id": "user-1"}, publisher.event.GetEvent().User)
	require.WithinDuration(t, time.Now(), publisher.event.GetReceivedAt(), time.Second)
	require.NotEqual(t, time.UnixMilli(timestamp), publisher.event.GetReceivedAt())
}

func TestService_SendEventValidation(t *testing.T) {
	publisher := &testEventPublisher{}
	service := NewService(ServiceOptions{EventPublisher: publisher})

	tests := []struct {
		name    string
		request *apiv2.SendEventRequest
		error   string
	}{
		{
			name:    "missing event name",
			request: &apiv2.SendEventRequest{},
			error:   "Event name is required",
		},
		{
			name:    "reserved event name",
			request: &apiv2.SendEventRequest{Name: "Inngest/function.finished"},
			error:   "reserved for internal use",
		},
		{
			name:    "event name too long",
			request: &apiv2.SendEventRequest{Name: strings.Repeat("a", 513)},
			error:   "cannot exceed 512 characters",
		},
		{
			name: "timestamp too old",
			request: &apiv2.SendEventRequest{
				Name: "test/event",
				Ts:   int64Pointer(time.Date(1979, 12, 31, 23, 59, 59, 0, time.UTC).UnixMilli()),
			},
			error: "timestamp is before Jan 1, 1980",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.SendEvent(context.Background(), tt.request)

			require.Nil(t, resp)
			require.ErrorContains(t, err, tt.error)
		})
	}
	require.Nil(t, publisher.event)
}

func TestService_SendEventAcceptsV1MaximumEventNameLength(t *testing.T) {
	publisher := &testEventPublisher{}
	service := NewService(ServiceOptions{EventPublisher: publisher})

	resp, err := service.SendEvent(context.Background(), &apiv2.SendEventRequest{
		Name: strings.Repeat("a", consts.MaxEventNameLength),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, publisher.event)
}

func TestService_SendEventRejectsRateLimitAndOversizedPayload(t *testing.T) {
	t.Run("rate limited", func(t *testing.T) {
		publisher := &testEventPublisher{}
		service := NewService(ServiceOptions{
			EventPublisher:    publisher,
			RateLimitProvider: &testEventRateLimiter{limited: true},
		})

		resp, err := service.SendEvent(context.Background(), &apiv2.SendEventRequest{Name: "test/event"})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "API rate limit exceeded")
		require.Nil(t, publisher.event)
	})

	t.Run("payload too large", func(t *testing.T) {
		publisher := &testEventPublisher{}
		data, err := structpb.NewStruct(map[string]any{"value": strings.Repeat("x", 100)})
		require.NoError(t, err)
		service := NewService(ServiceOptions{
			EventPublisher: publisher,
			EventContext: func(context.Context) (EventPublishContext, error) {
				return EventPublishContext{MaxSizeBytes: 50}, nil
			},
		})

		resp, err := service.SendEvent(context.Background(), &apiv2.SendEventRequest{
			Name: "test/event",
			Data: data,
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Event payload cannot exceed 50 bytes")
		require.Nil(t, publisher.event)
	})
}

func TestService_SendEventPublishFailure(t *testing.T) {
	service := NewService(ServiceOptions{
		EventPublisher: &testEventPublisher{err: errors.New("publish failed")},
	})

	resp, err := service.SendEvent(context.Background(), &apiv2.SendEventRequest{Name: "test/event"})

	require.Nil(t, resp)
	require.ErrorContains(t, err, "Unable to send event")
}

func TestHTTPGateway_SendEvent(t *testing.T) {
	publisher := &testEventPublisher{}
	handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{EventPublisher: publisher}, HTTPHandlerOptions{})
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/events",
		strings.NewReader(`{"name":"test/event","data":{"message":"hello"},"id":"custom-id","ts":1786032000000}`),
	)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	var body struct {
		Data struct {
			EventID string `json:"eventId"`
		} `json:"data"`
		Metadata struct {
			FetchedAt time.Time `json:"fetchedAt"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, publisher.event.GetInternalID().String(), body.Data.EventID)
	require.False(t, body.Metadata.FetchedAt.IsZero())
	require.Equal(t, "custom-id", publisher.event.GetEvent().ID)
	require.Equal(t, map[string]any{"message": "hello"}, publisher.event.GetEvent().Data)
}

func TestHTTPGateway_SendEventRejectsOversizedRequestBody(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		contentLength int64
	}{
		{
			name:          "declared content length",
			body:          `{}`,
			contentLength: consts.AbsoluteMaxEventSize + 1,
		},
		{
			name:          "streamed body",
			body:          `{"name":"test/event","data":{"value":"` + strings.Repeat("x", consts.AbsoluteMaxEventSize) + `"}}`,
			contentLength: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{}, HTTPHandlerOptions{})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v2/events", strings.NewReader(tt.body))
			req.ContentLength = tt.contentLength
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
			require.JSONEq(t, fmt.Sprintf(
				`{"errors":[{"code":"invalid_request","message":"Request body cannot exceed %d bytes"}]}`,
				consts.AbsoluteMaxEventSize,
			), recorder.Body.String())
		})
	}
}

func TestHTTPGateway_SendEventBodyLimitDoesNotApplyToOtherRoutes(t *testing.T) {
	handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{}, HTTPHandlerOptions{})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/health", strings.NewReader(`{}`))
	req.ContentLength = consts.AbsoluteMaxEventSize + 1
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.NotEqual(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

func stringPointer(value string) *string {
	return &value
}

func int64Pointer(value int64) *int64 {
	return &value
}
