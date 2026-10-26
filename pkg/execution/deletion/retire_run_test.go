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
	return nil, nil
}
func (s *retirementState) Delete(context.Context, sv2.ID, ...sv2.DeleteOption) error {
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
	for _, name := range []string{"success", "legacy counter not released", "missing metadata", "missing input", "both missing", "already removed", "inventoried orphan", "wrong tenant", "wrong metadata", "expired lease", "current job", "future run", "unknown manual semaphore", "extra index item", "wrong indexed item", "queue read error", "state read error", "input read error", "dequeue error", "raw verification", "index verification", "delete error", "hold lost", "no guard"} {
		t.Run(name, func(t *testing.T) {
			id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}
			sem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDFnKey(id.FunctionID, "event.data.app_id"), Release: constraintapi.SemaphoreReleaseManual, Weight: 1}
			req := RunRetirement{ID: id, JobID: "job", Before: time.Now().Add(time.Minute), RetiredSemaphoreID: sem.ID}
			q := &retirementShard{count: 1, indexID: "job", item: &queue.QueueItem{ID: "job", FunctionID: id.FunctionID, Data: queue.Item{Kind: queue.KindStart, Identifier: state.Identifier{RunID: id.RunID, WorkflowID: id.FunctionID, AccountID: id.Tenant.AccountID, WorkspaceID: id.Tenant.EnvID, AppID: id.Tenant.AppID, Semaphores: []constraintapi.Semaphore{sem}}, Semaphores: []constraintapi.Semaphore{sem}}}}
			s := &retirementState{md: sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{Semaphores: []constraintapi.Semaphore{sem}})}}
			calls := 0
			guard := func(context.Context) error {
				calls++
				if name == "hold lost" && calls > 1 {
					return errors.New("hold lost")
				}
				return nil
			}
			ctx := context.Background()
			failure := errors.New("injected failure")
			wantErr := true
			switch name {
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
			case "unknown manual semaphore":
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
			require.True(t, s.deleted)
			// There is deliberately no semaphore manager to call: the counter is
			// preserved, irrespective of whether the old run actually owned it.
			_, err = RetireMaintenanceRun(ctx, q, s, req, guard)
			require.NoError(t, err)
		})
	}
}
