package queue

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	pb "github.com/inngest/inngest/proto/gen/queue/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/durationpb"
)

func QueueItemToProto(item QueueItem) (*pb.QueueItem, error) {
	data, err := ItemToProto(item.Data)
	if err != nil {
		return nil, err
	}

	return &pb.QueueItem{
		Id:                item.ID,
		EarliestPeekTime:  item.EarliestPeekTime,
		AtMs:              item.AtMS,
		WallTimeMs:        item.WallTimeMS,
		FunctionId:        item.FunctionID.String(),
		WorkspaceId:       item.WorkspaceID.String(),
		LeaseId:           ulidPtrToString(item.LeaseID),
		GenerationId:      int64(item.GenerationID),
		Data:              data,
		QueueName:         item.QueueName,
		IdempotencyPeriod: durationPtrToProto(item.IdempotencyPeriod),
		RefilledFrom:      item.RefilledFrom,
		RefilledAt:        item.RefilledAt,
		EnqueuedAt:        item.EnqueuedAt,
		ScavengeCount:     int64(item.ScavengeCount),
		CapacityLease:     CapacityLeaseToProto(item.CapacityLease),
	}, nil
}

func QueueItemFromProto(msg *pb.QueueItem) (QueueItem, error) {
	if msg == nil {
		return QueueItem{}, nil
	}

	fnID, err := parseUUID(msg.GetFunctionId(), "queue item function_id")
	if err != nil {
		return QueueItem{}, err
	}
	workspaceID, err := parseUUID(msg.GetWorkspaceId(), "queue item workspace_id")
	if err != nil {
		return QueueItem{}, err
	}
	leaseID, err := ulidPtrFromString(msg.LeaseId, "queue item lease_id")
	if err != nil {
		return QueueItem{}, err
	}
	data, err := ItemFromProto(msg.GetData())
	if err != nil {
		return QueueItem{}, err
	}
	capacityLease, err := CapacityLeaseFromProto(msg.GetCapacityLease())
	if err != nil {
		return QueueItem{}, err
	}

	item := QueueItem{
		ID:                msg.GetId(),
		EarliestPeekTime:  msg.GetEarliestPeekTime(),
		AtMS:              msg.GetAtMs(),
		WallTimeMS:        msg.GetWallTimeMs(),
		FunctionID:        fnID,
		WorkspaceID:       workspaceID,
		LeaseID:           leaseID,
		GenerationID:      int(msg.GetGenerationId()),
		Data:              data,
		QueueName:         msg.QueueName,
		IdempotencyPeriod: durationPtrFromProto(msg.GetIdempotencyPeriod()),
		RefilledFrom:      msg.GetRefilledFrom(),
		RefilledAt:        msg.GetRefilledAt(),
		EnqueuedAt:        msg.GetEnqueuedAt(),
		ScavengeCount:     int(msg.GetScavengeCount()),
		CapacityLease:     capacityLease,
	}
	item.Data.EnqueuedAt = time.UnixMilli(item.EnqueuedAt)
	item.Data.At = time.UnixMilli(item.AtMS)

	return item, nil
}

func ItemToProto(item Item) (*pb.Item, error) {
	identifier, err := IdentifierToProto(item.Identifier)
	if err != nil {
		return nil, err
	}
	payload, err := jsonBytes(item.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal item payload: %w", err)
	}
	metadata, err := jsonBytes(item.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal item metadata: %w", err)
	}

	return &pb.Item{
		JobId:       item.JobID,
		GroupId:     item.GroupID,
		WorkspaceId: item.WorkspaceID.String(),
		Kind:        item.Kind,
		Identifier:  identifier,
		Attempt:     int64(item.Attempt),
		MaxAttempts: intPtrToInt64Ptr(item.MaxAttempts),
		Payload:     payload,
		Metadata:    metadata,
		QueueName:   item.QueueName,
		RunInfo:     RunInfoToProto(item.RunInfo),
		Throttle:    ThrottleToProto(item.Throttle),
		Singleton:   SingletonToProto(item.Singleton),
		CustomConcurrencyKeys: CustomConcurrencySliceToProto(
			item.CustomConcurrencyKeys,
		),
		PriorityFactor:      item.PriorityFactor,
		ParallelMode:        item.ParallelMode.String(),
		ParallelCoalesceKey: item.ParallelCoalesceKey,
		Semaphores:          SemaphoreSliceToProto(item.Semaphores),
	}, nil
}

