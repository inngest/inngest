package apiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func replayRequest() *apiv2.CreateReplayRequest {
	return &apiv2.CreateReplayRequest{Name: "Replay orders", AppId: "my-app", FunctionId: "orders-created", From: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)), Until: timestamppb.New(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)), Statuses: []string{"FAILED"}}
}

func TestHTTPGatewayCreateReplay(t *testing.T) {
	provider := &mockReplayProvider{}
	req := replayRequest()
	created := ReplayResult{ID: uuid.New(), CreatedAt: time.Now().UTC()}
	provider.On("CreateReplay", mock.Anything, CreateReplayOpts{Name: req.Name, AppID: req.AppId, FunctionID: req.FunctionId, From: req.From.AsTime(), Until: req.Until.AsTime(), Statuses: []enums.ReplayRunStatus{enums.ReplayRunStatusFailed}}).Return(created, nil).Once()
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Replays: provider}, HTTPHandlerOptions{})
	require.NoError(t, err)
	body, err := protojson.Marshal(req)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v2/apps/my-app/functions/orders-created/replays", strings.NewReader(string(body)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var result apiv2.CreateReplayResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &result))
	require.Equal(t, created.ID.String(), result.Data.Id)
	require.Equal(t, "orders-created", result.Data.FunctionId)
	require.Equal(t, []string{"FAILED"}, result.Data.Statuses)
	provider.AssertExpectations(t)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v2/replays", strings.NewReader(string(body))))
	require.Equal(t, http.StatusNotFound, response.Code)
}

func TestCreateReplayValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*apiv2.CreateReplayRequest)
	}{
		{"missing name", func(r *apiv2.CreateReplayRequest) { r.Name = " " }},
		{"short name", func(r *apiv2.CreateReplayRequest) { r.Name = "ab" }},
		{"long name", func(r *apiv2.CreateReplayRequest) { r.Name = strings.Repeat("a", 65) }},
		{"missing app", func(r *apiv2.CreateReplayRequest) { r.AppId = " " }},
		{"missing function", func(r *apiv2.CreateReplayRequest) { r.FunctionId = " " }},
		{"long function", func(r *apiv2.CreateReplayRequest) { r.FunctionId = strings.Repeat("a", 257) }},
		{"missing from", func(r *apiv2.CreateReplayRequest) { r.From = nil }},
		{"missing until", func(r *apiv2.CreateReplayRequest) { r.Until = nil }},
		{"invalid timestamp", func(r *apiv2.CreateReplayRequest) { r.From = &timestamppb.Timestamp{Nanos: -1} }},
		{"reversed bounds", func(r *apiv2.CreateReplayRequest) { r.From, r.Until = r.Until, r.From }},
		{"future bound", func(r *apiv2.CreateReplayRequest) { r.Until = timestamppb.New(time.Now().Add(time.Hour)) }},
		{"missing statuses", func(r *apiv2.CreateReplayRequest) { r.Statuses = nil }},
		{"unknown status", func(r *apiv2.CreateReplayRequest) { r.Statuses = []string{"RUNNING"} }},
		{"all plus specific", func(r *apiv2.CreateReplayRequest) { r.Statuses = []string{"ALL", "FAILED"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := replayRequest()
			tc.change(req)
			provider := &mockReplayProvider{}
			_, err := NewService(ServiceOptions{Replays: provider}).CreateReplay(t.Context(), req)
			require.Error(t, err)
			provider.AssertNotCalled(t, "CreateReplay", mock.Anything, mock.Anything)
		})
	}
	// Equal bounds are useful when replaying a single timestamp. Normalize only
	// presentation differences; the selection itself must remain explicit.
	req := replayRequest()
	req.Name = "  Replay orders  "
	req.FunctionId = " orders "
	req.Until = req.From
	req.Statuses = []string{" failed ", "FAILED"}
	opts, err := createReplayOpts(req)
	require.NoError(t, err)
	require.Equal(t, "Replay orders", opts.Name)
	require.Equal(t, "orders", opts.FunctionID)
	require.Equal(t, opts.From, opts.Until)
	require.Equal(t, []enums.ReplayRunStatus{enums.ReplayRunStatusFailed}, opts.Statuses)
}

func TestHTTPGatewayReplayProviderErrors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{ErrFunctionNotFound, http.StatusNotFound}, {ErrFunctionPaused, http.StatusBadRequest}, {ErrReplayInvalid, http.StatusBadRequest}, {errors.New("private database address"), http.StatusInternalServerError},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			provider := &mockReplayProvider{}
			provider.On("CreateReplay", mock.Anything, mock.Anything).Return(ReplayResult{}, tc.err).Once()
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Replays: provider}, HTTPHandlerOptions{})
			require.NoError(t, err)
			body, err := protojson.Marshal(replayRequest())
			require.NoError(t, err)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v2/apps/my-app/functions/orders-created/replays", strings.NewReader(string(body))))
			require.Equal(t, tc.status, response.Code, response.Body.String())
			require.NotContains(t, response.Body.String(), "private database address")
		})
	}
}

func TestHTTPGatewayReplayAuthorization(t *testing.T) {
	for _, path := range []string{"/apps/my-app/functions/orders-created/replays", "/v2/apps/my-app/functions/orders-created/replays", "/api/v2/apps/my-app/functions/orders-created/replays", "/api/v2/apps/my-app/functions/orders-created/replays/"} {
		t.Run(path, func(t *testing.T) {
			handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{}, HTTPHandlerOptions{AuthzMiddleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
			}})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
			if strings.HasSuffix(path, "/") {
				require.Equal(t, http.StatusNotFound, response.Code)
			} else {
				require.Equal(t, http.StatusForbidden, response.Code)
			}
		})
	}
}

func TestHTTPGatewayReplayRejectsAmbiguousInput(t *testing.T) {
	original, err := protojson.Marshal(replayRequest())
	require.NoError(t, err)
	for _, tc := range []struct{ name, path, body string }{
		{"query selector", "/v2/apps/my-app/functions/orders-created/replays?functionId=other", string(original)},
		{"array selector", "/v2/apps/my-app/functions/orders-created/replays", `{"name":"Replay orders","functionIds":["orders","other"],"from":"2026-01-01T00:00:00Z","until":"2026-01-02T00:00:00Z","statuses":["FAILED"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &mockReplayProvider{}
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Replays: provider}, HTTPHandlerOptions{})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			provider.AssertNotCalled(t, "CreateReplay", mock.Anything, mock.Anything)
		})
	}
	// Exercise the handler's limiter before admission; provider errors must not be
	// mistaken for a successful replay.
	limiter := &mockRateLimitProvider{}
	limiter.On("CheckRateLimit", mock.Anything, apiv2.V2_CreateReplay_FullMethodName).Return(RateLimitResult{Limited: true})
	provider := &mockReplayProvider{}
	_, err = NewService(ServiceOptions{Replays: provider, RateLimitProvider: limiter}).CreateReplay(t.Context(), proto.Clone(replayRequest()).(*apiv2.CreateReplayRequest))
	require.ErrorContains(t, err, "rate limit")
	provider.AssertNotCalled(t, "CreateReplay", mock.Anything, mock.Anything)
}
