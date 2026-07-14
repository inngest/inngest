package queue

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	pb "github.com/inngest/inngest/proto/gen/queue/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestQueueItemProtoRoundTrip guards against dropping persisted queue item
// fields when items cross the queue-proxy boundary.
func TestQueueItemProtoRoundTrip(t *testing.T) {
	jobID := "job-1"
	queueName := "queue-system"
	leaseID := ulid.Make()
	capacityLeaseID := ulid.Make()
	idempotencyPeriod := 2 * time.Minute
	batchID := ulid.Make()
	originalRunID := ulid.Make()
	replayID := uuid.New()
	priorityFactor := int64(42)
	maxAttempts := 5
	parallelCoalesceKey := "parallel-key"

	original := QueueItem{
		ID:               "item-1",
		EarliestPeekTime: time.Now().Add(-time.Minute).UnixMilli(),
		AtMS:             time.Now().UnixMilli(),
		WallTimeMS:       time.Now().Add(-2 * time.Minute).UnixMilli(),
		FunctionID:       uuid.New(),
		WorkspaceID:      uuid.New(),
		LeaseID:          &leaseID,
		GenerationID:     7,
		Data: Item{
			JobID:       &jobID,
			GroupID:     "group-1",
			WorkspaceID: uuid.New(),
			Kind:        KindStart,
			Identifier: state.Identifier{
				RunID:           ulid.Make(),
				WorkflowID:      uuid.New(),
				WorkflowVersion: 3,
				EventID:         ulid.Make(),
				BatchID:         &batchID,
				EventIDs:        []ulid.ULID{ulid.Make(), ulid.Make()},
				Key:             "identifier-key",
				AccountID:       uuid.New(),
				WorkspaceID:     uuid.New(),
				AppID:           uuid.New(),
				OriginalRunID:   &originalRunID,
				ReplayID:        &replayID,
				PriorityFactor:  &priorityFactor,
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key:                       "f:fn:key-a",
						Hash:                      "hash-a",
						Limit:                     10,
						UnhashedEvaluatedKeyValue: "identifier-customer",
					},
				},
				Semaphores: []constraintapi.Semaphore{
					{
						ID:               "fn:identifier",
						EvaluatedKeyHash: "identifier-hash",
						Weight:           2,
						Release:          constraintapi.SemaphoreReleaseManual,
					},
				},
			},
			Attempt:     2,
			MaxAttempts: &maxAttempts,
			Payload:     json.RawMessage(`{"edge":"step"}`),
			Metadata: map[string]any{
				"trace": "abc",
				"count": float64(3),
			},
			QueueName: &queueName,
			RunInfo: &RunInfo{
				Latency:             3 * time.Second,
				SojournDelay:        4 * time.Second,
				Priority:            9,
				QueueShardName:      "queue-a",
				ContinueCount:       2,
				RefilledFromBacklog: "backlog-a",
				CapacityLease: &CapacityLease{
					LeaseID:    capacityLeaseID,
					IssuedAtMS: time.Now().UnixMilli(),
				},
				ScavengeCount: 1,
			},
			Throttle: &Throttle{
				Key:                 "throttle-key",
				Limit:               10,
				Burst:               20,
				Period:              30,
				UnhashedThrottleKey: "raw-throttle",
				KeyExpressionHash:   "expr-hash",
			},
			Singleton: &Singleton{
				Key:  "singleton-key",
				Mode: enums.SingletonModeCancel,
			},
			CustomConcurrencyKeys: []state.CustomConcurrency{
				{
					Key:                       "a:acct:key-b",
					Hash:                      "hash-b",
					Limit:                     4,
					UnhashedEvaluatedKeyValue: "item-customer",
				},
			},
			PriorityFactor:      &priorityFactor,
			ParallelMode:        enums.ParallelModeRace,
			ParallelCoalesceKey: &parallelCoalesceKey,
			Semaphores: []constraintapi.Semaphore{
				{
					ID:               "app:item",
					EvaluatedKeyHash: "item-hash",
					Weight:           1,
					Release:          constraintapi.SemaphoreReleaseAuto,
				},
			},
		},
		QueueName:         &queueName,
		IdempotencyPeriod: &idempotencyPeriod,
		RefilledFrom:      "backlog-a",
		RefilledAt:        time.Now().Add(-30 * time.Second).UnixMilli(),
		EnqueuedAt:        time.Now().Add(-3 * time.Minute).UnixMilli(),
		ScavengeCount:     3,
		CapacityLease: &CapacityLease{
			LeaseID:    capacityLeaseID,
			IssuedAtMS: time.Now().UnixMilli(),
		},
	}

	msg, err := QueueItemToProto(original)
	require.NoError(t, err)

	roundTripped, err := QueueItemFromProto(msg)
	require.NoError(t, err)

	require.Equal(t, original.ID, roundTripped.ID)
	require.Equal(t, original.EarliestPeekTime, roundTripped.EarliestPeekTime)
	require.Equal(t, original.AtMS, roundTripped.AtMS)
	require.Equal(t, original.WallTimeMS, roundTripped.WallTimeMS)
	require.Equal(t, original.FunctionID, roundTripped.FunctionID)
	require.Equal(t, original.WorkspaceID, roundTripped.WorkspaceID)
	require.Equal(t, original.LeaseID.String(), roundTripped.LeaseID.String())
	require.Equal(t, original.GenerationID, roundTripped.GenerationID)
	require.Equal(t, original.QueueName, roundTripped.QueueName)
	require.Equal(t, original.IdempotencyPeriod, roundTripped.IdempotencyPeriod)
	require.Equal(t, original.RefilledFrom, roundTripped.RefilledFrom)
	require.Equal(t, original.RefilledAt, roundTripped.RefilledAt)
	require.Equal(t, original.EnqueuedAt, roundTripped.EnqueuedAt)
	require.Equal(t, original.ScavengeCount, roundTripped.ScavengeCount)
	require.Equal(t, original.CapacityLease.LeaseID, roundTripped.CapacityLease.LeaseID)
	require.Equal(t, original.CapacityLease.IssuedAtMS, roundTripped.CapacityLease.IssuedAtMS)
	require.Equal(t, time.UnixMilli(original.EnqueuedAt), roundTripped.Data.EnqueuedAt)
	require.Equal(t, time.UnixMilli(original.AtMS), roundTripped.Data.At)
	assertItemEqual(t, original.Data, roundTripped.Data)
}