func ItemFromProto(msg *pb.Item) (Item, error) {
	if msg == nil {
		return Item{}, nil
	}

	workspaceID, err := parseUUID(msg.GetWorkspaceId(), "item workspace_id")
	if err != nil {
		return Item{}, err
	}
	identifier, err := IdentifierFromProto(msg.GetIdentifier())
	if err != nil {
		return Item{}, err
	}
	metadata, err := metadataFromProto(msg.GetMetadata())
	if err != nil {
		return Item{}, err
	}
	runInfo, err := RunInfoFromProto(msg.GetRunInfo())
	if err != nil {
		return Item{}, err
	}
	singleton, err := SingletonFromProto(msg.GetSingleton())
	if err != nil {
		return Item{}, err
	}
	semaphores, err := SemaphoreSliceFromProto(msg.GetSemaphores())
	if err != nil {
		return Item{}, err
	}
	parallelMode, err := parallelModeFromProto(msg.GetParallelMode())
	if err != nil {
		return Item{}, err
	}

	return Item{
		JobID:                 msg.JobId,
		GroupID:               msg.GetGroupId(),
		WorkspaceID:           workspaceID,
		Kind:                  msg.GetKind(),
		Identifier:            identifier,
		Attempt:               int(msg.GetAttempt()),
		MaxAttempts:           int64PtrToIntPtr(msg.MaxAttempts),
		Payload:               payloadFromProto(msg.GetPayload()),
		Metadata:              metadata,
		QueueName:             msg.QueueName,
		RunInfo:               runInfo,
		Throttle:              ThrottleFromProto(msg.GetThrottle()),
		Singleton:             singleton,
		CustomConcurrencyKeys: CustomConcurrencySliceFromProto(msg.GetCustomConcurrencyKeys()),
		PriorityFactor:        msg.PriorityFactor,
		ParallelMode:          parallelMode,
		ParallelCoalesceKey:   msg.ParallelCoalesceKey,
		Semaphores:            semaphores,
	}, nil
}

func IdentifierToProto(identifier state.Identifier) (*pb.Identifier, error) {
	eventIDs := make([]string, len(identifier.EventIDs))
	for i, id := range identifier.EventIDs {
		eventIDs[i] = id.String()
	}

	return &pb.Identifier{
		RunId:                 identifier.RunID.String(),
		WorkflowId:            identifier.WorkflowID.String(),
		WorkflowVersion:       int64(identifier.WorkflowVersion),
		EventId:               identifier.EventID.String(),
		BatchId:               ulidPtrToString(identifier.BatchID),
		EventIds:              eventIDs,
		Key:                   identifier.Key,
		AccountId:             identifier.AccountID.String(),
		WorkspaceId:           identifier.WorkspaceID.String(),
		AppId:                 identifier.AppID.String(),
		OriginalRunId:         ulidPtrToString(identifier.OriginalRunID),
		ReplayId:              uuidPtrToString(identifier.ReplayID),
		PriorityFactor:        identifier.PriorityFactor,
		CustomConcurrencyKeys: CustomConcurrencySliceToProto(identifier.CustomConcurrencyKeys),
		Semaphores:            SemaphoreSliceToProto(identifier.Semaphores),
	}, nil
}

