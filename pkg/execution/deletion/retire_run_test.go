package deletion

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type retirementState struct {
	sv2.RunService
	md                            sv2.Metadata
	absent, eventsAbsent, deleted bool
	readErr, eventsErr, deleteErr error
	defers                        map[string]sv2.Defer
	defersErr                     error
	deleteMetadataFirst           bool
}

func (s *retirementState) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	if s.readErr != nil {
		return sv2.Metadata{}, s.readErr
	}
	if s.absent || s.deleted {
		return sv2.Metadata{}, sv2.ErrMetadataNotFound
	}
	return s.md, nil
}
func (s *retirementState) LoadEvents(context.Context, sv2.ID) ([]json.RawMessage, error) {
	if s.eventsErr != nil {
		return nil, s.eventsErr
	}
	if s.eventsAbsent || s.deleted {
		return nil, state.ErrEventNotFound
	}
	return []json.RawMessage{json.RawMessage(`{"name":"maintenance"}`)}, nil
}
func (s *retirementState) LoadDefers(context.Context, sv2.ID) (map[string]sv2.Defer, error) {
	return s.defers, s.defersErr
}
func (s *retirementState) Delete(context.Context, sv2.ID, ...sv2.DeleteOption) error {
	if s.deleteMetadataFirst {
		s.absent = true
	}
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = true
	return nil
}

type retirementShard struct {
	queue.QueueShard
	item                *queue.QueueItem
	count               int
	indexID             string
	keepRaw, keepIndex  bool
	dequeues            int
	readErr, dequeueErr error
	beforeDequeue       func()
	checkpoint          *queue.MaintenanceRetirement
	checkpointReadErr   error
	checkpointSaveErr   error
	checkpointDeleteErr error
}

func (q *retirementShard) LoadMaintenanceRetirement(_ context.Context, jobID string) (*queue.MaintenanceRetirement, error) {
	if q.checkpointReadErr != nil {
		return nil, q.checkpointReadErr
	}
	if q.checkpoint == nil || q.checkpoint.JobID != jobID {
		return nil, nil
	}
	value := *q.checkpoint
	return &value, nil
}

func (q *retirementShard) SaveMaintenanceRetirement(_ context.Context, value queue.MaintenanceRetirement) error {
	if q.checkpointSaveErr != nil {
		return q.checkpointSaveErr
	}
	if q.checkpoint != nil && *q.checkpoint != value {
		return errors.New("different checkpoint")
	}
	q.checkpoint = &value
	return nil
}

func (q *retirementShard) DeleteMaintenanceRetirement(_ context.Context, value queue.MaintenanceRetirement) error {
	if q.checkpointDeleteErr != nil {
		return q.checkpointDeleteErr
	}
	if q.checkpoint != nil && *q.checkpoint != value {
		return errors.New("different checkpoint")
	}
	q.checkpoint = nil
	return nil
}

