package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type cancellationCreatorFunc func(context.Context, inngest.DeployedFunction, CreateCancellationOpts) (*cqrs.Cancellation, error)

func (f cancellationCreatorFunc) CreateCancellation(ctx context.Context, fn inngest.DeployedFunction, opts CreateCancellationOpts) (*cqrs.Cancellation, error) {
	return f(ctx, fn, opts)
}

func TestHTTPGateway_CreateCancellation(t *testing.T) {
	for _, tc := range []struct {
		name                                      string
		bodyIDs                                   bool
		lookupErr, createErr                      error
		archived, noCreator, noFunctions, limited bool
		status                                    int
		code                                      string
	}{
		{
			name:   "created",
			status: 201,
		},
		{
			name:    "path IDs override body IDs",
			bodyIDs: true,
			status:  201,
		},
		{
			name:      "unknown function",
			lookupErr: ErrFunctionNotFound,
			status:    404,
			code:      "not_found",
		},
		{
			name:      "lookup failed",
			lookupErr: errors.New("database unavailable"),
			status:    500,
			code:      "internal_error",
		},
		{
			name:     "archived",
			archived: true,
			status:   404,
			code:     "not_found",
		},
		{
			name:      "provider scope miss",
			createErr: ErrFunctionNotFound,
			status:    404,
			code:      "not_found",
		},
		{
			name:      "provider failure",
			createErr: errors.New("writer unavailable"),
			status:    500,
			code:      "internal_error",
		},
		{
			name:      "provider expression error",
			createErr: expressions.NewCompileError(errors.New("invalid expression")),
			status:    400,
			code:      "invalid_field_format",
		},
		{
			name:      "no creator",
			noCreator: true,
			status:    501,
			code:      "not_implemented",
		},
		{
			name:        "no functions",
			noFunctions: true,
			status:      501,
			code:        "not_implemented",
		},
		{
			name:    "limited",
			limited: true,
			status:  429,
			code:    "rate_limited",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			until := from.Add(time.Hour)
			createdAt := until.Add(time.Hour)
			id := ulid.Make()
			fn := inngest.DeployedFunction{
				ID:            uuid.New(),
				AccountID:     uuid.New(),
				EnvironmentID: uuid.New(),
				AppID:         uuid.New(),
				Slug:          "app-fn",
				Function:      inngest.Function{Slug: "fn"},
			}
			if tc.archived {
				fn.ArchivedAt = from
			}
			functions := &mockFunctionProvider{}
			if !tc.noCreator && !tc.noFunctions && !tc.limited {
				functions.On("GetFunctionByApp", mock.Anything, "app", "fn").Return(fn, tc.lookupErr).Once()
			}
			t.Cleanup(func() { functions.AssertExpectations(t) })
			limiter := &testEventRateLimiter{limited: tc.limited}
			opts := ServiceOptions{RateLimitProvider: limiter}
			if !tc.noFunctions {
				opts.Functions = functions
			}
			calls := 0
			if !tc.noCreator {
				opts.Cancellations = cancellationCreatorFunc(func(_ context.Context, got inngest.DeployedFunction, input CreateCancellationOpts) (*cqrs.Cancellation, error) {
					calls++
					require.Equal(t, fn, got)
					require.Equal(t, CreateCancellationOpts{
						From:  from,
						Until: until,
						If:    new(`event.data.keep == false`),
						Name:  new("cleanup"),
					}, input)
					return &cqrs.Cancellation{
						ID:        id,
						CreatedAt: createdAt,
						Name:      input.Name,
						If:        input.If,
					}, tc.createErr
				})
			}
			handler, err := newTestHTTPHandler(t.Context(), opts, HTTPHandlerOptions{})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			requestBody := map[string]any{
				"from":  "2026-01-01T00:00:00Z",
				"until": "2026-01-01T01:00:00Z",
				"if":    "event.data.keep == false",
				"name":  "cleanup",
			}
			if tc.bodyIDs {
				requestBody["appId"] = "other-app"
				requestBody["functionId"] = "other-function"
			}
			data, err := json.Marshal(requestBody)
			require.NoError(t, err)
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v2/apps/app/functions/fn/cancellations", strings.NewReader(string(data))))
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Equal(t, apiv2.V2_CreateCancellation_FullMethodName, limiter.method)
			if tc.noCreator || tc.noFunctions || tc.limited || tc.lookupErr != nil || tc.archived {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			if tc.code != "" {
				var body errorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				require.Len(t, body.Errors, 1)
				require.Equal(t, tc.code, body.Errors[0].Code)
				return
			}
			var body apiv2.CreateCancellationResponse
			require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), &body))
			require.True(t, proto.Equal(&apiv2.Cancellation{
				Id:         id.String(),
				AppId:      "app",
				FunctionId: "fn",
				Name:       new("cleanup"),
				If:         new(`event.data.keep == false`),
				From:       timestamppb.New(from),
				Until:      timestamppb.New(until),
				CreatedAt:  timestamppb.New(createdAt),
			}, body.Data), rec.Body.String())
			require.WithinDuration(t, time.Now(), body.Metadata.FetchedAt.AsTime(), time.Second)
			require.Empty(t, rec.Header().Get("Grpc-Metadata-X-Http-Code"))
		})
	}
}