func IdentifierFromProto(msg *pb.Identifier) (state.Identifier, error) {
	if msg == nil {
		return state.Identifier{}, nil
	}

	runID, err := parseULID(msg.GetRunId(), "identifier run_id")
	if err != nil {
		return state.Identifier{}, err
	}
	workflowID, err := parseUUID(msg.GetWorkflowId(), "identifier workflow_id")
	if err != nil {
		return state.Identifier{}, err
	}
	eventID, err := parseULID(msg.GetEventId(), "identifier event_id")
	if err != nil {
		return state.Identifier{}, err
	}
	eventIDs, err := ulidSliceFromStrings(msg.GetEventIds(), "identifier event_ids")
	if err != nil {
		return state.Identifier{}, err
	}
	accountID, err := parseUUID(msg.GetAccountId(), "identifier account_id")
	if err != nil {
		return state.Identifier{}, err
	}
	workspaceID, err := parseUUID(msg.GetWorkspaceId(), "identifier workspace_id")
	if err != nil {
		return state.Identifier{}, err
	}
	appID, err := parseUUID(msg.GetAppId(), "identifier app_id")
	if err != nil {
		return state.Identifier{}, err
	}
	batchID, err := ulidPtrFromString(msg.BatchId, "identifier batch_id")
	if err != nil {
		return state.Identifier{}, err
	}
	originalRunID, err := ulidPtrFromString(msg.OriginalRunId, "identifier original_run_id")
	if err != nil {
		return state.Identifier{}, err
	}
	replayID, err := uuidPtrFromString(msg.ReplayId, "identifier replay_id")
	if err != nil {
		return state.Identifier{}, err
	}
	semaphores, err := SemaphoreSliceFromProto(msg.GetSemaphores())
	if err != nil {
		return state.Identifier{}, err
	}

	return state.Identifier{
		RunID:                 runID,
		WorkflowID:            workflowID,
		WorkflowVersion:       int(msg.GetWorkflowVersion()),
		EventID:               eventID,
		BatchID:               batchID,
		EventIDs:              eventIDs,
		Key:                   msg.GetKey(),
		AccountID:             accountID,
		WorkspaceID:           workspaceID,
		AppID:                 appID,
		OriginalRunID:         originalRunID,
		ReplayID:              replayID,
		PriorityFactor:        msg.PriorityFactor,
		CustomConcurrencyKeys: CustomConcurrencySliceFromProto(msg.GetCustomConcurrencyKeys()),
		Semaphores:            semaphores,
	}, nil
}

func RunInfoToProto(info *RunInfo) *pb.RunInfo {
	if info == nil {
		return nil
	}
	return &pb.RunInfo{
		Latency:             durationpb.New(info.Latency),
		SojournDelay:        durationpb.New(info.SojournDelay),
		Priority:            uint64(info.Priority),
		QueueShardName:      info.QueueShardName,
		ContinueCount:       uint64(info.ContinueCount),
		RefilledFromBacklog: info.RefilledFromBacklog,
		CapacityLease:       CapacityLeaseToProto(info.CapacityLease),
		ScavengeCount:       int64(info.ScavengeCount),
	}
}

func RunInfoFromProto(msg *pb.RunInfo) (*RunInfo, error) {
	if msg == nil {
		return nil, nil
	}
	capacityLease, err := CapacityLeaseFromProto(msg.GetCapacityLease())
	if err != nil {
		return nil, err
	}
	return &RunInfo{
		Latency:             durationFromProto(msg.GetLatency()),
		SojournDelay:        durationFromProto(msg.GetSojournDelay()),
		Priority:            uint(msg.GetPriority()),
		QueueShardName:      msg.GetQueueShardName(),
		ContinueCount:       uint(msg.GetContinueCount()),
		RefilledFromBacklog: msg.GetRefilledFromBacklog(),
		CapacityLease:       capacityLease,
		ScavengeCount:       int(msg.GetScavengeCount()),
	}, nil
}

func CapacityLeaseToProto(lease *CapacityLease) *pb.CapacityLease {
	if lease == nil {
		return nil
	}
	return &pb.CapacityLease{
		LeaseId:    lease.LeaseID.String(),
		IssuedAtMs: lease.IssuedAtMS,
	}
}