func TestRetirementRetryPreservesInspection(t *testing.T) {
	for _, name := range []string{"missing metadata", "partial metadata deletion", "checkpoint removal failure", "checkpoint read failure", "checkpoint save failure"} {
		t.Run(name, func(t *testing.T) {
			id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}
			sem := constraintapi.Semaphore{ID: "retired", Release: constraintapi.SemaphoreReleaseManual}
			q := &retirementShard{count: 1, indexID: "job", item: &queue.QueueItem{ID: "job", FunctionID: id.FunctionID, Data: queue.Item{Kind: queue.KindStart, Identifier: state.Identifier{RunID: id.RunID, WorkflowID: id.FunctionID, AccountID: id.Tenant.AccountID, WorkspaceID: id.Tenant.EnvID, AppID: id.Tenant.AppID}, Semaphores: []constraintapi.Semaphore{sem}}}}
			s := &retirementState{absent: name != "partial metadata deletion", md: sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{Semaphores: []constraintapi.Semaphore{sem}})}}
			req := RunRetirement{ID: id, JobID: "job", Before: time.Now().Add(time.Minute), RetiredSemaphoreID: sem.ID, RequireRetiredSemaphore: true, DryRun: true}
			guard := func(context.Context) error { return nil }
			_, err := RetireMaintenanceRun(context.Background(), q, s, req, guard)
			require.NoError(t, err)
			require.Nil(t, q.checkpoint, "dry run must not write a checkpoint")
			require.Zero(t, q.dequeues)
			req.DryRun = false
			failure := errors.New("injected failure")
			switch name {
			case "checkpoint removal failure":
				q.checkpointDeleteErr = failure
			case "checkpoint read failure":
				q.checkpointReadErr = failure
			case "checkpoint save failure":
				q.checkpointSaveErr = failure
			default:
				s.deleteErr = failure
				s.deleteMetadataFirst = name == "partial metadata deletion"
			}
			_, err = RetireMaintenanceRun(context.Background(), q, s, req, guard)
			require.ErrorIs(t, err, failure)
			if name == "checkpoint read failure" || name == "checkpoint save failure" {
				require.Zero(t, q.dequeues)
				require.NotNil(t, q.item)
				require.False(t, s.deleted)
				return
			}
			require.Nil(t, q.item)
			require.Zero(t, q.count)
			require.NotNil(t, q.checkpoint, "evidence must survive failure after queue deletion")
			if !s.deleted {
				// Inspection evidence is specific to every field of the original manifest.
				for _, change := range []func(*RunRetirement){
					func(r *RunRetirement) { r.Before = r.Before.Add(time.Millisecond) },
					func(r *RunRetirement) { r.ID.Tenant.AccountID = uuid.New() },
					func(r *RunRetirement) { r.ID.Tenant.EnvID = uuid.New() },
					func(r *RunRetirement) { r.ID.Tenant.AppID = uuid.New() },
					func(r *RunRetirement) { r.ID.FunctionID = uuid.New() },
					func(r *RunRetirement) { r.ID.RunID = ulid.Make() },
					func(r *RunRetirement) { r.JobID = "other-job" },
					func(r *RunRetirement) { r.RetiredSemaphoreID = "other-semaphore" },
				} {
					changed := req
					change(&changed)
					_, err := RetireMaintenanceRun(context.Background(), q, s, changed, guard)
					require.ErrorIs(t, err, ErrRunRetirementUnsafe)
					require.False(t, s.deleted)
				}
				_, err = RetireMaintenanceRun(context.Background(), q, s, req, func(context.Context) error { return failure })
				require.ErrorIs(t, err, failure, "checkpoint must not bypass current guards")
				s.defers = map[string]sv2.Defer{"pending": {}}
				_, err = RetireMaintenanceRun(context.Background(), q, s, req, guard)
				require.ErrorIs(t, err, ErrRunRetirementUnsafe, "checkpoint must not bypass deferred-work checks")
				s.defers = nil
				s.eventsErr = failure
				_, err = RetireMaintenanceRun(context.Background(), q, s, req, guard)
				require.ErrorIs(t, err, failure, "checkpoint must not bypass unknown state reads")
				s.eventsErr = nil
				req.DryRun = true
				_, err = RetireMaintenanceRun(context.Background(), q, s, req, guard)
				require.NoError(t, err)
				require.False(t, s.deleted)
				require.NotNil(t, q.checkpoint, "dry-run retries must preserve their checkpoint")
				req.DryRun = false
			}
			s.deleteErr = nil
			q.checkpointDeleteErr = nil
			_, err = RetireMaintenanceRun(context.Background(), q, s, req, guard)
			require.NoError(t, err)
			require.True(t, s.deleted)
			require.Equal(t, 1, q.dequeues)
			require.Nil(t, q.checkpoint, "verified cleanup removes its checkpoint")
		})
	}
}