// TestItemProtoRoundTrip_NilOptionalFields guards nil optional fields so the
// proxy does not materialize absent queue data as non-nil zero values.
func TestItemProtoRoundTrip_NilOptionalFields(t *testing.T) {
	original := Item{
		WorkspaceID: uuid.New(),
		Kind:        KindEdge,
		Identifier: state.Identifier{
			RunID:       ulid.Make(),
			WorkflowID:  uuid.New(),
			EventID:     ulid.Make(),
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
		},
		Payload: json.RawMessage(`null`),
	}

	msg, err := ItemToProto(original)
	require.NoError(t, err)

	roundTripped, err := ItemFromProto(msg)
	require.NoError(t, err)

	require.Nil(t, roundTripped.JobID)
	require.Nil(t, roundTripped.MaxAttempts)
	require.Nil(t, roundTripped.QueueName)
	require.Nil(t, roundTripped.RunInfo)
	require.Nil(t, roundTripped.Throttle)
	require.Nil(t, roundTripped.Singleton)
	require.Nil(t, roundTripped.PriorityFactor)
	require.Nil(t, roundTripped.ParallelCoalesceKey)
	require.Nil(t, roundTripped.Metadata)
	require.Equal(t, json.RawMessage(`null`), roundTripped.Payload)
	require.Equal(t, original.WorkspaceID, roundTripped.WorkspaceID)
	require.Equal(t, original.Identifier.RunID, roundTripped.Identifier.RunID)
}