func CapacityLeaseFromProto(msg *pb.CapacityLease) (*CapacityLease, error) {
	if msg == nil {
		return nil, nil
	}
	leaseID, err := parseULID(msg.GetLeaseId(), "capacity lease lease_id")
	if err != nil {
		return nil, err
	}
	return &CapacityLease{
		LeaseID:    leaseID,
		IssuedAtMS: msg.GetIssuedAtMs(),
	}, nil
}

func ThrottleToProto(throttle *Throttle) *pb.Throttle {
	if throttle == nil {
		return nil
	}
	return &pb.Throttle{
		Key:               throttle.Key,
		Limit:             int64(throttle.Limit),
		Burst:             int64(throttle.Burst),
		Period:            int64(throttle.Period),
		KeyExpressionHash: throttle.KeyExpressionHash,
	}
}

func ThrottleFromProto(msg *pb.Throttle) *Throttle {
	if msg == nil {
		return nil
	}
	return &Throttle{
		Key:               msg.GetKey(),
		Limit:             int(msg.GetLimit()),
		Burst:             int(msg.GetBurst()),
		Period:            int(msg.GetPeriod()),
		KeyExpressionHash: msg.GetKeyExpressionHash(),
	}
}

func SingletonToProto(singleton *Singleton) *pb.Singleton {
	if singleton == nil {
		return nil
	}
	return &pb.Singleton{
		Key:  singleton.Key,
		Mode: singleton.Mode.String(),
	}
}

func SingletonFromProto(msg *pb.Singleton) (*Singleton, error) {
	if msg == nil {
		return nil, nil
	}
	mode, err := singletonModeFromProto(msg.GetMode())
	if err != nil {
		return nil, fmt.Errorf("singleton mode: %w", err)
	}
	return &Singleton{
		Key:  msg.GetKey(),
		Mode: mode,
	}, nil
}

func CustomConcurrencySliceToProto(keys []state.CustomConcurrency) []*pb.CustomConcurrency {
	if len(keys) == 0 {
		return nil
	}
	result := make([]*pb.CustomConcurrency, len(keys))
	for i, key := range keys {
		result[i] = &pb.CustomConcurrency{
			Key:                       key.Key,
			Hash:                      key.Hash,
			Limit:                     int64(key.Limit),
			UnhashedEvaluatedKeyValue: key.UnhashedEvaluatedKeyValue,
		}
	}
	return result
}

func CustomConcurrencySliceFromProto(keys []*pb.CustomConcurrency) []state.CustomConcurrency {
	if len(keys) == 0 {
		return nil
	}
	result := make([]state.CustomConcurrency, len(keys))
	for i, key := range keys {
		result[i] = state.CustomConcurrency{
			Key:                       key.GetKey(),
			Hash:                      key.GetHash(),
			Limit:                     int(key.GetLimit()),
			UnhashedEvaluatedKeyValue: key.GetUnhashedEvaluatedKeyValue(),
		}
	}
	return result
}

func SemaphoreSliceToProto(semaphores []constraintapi.Semaphore) []*pb.Semaphore {
	if len(semaphores) == 0 {
		return nil
	}
	result := make([]*pb.Semaphore, len(semaphores))
	for i, semaphore := range semaphores {
		result[i] = &pb.Semaphore{
			Id:               semaphore.ID,
			EvaluatedKeyHash: semaphore.EvaluatedKeyHash,
			Weight:           semaphore.Weight,
			Release:          strconv.Itoa(int(semaphore.Release)),
		}
	}
	return result
}

func SemaphoreSliceFromProto(semaphores []*pb.Semaphore) ([]constraintapi.Semaphore, error) {
	if len(semaphores) == 0 {
		return nil, nil
	}
	result := make([]constraintapi.Semaphore, len(semaphores))
	for i, semaphore := range semaphores {
		release, err := semaphoreReleaseFromProto(semaphore.GetRelease())
		if err != nil {
			return nil, err
		}
		result[i] = constraintapi.Semaphore{
			ID:               semaphore.GetId(),
			EvaluatedKeyHash: semaphore.GetEvaluatedKeyHash(),
			Weight:           semaphore.GetWeight(),
			Release:          release,
		}
	}
	return result, nil
}