func (q *retirementShard) MaintenanceLeaseCount(context.Context, queue.Scope) (int64, error) {
	return 0, nil
}
func (q *retirementShard) DequeueMaintenance(ctx context.Context, item queue.QueueItem) error {
	if q.beforeDequeue != nil {
		q.beforeDequeue()
	}
	if q.item != nil && (q.item.LeaseID != nil || q.item.CapacityLease != nil) {
		return errors.New("atomic lease refusal")
	}
	return q.Dequeue(ctx, item)
}

func (q *retirementShard) LoadQueueItem(context.Context, string) (*queue.QueueItem, error) {
	if q.readErr != nil {
		return nil, q.readErr
	}
	if q.item == nil {
		return nil, queue.ErrQueueItemNotFound
	}
	return q.item, nil
}

func TestRetirementLateLeaseAndLegacyEvidence(t *testing.T) {
	for _, name := range []string{"late lease", "capacity holder", "no legacy evidence"} {
		t.Run(name, func(t *testing.T) {
			id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}
			sem := constraintapi.Semaphore{ID: "retired", Release: constraintapi.SemaphoreReleaseManual}
			s := &retirementState{md: sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{Semaphores: []constraintapi.Semaphore{sem}})}}
			q := &retirementShard{count: 1, indexID: "job", item: &queue.QueueItem{ID: "job", FunctionID: id.FunctionID, Data: queue.Item{Kind: queue.KindStart, Identifier: state.Identifier{RunID: id.RunID, WorkflowID: id.FunctionID, AccountID: id.Tenant.AccountID, WorkspaceID: id.Tenant.EnvID, AppID: id.Tenant.AppID}}}}
			if name == "late lease" {
				q.beforeDequeue = func() { lease := ulid.Make(); q.item.LeaseID = &lease }
			}
			if name == "capacity holder" {
				q.item.CapacityLease = &queue.CapacityLease{LeaseID: ulid.Make()}
			}
			if name == "no legacy evidence" {
				s.md.Config.Semaphores = nil
			}
			_, err := RetireMaintenanceRun(context.Background(), q, s, RunRetirement{ID: id, JobID: "job", Before: time.Now().Add(time.Minute), RetiredSemaphoreID: "retired", RequireRetiredSemaphore: true}, func(context.Context) error { return nil })
			require.Error(t, err)
			require.NotNil(t, q.item)
			require.False(t, s.deleted)
			require.Zero(t, q.dequeues)
		})
	}
}
func (q *retirementShard) OutstandingJobCount(context.Context, queue.Scope, ulid.ULID) (int, error) {
	return q.count, nil
}
func (q *retirementShard) RunJobs(context.Context, queue.Scope, ulid.ULID, int64, int64) ([]queue.JobResponse, error) {
	return []queue.JobResponse{{JobID: q.indexID}}, nil
}
func (q *retirementShard) Dequeue(context.Context, queue.QueueItem, ...queue.DequeueOptionFn) error {
	q.dequeues++
	if q.dequeueErr != nil {
		return q.dequeueErr
	}
	if !q.keepRaw {
		q.item = nil
	}
	if !q.keepIndex {
		q.count = 0
	}
	return nil
}

