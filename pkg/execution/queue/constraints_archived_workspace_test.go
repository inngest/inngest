package queue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/stretchr/testify/require"
)

func TestBypassArchivedWorkspaceAppSemaphore(t *testing.T) {
	accountID := uuid.New()
	workspaceID := uuid.New()
	appSemaphore := constraintapi.Semaphore{ID: "app:" + uuid.NewString()}
	functionSemaphore := constraintapi.Semaphore{ID: "fn:" + uuid.NewString()}

	tests := []struct {
		name          string
		semaphores    []constraintapi.Semaphore
		callbackValue bool
		want          bool
		wantCalls     int
	}{
		{
			name:          "app semaphore and callback enabled",
			semaphores:    []constraintapi.Semaphore{appSemaphore},
			callbackValue: true,
			want:          true,
			wantCalls:     1,
		},
		{
			name:       "app semaphore and callback disabled",
			semaphores: []constraintapi.Semaphore{appSemaphore},
			wantCalls:  1,
		},
		{
			name:       "non-app semaphore does not check workspace",
			semaphores: []constraintapi.Semaphore{functionSemaphore},
		},
		{
			name: "no semaphores does not check workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			processor := &queueProcessor{QueueOptions: NewQueueOptions(
				WithBypassArchivedWorkspaceAppSemaphore(func(_ context.Context, gotAccountID, gotWorkspaceID uuid.UUID) bool {
					calls++
					require.Equal(t, accountID, gotAccountID)
					require.Equal(t, workspaceID, gotWorkspaceID)
					return tt.callbackValue
				}),
			)}

			got := processor.bypassArchivedWorkspaceAppSemaphore(
				context.Background(),
				accountID,
				workspaceID,
				tt.semaphores,
			)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestArchivedWorkspaceAppSemaphoreBypassPreservesOtherSemaphores(t *testing.T) {
	appSemaphore := constraintapi.Semaphore{ID: "app:" + uuid.NewString()}
	functionSemaphore := constraintapi.Semaphore{ID: "fn:" + uuid.NewString()}

	require.True(t, hasAppSemaphoreConfig([]constraintapi.Semaphore{appSemaphore, functionSemaphore}))
	require.True(t, hasNonAppSemaphore([]constraintapi.Semaphore{appSemaphore, functionSemaphore}))
	require.False(t, hasNonAppSemaphore([]constraintapi.Semaphore{appSemaphore}))
}