// TestEnqueueRequestTimeContract guards the enqueue boundary: scheduled time is
// carried by EnqueueRequest.at, not by Item.At, and enqueue time is assigned by
// the queue rather than producer-provided Item.EnqueuedAt.
func TestEnqueueRequestTimeContract(t *testing.T) {
	itemAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	itemEnqueuedAt := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

	requestAt := time.Date(2026, 1, 2, 4, 0, 0, 0, time.UTC)

	msg, err := ItemToProto(Item{
		WorkspaceID: uuid.New(),
		Kind:        KindEdge,
		Identifier:  validIdentifier(),
		Payload:     json.RawMessage(`{"edge":"step"}`),
		At:          itemAt,
		EnqueuedAt:  itemEnqueuedAt,
	})
	require.NoError(t, err)

	req := &pb.EnqueueRequest{
		Item: msg,
		At:   timestamppb.New(requestAt),
	}

	roundTripped, err := ItemFromProto(req.GetItem())
	require.NoError(t, err)

	// roundTripped.At and EnqueuedAt are zero because the enqueue proto request
	// does not carry those times and hence cannot be re-constructed from the item proto.
	require.True(t, roundTripped.At.IsZero())
	require.True(t, roundTripped.EnqueuedAt.IsZero())
	require.Equal(t, requestAt, req.GetAt().AsTime())
}

// TestQueueItemFromProtoReconstructsItemTimes guards the dequeue/requeue
// boundary: item runtime times are derived from the outer QueueItem envelope.
func TestQueueItemFromProtoReconstructsItemTimes(t *testing.T) {
	at := time.Date(2026, 1, 2, 4, 0, 0, 0, time.UTC)
	enqueuedAt := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

	item, err := QueueItemFromProto(&pb.QueueItem{
		AtMs:        at.UnixMilli(),
		EnqueuedAt:  enqueuedAt.UnixMilli(),
		FunctionId:  uuid.NewString(),
		WorkspaceId: uuid.NewString(),
		Data: &pb.Item{
			WorkspaceId: uuid.NewString(),
			Identifier:  validIdentifierProto(),
		},
	})
	require.NoError(t, err)
	require.Equal(t, at.UnixMilli(), item.Data.At.UnixMilli())
	require.Equal(t, enqueuedAt.UnixMilli(), item.Data.EnqueuedAt.UnixMilli())
}

// TestEnqueueOptionsProtoRoundTrip guards producer option drift between the Go
// interface and the proto request schema.
func TestEnqueueOptionsProtoRoundTrip(t *testing.T) {
	idempotencyPeriod := 30 * time.Second
	original := EnqueueOpts{
		PassthroughJobId:       true,
		ForceQueueShardName:    "queue-a",
		NormalizeFromBacklogID: "backlog-a",
		IdempotencyPeriod:      &idempotencyPeriod,
	}

	msg := EnqueueOptionsToProto(original)
	roundTripped := EnqueueOptionsFromProto(msg)

	require.Equal(t, original, roundTripped)
	require.Equal(t, EnqueueOpts{}, EnqueueOptionsFromProto(nil))
}

// TestQueueItemFromProtoInvalidIDs guards against malformed proxy payloads
// silently becoming zero UUIDs or ULIDs.
func TestQueueItemFromProtoInvalidIDs(t *testing.T) {
	_, err := QueueItemFromProto(&pb.QueueItem{
		FunctionId: "not-a-uuid",
	})
	require.ErrorContains(t, err, "queue item function_id")

	_, err = QueueItemFromProto(&pb.QueueItem{
		FunctionId:  uuid.NewString(),
		WorkspaceId: uuid.NewString(),
		LeaseId:     protoStringPtr("not-a-ulid"),
	})
	require.ErrorContains(t, err, "queue item lease_id")
}

// TestQueueItemProtoNestedErrorPropagation guards top-level queue item
// conversion from swallowing errors produced by nested item metadata and
// runtime lease conversion.
func TestQueueItemProtoNestedErrorPropagation(t *testing.T) {
	_, err := QueueItemToProto(QueueItem{
		FunctionID:  uuid.New(),
		WorkspaceID: uuid.New(),
		Data: Item{
			WorkspaceID: uuid.New(),
			Identifier:  validIdentifier(),
			Metadata:    map[string]any{"bad": make(chan struct{})},
		},
	})
	require.ErrorContains(t, err, "marshal item metadata")

	_, err = QueueItemFromProto(&pb.QueueItem{
		FunctionId:  uuid.NewString(),
		WorkspaceId: uuid.NewString(),
		Data: &pb.Item{
			WorkspaceId: uuid.NewString(),
			Identifier:  validIdentifierProto(),
			Metadata:    []byte(`{`),
		},
	})
	require.ErrorContains(t, err, "unmarshal item metadata")

	_, err = QueueItemFromProto(&pb.QueueItem{
		FunctionId:  uuid.NewString(),
		WorkspaceId: uuid.NewString(),
		Data: &pb.Item{
			WorkspaceId: uuid.NewString(),
			Identifier:  validIdentifierProto(),
		},
		CapacityLease: &pb.CapacityLease{LeaseId: "bad"},
	})
	require.ErrorContains(t, err, "capacity lease lease_id")
}