func TestRetireMaintenanceRun(t *testing.T) {
	for _, name := range []string{"success", "legacy counter not released", "missing metadata", "missing input", "both missing", "already removed", "inventoried orphan", "wrong tenant", "wrong metadata", "expired lease", "current job", "future run", "unknown manual-release semaphore", "extra index item", "wrong indexed item", "queue read error", "state read error", "input read error", "deferred work", "deferred work without metadata", "defer read error", "dequeue error", "raw verification", "index verification", "delete error", "function no longer paused", "no guard"} {
		t.Run(name, func(t *testing.T) {
			id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}
			sem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDFnKey(id.FunctionID, "event.data.app_id"), Release: constraintapi.SemaphoreReleaseManual, Weight: 1}
			req := RunRetirement{ID: id, JobID: "job", Before: time.Now().Add(time.Minute), RetiredSemaphoreID: sem.ID}
			q := &retirementShard{count: 1, indexID: "job", item: &queue.QueueItem{ID: "job", FunctionID: id.FunctionID, Data: queue.Item{Kind: queue.KindStart, Identifier: state.Identifier{RunID: id.RunID, WorkflowID: id.FunctionID, AccountID: id.Tenant.AccountID, WorkspaceID: id.Tenant.EnvID, AppID: id.Tenant.AppID, Semaphores: []constraintapi.Semaphore{sem}}, Semaphores: []constraintapi.Semaphore{sem}}}}
			s := &retirementState{md: sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{Semaphores: []constraintapi.Semaphore{sem}})}}
			calls := 0
			guard := func(context.Context) error {
				calls++
				if name == "function no longer paused" && calls > 1 {
					return errors.New("function no longer paused")
				}
				return nil
			}
			ctx := context.Background()
			failure := errors.New("injected failure")
			wantErr := true
			switch name {
			case "deferred work", "deferred work without metadata":
				s.defers = map[string]sv2.Defer{"pending": {}}
				s.absent = name == "deferred work without metadata"
			case "defer read error":
				s.defersErr = failure
			case "success", "legacy counter not released":
				wantErr = false
			case "missing metadata":
				s.absent = true
				wantErr = false
			case "missing input":
				s.eventsAbsent = true
				wantErr = false
			case "both missing":
				s.absent = true
				s.eventsAbsent = true
				wantErr = false
			case "already removed":
				q.item = nil
				q.count = 0
				s.absent = true
				s.eventsAbsent = true
				wantErr = false
			case "inventoried orphan":
				q.count = 0
				wantErr = false
			case "wrong tenant":
				q.item.Data.Identifier.AccountID = uuid.New()
			case "wrong metadata":
				s.md.ID.FunctionID = uuid.New()
			case "expired lease":
				lease := ulid.MustNew(ulid.Timestamp(time.Now().Add(-time.Hour)), nil)
				q.item.LeaseID = &lease
			case "current job":
				ctx = queue.WithJobID(ctx, "job")
			case "future run":
				req.Before = time.Now().Add(-time.Hour)
			case "unknown manual-release semaphore":
				req.RetiredSemaphoreID = "different"
			case "extra index item":
				q.count = 2
			case "wrong indexed item":
				q.indexID = "other"
			case "queue read error":
				q.readErr = failure
			case "state read error":
				s.readErr = failure
			case "input read error":
				s.eventsErr = failure
			case "dequeue error":
				q.dequeueErr = failure
			case "raw verification":
				q.keepRaw = true
			case "index verification":
				q.keepIndex = true
			case "delete error":
				s.deleteErr = failure
			case "no guard":
				guard = nil
			}
			if name == "success" {
				req.DryRun = true
				_, err := RetireMaintenanceRun(ctx, q, s, req, guard)
				require.NoError(t, err)
				require.Zero(t, q.dequeues)
				require.False(t, s.deleted)
				req.DryRun = false
			}
			_, err := RetireMaintenanceRun(ctx, q, s, req, guard)
			if wantErr {
				require.Error(t, err)
				require.False(t, s.deleted)
				if name != "dequeue error" && name != "raw verification" && name != "index verification" && name != "delete error" {
					require.Zero(t, q.dequeues)
				}
				return
			}
			require.NoError(t, err)
			require.Nil(t, q.item)
			require.Zero(t, q.count)
			if name != "already removed" {
				require.True(t, s.deleted)
			}
			// There is deliberately no semaphore manager to call: the counter is
			// preserved, irrespective of whether the old run actually owned it.
			_, err = RetireMaintenanceRun(ctx, q, s, req, guard)
			require.NoError(t, err)
		})
	}
}
