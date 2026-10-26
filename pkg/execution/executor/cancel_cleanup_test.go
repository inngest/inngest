package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type cleanupState struct {
	sv2.RunService
	md                                sv2.Metadata
	metadataErr, eventsErr, deleteErr error
	deleted                           bool
}

func (s *cleanupState) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	return s.md, s.metadataErr
}
func (s *cleanupState) LoadEvents(context.Context, sv2.ID) ([]json.RawMessage, error) {
	if s.eventsErr != nil {
		return nil, s.eventsErr
	}
	return []json.RawMessage{json.RawMessage(`{"name":"test"}`)}, s.eventsErr
}
func (s *cleanupState) LoadDefers(context.Context, sv2.ID) (map[string]sv2.Defer, error) {
	return nil, nil
}
func (s *cleanupState) Delete(context.Context, sv2.ID, ...sv2.DeleteOption) error {
	if s.deleteErr == nil {
		s.deleted = true
	}
	return s.deleteErr
}

type cleanupSemaphores struct {
	constraintapi.SemaphoreManager
	err               error
	held, legacyCalls int
}

func (s *cleanupSemaphores) ReleaseOwnedSemaphore(context.Context, uuid.UUID, string, string, string) error {
	if s.err != nil {
		return s.err
	}
	s.held = 0
	return nil
}
func (s *cleanupSemaphores) ReleaseSemaphore(context.Context, uuid.UUID, string, string, string, int64) error {
	s.legacyCalls++
	return errors.New("ownership-blind release must not be called")
}

type cleanupQueue struct {
	queue.Queue
	items               map[string]*queue.QueueItem
	indexed             map[string]bool
	readErr, dequeueErr error
	keepIndexOnDequeue  bool
	keepRawOnDequeue    bool
}

func (q *cleanupQueue) RunJobs(_ context.Context, _ string, _ queue.Scope, _ ulid.ULID, limit, offset int64) ([]queue.JobResponse, error) {
	if q.readErr != nil {
		return nil, q.readErr
	}
	ids := make([]string, 0, len(q.indexed))
	for id := range q.indexed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var jobs []queue.JobResponse
	for i, id := range ids {
		if int64(i) < offset || int64(i) >= offset+limit {
			continue
		}
		if item := q.items[id]; item != nil {
			jobs = append(jobs, queue.JobResponse{Raw: item})
		}
	}
	return jobs, nil
}
func (q *cleanupQueue) LoadQueueItem(_ context.Context, _ string, id string) (*queue.QueueItem, error) {
	if item := q.items[id]; item != nil {
		return item, nil
	}
	return nil, queue.ErrQueueItemNotFound
}
func (q *cleanupQueue) Dequeue(_ context.Context, _ string, item queue.QueueItem, _ ...queue.DequeueOptionFn) error {
	if q.dequeueErr != nil {
		return q.dequeueErr
	}
	if !q.keepRawOnDequeue {
		delete(q.items, item.ID)
	}
	if !q.keepIndexOnDequeue {
		delete(q.indexed, item.ID)
	}
	return nil
}

type cleanupShard struct {
	queue.QueueShard
	q *cleanupQueue
}

func (s *cleanupShard) Name() string { return "test" }
func (s *cleanupShard) OutstandingJobCount(context.Context, queue.Scope, ulid.ULID) (int, error) {
	return len(s.q.indexed), nil
}

type cleanupRegistry struct {
	queue.ShardRegistry
	shard *cleanupShard
}

func (r *cleanupRegistry) Resolve(context.Context, queue.Scope, *string) (queue.QueueShard, error) {
	return r.shard, nil
}

func newCleanupFixture(t *testing.T, jobs int) (*executor, *cleanupState, *cleanupQueue, *cleanupSemaphores) {
	t.Helper()
	id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}
	s := &cleanupState{md: sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{Semaphores: []constraintapi.Semaphore{{ID: "fn:test", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}}})}}
	q := &cleanupQueue{items: map[string]*queue.QueueItem{}, indexed: map[string]bool{}}
	for i := range jobs {
		item := &queue.QueueItem{ID: fmt.Sprint(i), FunctionID: id.FunctionID, Data: queue.Item{Identifier: state.Identifier{RunID: id.RunID, WorkflowID: id.FunctionID, AccountID: id.Tenant.AccountID, WorkspaceID: id.Tenant.EnvID}}}
		q.items[item.ID], q.indexed[item.ID] = item, true
	}
	m := &cleanupSemaphores{held: 1}
	e := &executor{log: logger.VoidLogger(), smv2: s, queue: q, semaphoreManager: m,
		shards: &cleanupRegistry{shard: &cleanupShard{q: q}}, fl: &failedRunFunctionLoader{slug: "test"},
		tracerProvider: tracing.NewOtelTracerProvider(nil, time.Millisecond)}
	return e, s, q, m
}

