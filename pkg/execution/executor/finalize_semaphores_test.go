package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

// startedRunService is a run state store with one run.  it counts metadata
// loads and returns the configured error.
type startedRunService struct {
	sv2.RunService
	md      sv2.Metadata
	loadErr error
	loads   int
}

func (s *startedRunService) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	s.loads++
	return s.md, s.loadErr
}

// finalize releases the manual-release semaphores of a run that has
// StartedAt in run metadata at the time of the finalize.  the copy from its
// caller does not decide.  a run that is gone has nothing to release,
// because the finalize that deleted it released.
func TestSemaphoresToRelease(t *testing.T) {
	manual := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDFn(uuid.New()),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseManual,
	}
	auto := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDApp(uuid.New()),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseAuto,
	}
	started := time.Now()

	tests := []struct {
		name       string
		semaphores []constraintapi.Semaphore
		callerAt   time.Time
		storedAt   time.Time
		loadErr    error
		want       []constraintapi.Semaphore
		wantLoads  int
	}{
		{
			name:       "run started after the caller loaded metadata",
			semaphores: []constraintapi.Semaphore{auto, manual},
			storedAt:   started,
			want:       []constraintapi.Semaphore{manual},
			wantLoads:  1,
		},
		{
			name:       "run was cancelled before it started",
			semaphores: []constraintapi.Semaphore{manual},
			wantLoads:  1,
		},
		{
			name:       "run was finalized by a different process",
			semaphores: []constraintapi.Semaphore{manual},
			callerAt:   started,
			loadErr:    state.ErrRunNotFound,
			wantLoads:  1,
		},
		{
			name:       "run has no function concurrency",
			semaphores: []constraintapi.Semaphore{auto},
			storedAt:   started,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := sv2.ID{
				RunID:      ulid.Make(),
				FunctionID: uuid.New(),
			}
			runState := &startedRunService{
				md: sv2.Metadata{
					ID: id,
					Config: sv2.Config{
						Semaphores: tc.semaphores,
						StartedAt:  tc.storedAt,
					},
				},
				loadErr: tc.loadErr,
			}
			e := &executor{
				smv2: runState,
			}

			got := e.semaphoresToRelease(context.Background(), sv2.Metadata{
				ID: id,
				Config: sv2.Config{
					Semaphores: tc.semaphores,
					StartedAt:  tc.callerAt,
				},
			})
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantLoads, runState.loads)
		})
	}
}

// only a start item that carries a manual-release semaphore and has a
// capacity lease took a function concurrency slot.  a retry of a started
// run carries no manual semaphore, and takes none.
func TestTakesFnConcurrencySlot(t *testing.T) {
	manual := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDFn(uuid.New()),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseManual,
	}
	auto := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDApp(uuid.New()),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseAuto,
	}
	lease := &queue.CapacityLease{
		LeaseID: ulid.Make(),
	}

	tests := []struct {
		name       string
		kind       string
		semaphores []constraintapi.Semaphore
		lease      *queue.CapacityLease
		want       bool
	}{
		{
			name:       "start item with a manual semaphore",
			kind:       queue.KindStart,
			semaphores: []constraintapi.Semaphore{auto, manual},
			lease:      lease,
			want:       true,
		},
		{
			name:       "retry of a started run",
			kind:       queue.KindStart,
			semaphores: []constraintapi.Semaphore{auto},
			lease:      lease,
		},
		{
			name:       "not a start item",
			kind:       queue.KindEdge,
			semaphores: []constraintapi.Semaphore{manual},
			lease:      lease,
		},
		{
			name:       "no capacity lease",
			kind:       queue.KindStart,
			semaphores: []constraintapi.Semaphore{manual},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := takesFnConcurrencySlot(queue.Item{
				Kind:       tc.kind,
				Semaphores: tc.semaphores,
				RunInfo: &queue.RunInfo{
					CapacityLease: tc.lease,
				},
			})
			require.Equal(t, tc.want, got)
		})
	}
}
