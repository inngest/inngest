package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRunServiceForReset struct {
	sv2.RunService
	mock.Mock
}

func (m *mockRunServiceForReset) UpdateMetadata(ctx context.Context, id sv2.ID, config sv2.MutableConfig) error {
	args := m.Called(ctx, id, config)
	return args.Error(0)
}

func TestMaybeResetForceStepPlan(t *testing.T) {
	tests := []struct {
		name           string
		requestVersion int
		forceStepPlan  bool
		updateErr      error
		expectCall     bool
		expectErr      bool
	}{
		{
			name:           "v1_noop",
			requestVersion: 1,
			forceStepPlan:  true,
			expectCall:     false,
		},
		{
			name:           "v2_already_false",
			requestVersion: 2,
			forceStepPlan:  false,
			expectCall:     false,
		},
		{
			name:           "v2_resets",
			requestVersion: 2,
			forceStepPlan:  true,
			expectCall:     true,
		},
		{
			name:           "v2_propagates_error",
			requestVersion: 2,
			forceStepPlan:  true,
			updateErr:      errors.New("redis unavailable"),
			expectCall:     true,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := new(mockRunServiceForReset)
			cfg := sv2.InitConfig(&sv2.Config{
				RequestVersion: tt.requestVersion,
				ForceStepPlan:  tt.forceStepPlan,
				StartedAt:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			})
			md := sv2.Metadata{Config: *cfg}

			if tt.expectCall {
				ms.On("UpdateMetadata", mock.Anything, md.ID, mock.MatchedBy(func(cfg sv2.MutableConfig) bool {
					return !cfg.ForceStepPlan &&
						cfg.RequestVersion == tt.requestVersion &&
						cfg.StartedAt.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
				})).Return(tt.updateErr)
			}

			e := &executor{smv2: ms}
			err := e.maybeResetForceStepPlan(context.Background(), &md)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.updateErr.Error())
			} else {
				require.NoError(t, err)
			}

			if tt.expectCall {
				ms.AssertExpectations(t)
			} else {
				ms.AssertNotCalled(t, "UpdateMetadata", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}
