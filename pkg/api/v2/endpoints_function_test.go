package apiv2

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type skipScheduler struct {
	err error
}

func (s *skipScheduler) Schedule(ctx context.Context, req execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	return nil, nil, s.err
}

type noopEventPublisher struct{}

func (noopEventPublisher) Publish(ctx context.Context, evt event.TrackedEvent) error {
	return nil
}

func TestService_InvokeFunctionSkipped(t *testing.T) {
	tests := []struct {
		name        string
		scheduleErr error
		message     string
	}{
		{
			name:        "includes skip reason",
			scheduleErr: executor.SkippedError{Reason: enums.SkipReasonAccountExecutionCapHit},
			message:     "Function invocation was skipped: AccountExecutionCapHit.",
		},
		{
			name:        "falls back to generic message",
			scheduleErr: executor.ErrFunctionSkipped,
			message:     "Function invocation was skipped.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functions := &mockFunctionProvider{}
			functions.On("GetFunctionByApp", mock.Anything, "my-app", "hello-world").Return(inngest.DeployedFunction{
				ID:            uuid.New(),
				Slug:          "hello-world",
				AccountID:     uuid.New(),
				EnvironmentID: uuid.New(),
				AppID:         uuid.New(),
			}, nil).Once()
			t.Cleanup(func() {
				functions.AssertExpectations(t)
			})

			service := NewService(ServiceOptions{
				Functions:      functions,
				Executor:       &skipScheduler{err: tt.scheduleErr},
				EventPublisher: noopEventPublisher{},
			})

			data, err := structpb.NewStruct(map[string]any{"message": "hello"})
			require.NoError(t, err)

			resp, err := service.InvokeFunction(context.Background(), &apiv2.InvokeFunctionRequest{
				AppId:      "my-app",
				FunctionId: "hello-world",
				Data:       data,
			})

			require.Nil(t, resp)
			require.ErrorContains(t, err, tt.message)
			require.ErrorContains(t, err, apiv2base.ErrorFunctionSkipped)

			var statusErr interface{ HTTPStatus() int }
			require.ErrorAs(t, err, &statusErr)
			require.Equal(t, http.StatusUnprocessableEntity, statusErr.HTTPStatus())
		})
	}
}