func TestVerifiedCancellationCleanup(t *testing.T) {
	for _, name := range []string{"success", "metadata missing", "input missing", "permit failure", "leased item", "executing item", "read failure", "dequeue failure", "delete failure", "unreadable index", "inventoried orphan", "inventoried absent", "wrong run", "large run"} {
		t.Run(name, func(t *testing.T) {
			jobs := 3
			if name == "large run" {
				jobs = 3001
			}
			e, s, q, m := newCleanupFixture(t, jobs)
			req := execution.CancelRequest{ForceLifecycleHook: true, RequireCleanupVerification: true}
			sentinel := errors.New("injected dependency failure")
			ctx := context.Background()
			switch name {
			case "metadata missing":
				s.metadataErr = sv2.ErrMetadataNotFound
			case "input missing":
				s.eventsErr = state.ErrEventNotFound
			case "permit failure":
				m.err = sentinel
			case "leased item":
				lease := ulid.Make()
				q.items["0"].LeaseID = &lease
			case "executing item":
				ctx = queue.WithJobID(ctx, "0")
			case "read failure":
				q.readErr = sentinel
			case "dequeue failure":
				q.dequeueErr = sentinel
			case "delete failure":
				s.deleteErr = sentinel
			case "unreadable index":
				q.indexed["missing-raw-record"] = true
			case "inventoried orphan":
				delete(q.indexed, "0")
				req.KnownQueueItemIDs = []string{"0", "1", "2"}
			case "inventoried absent":
				req.KnownQueueItemIDs = []string{"already-absent"}
			case "wrong run":
				q.items["0"].Data.Identifier.RunID = ulid.Make()
				req.KnownQueueItemIDs = []string{"0"}
			}
			err := e.Cancel(ctx, s.md.ID, req)
			success := name == "success" || name == "large run" || name == "inventoried orphan" || name == "inventoried absent"
			if success {
				require.NoError(t, err)
				require.True(t, s.deleted)
				require.Empty(t, q.items)
				require.Empty(t, q.indexed)
			} else {
				require.Error(t, err)
				require.False(t, s.deleted, "keep recovery metadata on incomplete cleanup")
			}
			require.Zero(t, m.legacyCalls, "strict mode must never use the old release")
			if name == "metadata missing" || name == "permit failure" || name == "input missing" || name == "leased item" || name == "executing item" || name == "wrong run" || name == "unreadable index" || name == "read failure" {
				require.Len(t, q.items, jobs)
				require.Equal(t, 1, m.held, "do not release a permit when recovery prerequisites are missing")
			}
			if name == "input missing" {
				require.ErrorIs(t, err, state.ErrEventNotFound)
			}
			if name == "leased item" {
				require.Contains(t, q.items, "0", "leave potentially executing work for separate recovery")
			}
			if name == "wrong run" {
				require.Contains(t, q.items, "0", "do not delete another run's item")
			}
			if name == "permit failure" || name == "read failure" || name == "dequeue failure" || name == "delete failure" {
				m.err, q.readErr, q.dequeueErr, s.deleteErr = nil, nil, nil, nil
				require.NoError(t, e.Cancel(context.Background(), s.md.ID, req), "retry after dependency recovery")
				require.True(t, s.deleted)
				require.Empty(t, q.items)
				require.Zero(t, m.held)
			}
		})
	}
}

func TestVerifiedCancellationRequiresInputs(t *testing.T) {
	e, s, q, m := newCleanupFixture(t, 3)
	err := e.Finalize(context.Background(), execution.FinalizeOpts{
		Metadata: s.md,
		Optional: execution.FinalizeOptional{Cancel: true, RequireCleanupVerification: true},
	})
	require.ErrorIs(t, err, ErrCancellationCleanupUnverified)
	require.False(t, s.deleted)
	require.Len(t, q.items, 3)
	require.Equal(t, 1, m.held)
}

func TestVerifiedCancellationDetectsIncompleteDequeue(t *testing.T) {
	for _, name := range []string{"index remains", "raw record remains"} {
		t.Run(name, func(t *testing.T) {
			e, s, q, m := newCleanupFixture(t, 3)
			// Independently leave either index entries or raw records behind.
			// Dequeue reports success in both cases.
			q.keepIndexOnDequeue = name == "index remains"
			q.keepRawOnDequeue = name == "raw record remains"
			err := e.Cancel(context.Background(), s.md.ID, execution.CancelRequest{RequireCleanupVerification: true})
			require.ErrorIs(t, err, ErrCancellationCleanupUnverified)
			require.False(t, s.deleted, "do not delete metadata after an unverified dequeue")
			require.Zero(t, m.legacyCalls)
			if q.keepIndexOnDequeue {
				require.Len(t, q.indexed, 3)
				require.Empty(t, q.items, "raw-record verification alone would miss this")
			} else {
				require.Empty(t, q.indexed, "index verification alone would miss this")
				require.Len(t, q.items, 3)
			}
		})
	}
}