func TestHTTPGateway_CreateCancellationValidation(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		value       any
		remove      bool
		message     string
	}{
		{
			name:    "missing from",
			field:   "from",
			remove:  true,
			message: "from is required",
		},
		{
			name:    "missing until",
			field:   "until",
			remove:  true,
			message: "until is required",
		},
		{
			name:    "null from",
			field:   "from",
			value:   nil,
			message: "from is required",
		},
		{
			name:    "invalid timestamp",
			field:   "from",
			value:   "nope",
			message: "Timestamp",
		},
		{
			name:    "reversed range",
			field:   "from",
			value:   "2026-01-02T00:00:00Z",
			message: "from must not be after until",
		},
		{
			name:    "future",
			field:   "until",
			value:   time.Now().Add(time.Hour).Format(time.RFC3339),
			message: "future",
		},
		{
			name:    "bad expression",
			field:   "if",
			value:   "event.data. ==",
			message: "Invalid expression",
		},
		{
			name:    "restricted expression",
			field:   "if",
			value:   "[1, 2].exists(x, x == 1)",
			message: "Invalid expression",
		},
		{
			name:    "empty expression",
			field:   "if",
			value:   "",
			message: "Invalid expression",
		},
		{
			name:    "long expression",
			field:   "if",
			value:   strings.Repeat("x", 1025),
			message: "1024",
		},

		{
			name:    "empty name",
			field:   "name",
			value:   "",
			message: "name must be between 1 and 255 characters",
		},
		{
			name:    "blank name",
			field:   "name",
			value:   " \t\n",
			message: "name must be between 1 and 255 characters",
		},
		{
			name:    "long name",
			field:   "name",
			value:   strings.Repeat("x", 256),
			message: "name must be between 1 and 255 characters",
		},
		{
			name:    "long unicode name",
			field:   "name",
			value:   strings.Repeat("界", 256),
			message: "name must be between 1 and 255 characters",
		},
		{
			name:    "unknown field",
			field:   "extra",
			value:   true,
			message: "Unexpected request body field",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"from": "2026-01-01T00:00:00Z", "until": "2026-01-01T01:00:00Z"}
			if tc.remove {
				delete(body, tc.field)
			} else {
				body[tc.field] = tc.value
			}
			data, err := json.Marshal(body)
			require.NoError(t, err)
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v2/apps/app/functions/fn/cancellations", strings.NewReader(string(data))))
			require.Equal(t, 400, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tc.message)
		})
	}
}

func TestCreateCancellationEqualBoundsAndClockSkew(t *testing.T) {
	for _, offset := range []time.Duration{-time.Hour, 3 * time.Second} {
		at := time.Now().Add(offset)
		fn := inngest.DeployedFunction{
			Slug:     "my app-fn",
			Function: inngest.Function{Slug: "fn"},
		}
		functions := &mockFunctionProvider{}
		functions.On("GetFunctionByApp", mock.Anything, "my app", "fn").Return(fn, nil).Once()
		svc := NewService(ServiceOptions{
			Functions: functions,
			Cancellations: cancellationCreatorFunc(func(_ context.Context, _ inngest.DeployedFunction, opts CreateCancellationOpts) (*cqrs.Cancellation, error) {
				require.Equal(t, at.UTC(), opts.From)
				require.Equal(t, at.UTC(), opts.Until)
				require.Nil(t, opts.If)
				require.Nil(t, opts.Name)
				return &cqrs.Cancellation{
					ID:        ulid.Make(),
					CreatedAt: time.Now(),
				}, nil
			}),
		})
		response, err := svc.CreateCancellation(t.Context(), &apiv2.CreateCancellationRequest{
			AppId:      "%20my%20app%20",
			FunctionId: "%20fn%20",
			From:       timestamppb.New(at),
			Until:      timestamppb.New(at),
		})
		require.NoError(t, err)
		require.Equal(t, "my app", response.Data.AppId)
		require.Equal(t, "fn", response.Data.FunctionId)
		functions.AssertExpectations(t)
	}
}