// TestItemFromProtoInvalidIDs guards nested item identifiers against malformed
// proxy payloads that would otherwise decode to zero values.
func TestItemFromProtoInvalidIDs(t *testing.T) {
	_, err := ItemFromProto(&pb.Item{
		WorkspaceId: "not-a-uuid",
	})
	require.ErrorContains(t, err, "item workspace_id")

	_, err = ItemFromProto(&pb.Item{
		WorkspaceId: uuid.NewString(),
		Identifier: &pb.Identifier{
			RunId: "not-a-ulid",
		},
	})
	require.ErrorContains(t, err, "identifier run_id")
}

// TestIdentifierProtoRoundTrip guards direct Identifier conversions used by
// queue item payloads.
func TestIdentifierProtoRoundTrip(t *testing.T) {
	batchID := ulid.Make()
	originalRunID := ulid.Make()
	replayID := uuid.New()
	priorityFactor := int64(11)
	original := state.Identifier{
		RunID:           ulid.Make(),
		WorkflowID:      uuid.New(),
		WorkflowVersion: 9,
		EventID:         ulid.Make(),
		BatchID:         &batchID,
		EventIDs:        []ulid.ULID{ulid.Make(), ulid.Make()},
		Key:             "key",
		AccountID:       uuid.New(),
		WorkspaceID:     uuid.New(),
		AppID:           uuid.New(),
		OriginalRunID:   &originalRunID,
		ReplayID:        &replayID,
		PriorityFactor:  &priorityFactor,
		CustomConcurrencyKeys: []state.CustomConcurrency{
			{
				Key:                       "f:fn:key",
				Hash:                      "hash",
				Limit:                     2,
				UnhashedEvaluatedKeyValue: "customer-1",
			},
		},
		Semaphores: []constraintapi.Semaphore{
			{ID: "app:test", EvaluatedKeyHash: "hash", Weight: 3, Release: constraintapi.SemaphoreReleaseManual},
		},
	}

	msg, err := IdentifierToProto(original)
	require.NoError(t, err)

	roundTripped, err := IdentifierFromProto(msg)
	require.NoError(t, err)
	require.Equal(t, original, roundTripped)
}