func EnqueueOptionsToProto(opts EnqueueOpts) *pb.EnqueueOptions {
	return &pb.EnqueueOptions{
		PassthroughJobId:       opts.PassthroughJobId,
		ForceQueueShardName:    opts.ForceQueueShardName,
		NormalizeFromBacklogId: opts.NormalizeFromBacklogID,
		IdempotencyPeriod:      durationPtrToProto(opts.IdempotencyPeriod),
	}
}

func EnqueueOptionsFromProto(msg *pb.EnqueueOptions) EnqueueOpts {
	if msg == nil {
		return EnqueueOpts{}
	}
	return EnqueueOpts{
		PassthroughJobId:       msg.GetPassthroughJobId(),
		ForceQueueShardName:    msg.GetForceQueueShardName(),
		NormalizeFromBacklogID: msg.GetNormalizeFromBacklogId(),
		IdempotencyPeriod:      durationPtrFromProto(msg.GetIdempotencyPeriod()),
	}
}

func jsonBytes(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	if raw, ok := v.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(v)
}

func payloadFromProto(payload []byte) any {
	if len(payload) == 0 {
		return nil
	}
	return json.RawMessage(payload)
}

func metadataFromProto(metadata []byte) (map[string]any, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	result := map[string]any{}
	if err := json.Unmarshal(metadata, &result); err != nil {
		return nil, fmt.Errorf("unmarshal item metadata: %w", err)
	}
	return result, nil
}

func parseUUID(value, field string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", field, err)
	}
	return id, nil
}

func parseULID(value, field string) (ulid.ULID, error) {
	if value == "" {
		return ulid.ULID{}, nil
	}
	id, err := ulid.Parse(value)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("%s: %w", field, err)
	}
	return id, nil
}

func uuidPtrToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}

func uuidPtrFromString(value *string, field string) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	id, err := parseUUID(*value, field)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func ulidPtrToString(id *ulid.ULID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}

func ulidPtrFromString(value *string, field string) (*ulid.ULID, error) {
	if value == nil {
		return nil, nil
	}
	id, err := parseULID(*value, field)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func ulidSliceFromStrings(values []string, field string) ([]ulid.ULID, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]ulid.ULID, len(values))
	for i, value := range values {
		id, err := parseULID(value, field)
		if err != nil {
			return nil, err
		}
		result[i] = id
	}
	return result, nil
}

func durationPtrToProto(duration *time.Duration) *durationpb.Duration {
	if duration == nil {
		return nil
	}
	return durationpb.New(*duration)
}

func durationPtrFromProto(duration *durationpb.Duration) *time.Duration {
	if duration == nil {
		return nil
	}
	value := durationFromProto(duration)
	return &value
}

func durationFromProto(duration *durationpb.Duration) time.Duration {
	if duration == nil {
		return 0
	}
	return duration.AsDuration()
}

func intPtrToInt64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	result := int64(*value)
	return &result
}

func int64PtrToIntPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}

func parallelModeFromProto(value string) (enums.ParallelMode, error) {
	if value == "" {
		return enums.ParallelModeNone, nil
	}
	mode, err := enums.ParallelModeString(value)
	if err != nil {
		return enums.ParallelModeNone, fmt.Errorf("parallel mode: %w", err)
	}
	return mode, nil
}

func singletonModeFromProto(value string) (enums.SingletonMode, error) {
	if value == "" {
		return enums.SingletonModeSkip, nil
	}
	return enums.SingletonModeString(value)
}

func semaphoreReleaseFromProto(value string) (constraintapi.SemaphoreReleaseMode, error) {
	if value == "" {
		return constraintapi.SemaphoreReleaseAuto, nil
	}
	release, err := strconv.Atoi(value)
	if err != nil {
		return constraintapi.SemaphoreReleaseAuto, fmt.Errorf("semaphore release: %w", err)
	}
	return constraintapi.SemaphoreReleaseMode(release), nil
}