func TestCreateCancellationInvalidProtoTimestamp(t *testing.T) {
	_, err := NewService(ServiceOptions{}).CreateCancellation(t.Context(), &apiv2.CreateCancellationRequest{
		AppId:      "app",
		FunctionId: "fn",
		From:       &timestamppb.Timestamp{Seconds: 253402300800},
		Until:      timestamppb.Now(),
	})
	require.ErrorContains(t, err, "from must be a valid timestamp")
}

func TestHTTPGateway_CreateCancellationName(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{"trimmed", " \t cleanup \n", "cleanup"},
		{"one character", "x", "x"},
		{"255 characters", strings.Repeat("x", 255), strings.Repeat("x", 255)},
		{"255 unicode characters", strings.Repeat("界", 255), strings.Repeat("界", 255)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inngest.DeployedFunction{
				Slug:     "app-fn",
				Function: inngest.Function{Slug: "fn"},
			}
			functions := &mockFunctionProvider{}
			functions.On("GetFunctionByApp", mock.Anything, "app", "fn").Return(fn, nil).Once()
			creator := cancellationCreatorFunc(func(_ context.Context, _ inngest.DeployedFunction, opts CreateCancellationOpts) (*cqrs.Cancellation, error) {
				require.Equal(t, new(tc.want), opts.Name)
				return &cqrs.Cancellation{
					ID:        ulid.Make(),
					CreatedAt: time.Now(),
					Name:      opts.Name,
				}, nil
			})
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{
				Functions:     functions,
				Cancellations: creator,
			}, HTTPHandlerOptions{})
			require.NoError(t, err)
			data, err := json.Marshal(map[string]any{"from": "2026-01-01T00:00:00Z", "until": "2026-01-01T01:00:00Z", "name": tc.input})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v2/apps/app/functions/fn/cancellations", strings.NewReader(string(data))))
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			var response apiv2.CreateCancellationResponse
			require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), &response))
			require.Equal(t, new(tc.want), response.Data.Name)
			functions.AssertExpectations(t)
		})
	}
}

func TestCreateCancellationPathValidation(t *testing.T) {
	for _, tc := range []struct {
		name, appID, functionID, message string
	}{
		{"missing app", "", "fn", "appId is required"},
		{"blank app", "%20", "fn", "appId is required"},
		{"missing function", "app", "", "functionId is required"},
		{"blank function", "app", "%20", "functionId is required"},
		{"long function", "app", strings.Repeat("x", 257), "functionId cannot exceed 256 bytes"},
		{"long unicode function", "app", strings.Repeat("界", 86), "functionId cannot exceed 256 bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewService(ServiceOptions{}).CreateCancellation(t.Context(), &apiv2.CreateCancellationRequest{
				AppId:      tc.appID,
				FunctionId: tc.functionID,
			})
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestHTTPGateway_CreateCancellationRateLimitBeforeValidation(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`{"from":"2026-01-02T00:00:00Z","until":"2026-01-01T00:00:00Z"}`,
		`{"from":"2026-01-01T00:00:00Z","until":"2026-01-02T00:00:00Z","name":" "}`,
		`{"from":"2026-01-01T00:00:00Z","until":"2026-01-02T00:00:00Z","if":"event.data. =="}`,
	} {
		handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{
			RateLimitProvider: &testEventRateLimiter{limited: true},
		}, HTTPHandlerOptions{})
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v2/apps/app/functions/fn/cancellations", strings.NewReader(body)))
		require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"code":"rate_limited"`)
	}
}