// TestIdentifierFromProtoInvalidIDs guards every parsed Identifier ID field
// against malformed proxy input.
func TestIdentifierFromProtoInvalidIDs(t *testing.T) {
	valid := validIdentifierProto()

	tests := []struct {
		name    string
		mutate  func(*pb.Identifier)
		wantErr string
	}{
		{
			name:    "workflow id",
			mutate:  func(msg *pb.Identifier) { msg.WorkflowId = "bad" },
			wantErr: "identifier workflow_id",
		},
		{
			name:    "event id",
			mutate:  func(msg *pb.Identifier) { msg.EventId = "bad" },
			wantErr: "identifier event_id",
		},
		{
			name:    "event ids",
			mutate:  func(msg *pb.Identifier) { msg.EventIds = []string{"bad"} },
			wantErr: "identifier event_ids",
		},
		{
			name:    "account id",
			mutate:  func(msg *pb.Identifier) { msg.AccountId = "bad" },
			wantErr: "identifier account_id",
		},
		{
			name:    "workspace id",
			mutate:  func(msg *pb.Identifier) { msg.WorkspaceId = "bad" },
			wantErr: "identifier workspace_id",
		},
		{
			name:    "app id",
			mutate:  func(msg *pb.Identifier) { msg.AppId = "bad" },
			wantErr: "identifier app_id",
		},
		{
			name:    "batch id",
			mutate:  func(msg *pb.Identifier) { msg.BatchId = protoStringPtr("bad") },
			wantErr: "identifier batch_id",
		},
		{
			name:    "original run id",
			mutate:  func(msg *pb.Identifier) { msg.OriginalRunId = protoStringPtr("bad") },
			wantErr: "identifier original_run_id",
		},
		{
			name:    "replay id",
			mutate:  func(msg *pb.Identifier) { msg.ReplayId = protoStringPtr("bad") },
			wantErr: "identifier replay_id",
		},
		{
			name:    "semaphore release",
			mutate:  func(msg *pb.Identifier) { msg.Semaphores = []*pb.Semaphore{{Release: "bad"}} },
			wantErr: "semaphore release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := proto.Clone(valid).(*pb.Identifier)
			tt.mutate(msg)
			_, err := IdentifierFromProto(msg)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestRunInfoProtoRoundTrip guards direct runtime metadata conversion,
// including nil durations and nested capacity leases.
func TestRunInfoProtoRoundTrip(t *testing.T) {
	leaseID := ulid.Make()
	original := &RunInfo{
		Latency:             5 * time.Second,
		SojournDelay:        6 * time.Second,
		Priority:            7,
		QueueShardName:      "queue-a",
		ContinueCount:       8,
		RefilledFromBacklog: "backlog-a",
		CapacityLease:       &CapacityLease{LeaseID: leaseID, IssuedAtMS: 123},
		ScavengeCount:       9,
	}

	msg := RunInfoToProto(original)
	roundTripped, err := RunInfoFromProto(msg)
	require.NoError(t, err)
	require.Equal(t, original, roundTripped)

	roundTripped, err = RunInfoFromProto(&pb.RunInfo{})
	require.NoError(t, err)
	require.Equal(t, time.Duration(0), roundTripped.Latency)
	require.Equal(t, time.Duration(0), roundTripped.SojournDelay)
}

// TestCapacityLeaseProtoRoundTrip guards direct capacity lease conversion and
// malformed lease IDs.
func TestCapacityLeaseProtoRoundTrip(t *testing.T) {
	original := &CapacityLease{LeaseID: ulid.Make(), IssuedAtMS: 123}

	msg := CapacityLeaseToProto(original)
	roundTripped, err := CapacityLeaseFromProto(msg)
	require.NoError(t, err)
	require.Equal(t, original.LeaseID, roundTripped.LeaseID)
	require.Equal(t, original.IssuedAtMS, roundTripped.IssuedAtMS)

	require.Nil(t, CapacityLeaseToProto(nil))
	roundTripped, err = CapacityLeaseFromProto(nil)
	require.NoError(t, err)
	require.Nil(t, roundTripped)

	_, err = CapacityLeaseFromProto(&pb.CapacityLease{LeaseId: "bad"})
	require.ErrorContains(t, err, "capacity lease lease_id")
}

// TestLeafProtoRoundTrips guards small direct converters that are easy to
// accidentally break while changing queue metadata fields.
func TestLeafProtoRoundTrips(t *testing.T) {
	throttle := &Throttle{
		Key:                 "key",
		Limit:               1,
		Burst:               2,
		Period:              3,
		UnhashedThrottleKey: "customer-1",
		KeyExpressionHash:   "hash",
	}
	require.Equal(t, throttle, ThrottleFromProto(ThrottleToProto(throttle)))
	require.Nil(t, ThrottleToProto(nil))
	require.Nil(t, ThrottleFromProto(nil))

	singleton := &Singleton{Key: "singleton", Mode: enums.SingletonModeCancel}
	singletonMsg := SingletonToProto(singleton)
	roundTrippedSingleton, err := SingletonFromProto(singletonMsg)
	require.NoError(t, err)
	require.Equal(t, singleton, roundTrippedSingleton)
	require.Nil(t, SingletonToProto(nil))
	roundTrippedSingleton, err = SingletonFromProto(nil)
	require.NoError(t, err)
	require.Nil(t, roundTrippedSingleton)

	roundTrippedSingleton, err = SingletonFromProto(&pb.Singleton{Key: "default"})
	require.NoError(t, err)
	require.Equal(t, enums.SingletonModeSkip, roundTrippedSingleton.Mode)

	_, err = SingletonFromProto(&pb.Singleton{Mode: "unknown"})
	require.ErrorContains(t, err, "singleton mode")

	keys := []state.CustomConcurrency{
		{Key: "key", Hash: "hash", Limit: 1, UnhashedEvaluatedKeyValue: "customer-1"},
	}
	require.Equal(t, keys, CustomConcurrencySliceFromProto(CustomConcurrencySliceToProto(keys)))
	require.Nil(t, CustomConcurrencySliceToProto(nil))
	require.Nil(t, CustomConcurrencySliceFromProto(nil))

	semaphores := []constraintapi.Semaphore{
		{ID: "app:test", EvaluatedKeyHash: "hash", Weight: 1, Release: constraintapi.SemaphoreReleaseManual},
	}
	roundTrippedSemaphores, err := SemaphoreSliceFromProto(SemaphoreSliceToProto(semaphores))
	require.NoError(t, err)
	require.Equal(t, semaphores, roundTrippedSemaphores)
	require.Nil(t, SemaphoreSliceToProto(nil))
	roundTrippedSemaphores, err = SemaphoreSliceFromProto(nil)
	require.NoError(t, err)
	require.Nil(t, roundTrippedSemaphores)
	roundTrippedSemaphores, err = SemaphoreSliceFromProto([]*pb.Semaphore{{Id: "app:test"}})
	require.NoError(t, err)
	require.Equal(t, constraintapi.SemaphoreReleaseAuto, roundTrippedSemaphores[0].Release)
	_, err = SemaphoreSliceFromProto([]*pb.Semaphore{{Release: "bad"}})
	require.ErrorContains(t, err, "semaphore release")
}

// TestNilProtoInputs guards nil request fields so adapter code can safely
// decode missing nested messages.
func TestNilProtoInputs(t *testing.T) {
	queueItem, err := QueueItemFromProto(nil)
	require.NoError(t, err)
	require.Equal(t, QueueItem{}, queueItem)

	item, err := ItemFromProto(nil)
	require.NoError(t, err)
	require.Equal(t, Item{}, item)

	identifier, err := IdentifierFromProto(nil)
	require.NoError(t, err)
	require.Equal(t, state.Identifier{}, identifier)

	runInfo, err := RunInfoFromProto(nil)
	require.NoError(t, err)
	require.Nil(t, runInfo)
}

// TestMalformedItemProtoInputs guards lossy or invalid item payloads that can
// arrive through the queue-proxy boundary.
func TestMalformedItemProtoInputs(t *testing.T) {
	_, err := ItemToProto(Item{Payload: make(chan struct{})})
	require.ErrorContains(t, err, "marshal item payload")

	_, err = ItemToProto(Item{Metadata: map[string]any{"bad": make(chan struct{})}})
	require.ErrorContains(t, err, "marshal item metadata")

	_, err = ItemFromProto(&pb.Item{
		WorkspaceId: uuid.NewString(),
		Identifier:  validIdentifierProto(),
		Metadata:    []byte(`{`),
	})
	require.ErrorContains(t, err, "unmarshal item metadata")

	_, err = ItemFromProto(&pb.Item{
		WorkspaceId:  uuid.NewString(),
		Identifier:   validIdentifierProto(),
		ParallelMode: "unknown",
	})
	require.ErrorContains(t, err, "parallel mode")

	_, err = ItemFromProto(&pb.Item{
		WorkspaceId: uuid.NewString(),
		Identifier:  validIdentifierProto(),
		RunInfo: &pb.RunInfo{
			CapacityLease: &pb.CapacityLease{LeaseId: "bad"},
		},
	})
	require.ErrorContains(t, err, "capacity lease lease_id")
}

// TestJSONPayloadAndMetadataConversion guards raw JSON payload transport and
// metadata decoding separately from the full queue item fixture.
func TestJSONPayloadAndMetadataConversion(t *testing.T) {
	payload := json.RawMessage(`{"n":1}`)
	item, err := ItemFromProto(&pb.Item{
		WorkspaceId: uuid.NewString(),
		Kind:        "custom-cloud-kind",
		Identifier:  validIdentifierProto(),
		Payload:     payload,
		Metadata:    []byte(`{"flag":true,"count":2}`),
	})
	require.NoError(t, err)
	require.Equal(t, payload, item.Payload)
	require.Equal(t, map[string]any{"flag": true, "count": float64(2)}, item.Metadata)

	msg, err := ItemToProto(Item{
		WorkspaceID: uuid.New(),
		Identifier:  validIdentifier(),
		Payload:     payload,
		Metadata:    map[string]any{"flag": true},
	})
	require.NoError(t, err)
	require.Equal(t, []byte(payload), msg.GetPayload())
	require.JSONEq(t, `{"flag":true}`, string(msg.GetMetadata()))
}

// TestDurationPointerConversion guards optional duration semantics used by
// queue item idempotency and enqueue options.
func TestDurationPointerConversion(t *testing.T) {
	require.Nil(t, durationPtrToProto(nil))
	require.Nil(t, durationPtrFromProto(nil))
	require.Equal(t, time.Duration(0), durationFromProto(nil))

	duration := 3 * time.Second
	require.Equal(t, duration, durationPtrToProto(&duration).AsDuration())
	require.Equal(t, duration, *durationPtrFromProto(durationpb.New(duration)))
}

// TestProtoConversionFieldCoverage guards added or removed Go struct fields.
// Adding an exported field to a converted type should force an explicit mapping
// or an explicit ignored-field decision here.
func TestProtoConversionFieldCoverage(t *testing.T) {
	assertCoveredFields(t, reflect.TypeOf(QueueItem{}), fieldCoverage{
		covered: []string{
			"ID",
			"EarliestPeekTime",
			"AtMS",
			"WallTimeMS",
			"FunctionID",
			"WorkspaceID",
			"LeaseID",
			"GenerationID",
			"Data",
			"QueueName",
			"IdempotencyPeriod",
			"RefilledFrom",
			"RefilledAt",
			"EnqueuedAt",
			"ScavengeCount",
			"CapacityLease",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(Item{}), fieldCoverage{
		covered: []string{
			"JobID",
			"GroupID",
			"WorkspaceID",
			"Kind",
			"Identifier",
			"Attempt",
			"MaxAttempts",
			"Payload",
			"Metadata",
			"QueueName",
			"RunInfo",
			"Throttle",
			"Singleton",
			"CustomConcurrencyKeys",
			"PriorityFactor",
			"ParallelMode",
			"ParallelCoalesceKey",
			"Semaphores",
		},
		ignored: map[string]string{
			"EnqueuedAt": "derived from QueueItem.EnqueuedAt when decoding a QueueItem",
			"At":         "derived from QueueItem.AtMS when decoding a QueueItem",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(RunInfo{}), fieldCoverage{
		covered: []string{
			"Latency",
			"SojournDelay",
			"Priority",
			"QueueShardName",
			"ContinueCount",
			"RefilledFromBacklog",
			"CapacityLease",
			"ScavengeCount",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(CapacityLease{}), fieldCoverage{
		covered: []string{
			"LeaseID",
			"IssuedAtMS",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(Throttle{}), fieldCoverage{
		covered: []string{
			"Key",
			"Limit",
			"Burst",
			"Period",
			"KeyExpressionHash",
			"UnhashedThrottleKey",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(Singleton{}), fieldCoverage{
		covered: []string{
			"Key",
			"Mode",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(state.Identifier{}), fieldCoverage{
		covered: []string{
			"RunID",
			"WorkflowID",
			"WorkflowVersion",
			"EventID",
			"BatchID",
			"EventIDs",
			"Key",
			"AccountID",
			"WorkspaceID",
			"AppID",
			"OriginalRunID",
			"ReplayID",
			"PriorityFactor",
			"CustomConcurrencyKeys",
			"Semaphores",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(state.CustomConcurrency{}), fieldCoverage{
		covered: []string{
			"Key",
			"Hash",
			"Limit",
			"UnhashedEvaluatedKeyValue",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(constraintapi.Semaphore{}), fieldCoverage{
		covered: []string{
			"ID",
			"EvaluatedKeyHash",
			"Weight",
			"Release",
		},
	})

	assertCoveredFields(t, reflect.TypeOf(EnqueueOpts{}), fieldCoverage{
		covered: []string{
			"PassthroughJobId",
			"ForceQueueShardName",
			"NormalizeFromBacklogID",
			"IdempotencyPeriod",
		},
	})
}

func assertItemEqual(t *testing.T, expected, actual Item) {
	t.Helper()

	require.Equal(t, expected.JobID, actual.JobID)
	require.Equal(t, expected.GroupID, actual.GroupID)
	require.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	require.Equal(t, expected.Kind, actual.Kind)
	require.Equal(t, expected.Attempt, actual.Attempt)
	require.Equal(t, expected.MaxAttempts, actual.MaxAttempts)
	require.JSONEq(t, string(expected.Payload.(json.RawMessage)), string(actual.Payload.(json.RawMessage)))
	require.Equal(t, expected.Metadata, actual.Metadata)
	require.Equal(t, expected.QueueName, actual.QueueName)
	require.Equal(t, expected.Throttle, actual.Throttle)
	require.Equal(t, expected.Singleton, actual.Singleton)
	require.Equal(t, expected.CustomConcurrencyKeys, actual.CustomConcurrencyKeys)
	require.Equal(t, expected.PriorityFactor, actual.PriorityFactor)
	require.Equal(t, expected.ParallelMode, actual.ParallelMode)
	require.Equal(t, expected.ParallelCoalesceKey, actual.ParallelCoalesceKey)
	require.Equal(t, expected.Semaphores, actual.Semaphores)
	require.Equal(t, expected.Identifier, actual.Identifier)

	require.NotNil(t, actual.RunInfo)
	require.Equal(t, expected.RunInfo.Latency, actual.RunInfo.Latency)
	require.Equal(t, expected.RunInfo.SojournDelay, actual.RunInfo.SojournDelay)
	require.Equal(t, expected.RunInfo.Priority, actual.RunInfo.Priority)
	require.Equal(t, expected.RunInfo.QueueShardName, actual.RunInfo.QueueShardName)
	require.Equal(t, expected.RunInfo.ContinueCount, actual.RunInfo.ContinueCount)
	require.Equal(t, expected.RunInfo.RefilledFromBacklog, actual.RunInfo.RefilledFromBacklog)
	require.Equal(t, expected.RunInfo.CapacityLease.LeaseID, actual.RunInfo.CapacityLease.LeaseID)
	require.Equal(t, expected.RunInfo.CapacityLease.IssuedAtMS, actual.RunInfo.CapacityLease.IssuedAtMS)
	require.Equal(t, expected.RunInfo.ScavengeCount, actual.RunInfo.ScavengeCount)
}

func protoStringPtr(value string) *string {
	return &value
}

func validIdentifierProto() *pb.Identifier {
	return &pb.Identifier{
		RunId:           ulid.Make().String(),
		WorkflowId:      uuid.NewString(),
		WorkflowVersion: 1,
		EventId:         ulid.Make().String(),
		EventIds:        []string{ulid.Make().String()},
		AccountId:       uuid.NewString(),
		WorkspaceId:     uuid.NewString(),
		AppId:           uuid.NewString(),
	}
}

func validIdentifier() state.Identifier {
	return state.Identifier{
		RunID:       ulid.Make(),
		WorkflowID:  uuid.New(),
		EventID:     ulid.Make(),
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
	}
}

type fieldCoverage struct {
	covered []string
	ignored map[string]string
}

func assertCoveredFields(t *testing.T, typ reflect.Type, coverage fieldCoverage) {
	t.Helper()

	covered := map[string]struct{}{}
	for _, name := range coverage.covered {
		if _, ok := typ.FieldByName(name); !ok {
			t.Fatalf("%s.%s is listed in proto conversion coverage but no longer exists", typ.Name(), name)
		}
		covered[name] = struct{}{}
	}

	for name, reason := range coverage.ignored {
		if _, ok := typ.FieldByName(name); !ok {
			t.Fatalf("%s.%s is listed in ignored proto conversion fields but no longer exists", typ.Name(), name)
		}
		require.NotEmptyf(t, reason, "%s.%s ignored field must include a reason", typ.Name(), name)
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		if _, ok := covered[field.Name]; ok {
			continue
		}
		if _, ok := coverage.ignored[field.Name]; ok {
			continue
		}
		t.Fatalf("%s.%s is not listed in proto conversion coverage", typ.Name(), field.Name)
	}
}
