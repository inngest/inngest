package state

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	pb "github.com/inngest/inngest/proto/gen/state/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MetadataToProto converts internal Metadata to the proto Metadata message.
func MetadataToProto(md Metadata) (*pb.Metadata, error) {
	pbConfig, err := ConfigToProto(md.Config)
	if err != nil {
		return nil, fmt.Errorf("config to proto: %w", err)
	}

	return &pb.Metadata{
		Id:      IDToProto(md.ID),
		Config:  pbConfig,
		Metrics: RunMetricsToProto(md.Metrics),
		Stack:   md.Stack,
	}, nil
}

// MetadataFromProto converts a proto Metadata message to the internal Metadata type.
func MetadataFromProto(pbMd *pb.Metadata) (Metadata, error) {
	if pbMd == nil {
		return Metadata{}, nil
	}

	id, err := IDFromProto(pbMd.Id)
	if err != nil {
		return Metadata{}, fmt.Errorf("id from proto: %w", err)
	}

	cfg, err := ConfigFromProto(pbMd.Config)
	if err != nil {
		return Metadata{}, fmt.Errorf("config from proto: %w", err)
	}

	return Metadata{
		ID:      id,
		Config:  cfg,
		Metrics: RunMetricsFromProto(pbMd.Metrics),
		Stack:   pbMd.Stack,
	}, nil
}

// ConfigToProto converts internal Config to the proto Config message.
func ConfigToProto(c Config) (*pb.Config, error) {
	// EventIDs: ulid.ULID -> string
	eventIDs := make([]string, len(c.EventIDs))
	for i, eid := range c.EventIDs {
		eventIDs[i] = eid.String()
	}

	// BatchID: *ulid.ULID -> string
	var batchID string
	if c.BatchID != nil {
		batchID = c.BatchID.String()
	}

	// StartedAt: time.Time -> *timestamppb.Timestamp
	var startedAt *timestamppb.Timestamp
	if !c.StartedAt.IsZero() {
		startedAt = timestamppb.New(c.StartedAt)
	}

	// ReplayID: *uuid.UUID -> *string
	var replayID *string
	if c.ReplayID != nil {
		s := c.ReplayID.String()
		replayID = &s
	}

	// OriginalRunID: *ulid.ULID -> *string
	var originalRunID *string
	if c.OriginalRunID != nil {
		s := c.OriginalRunID.String()
		originalRunID = &s
	}

	// PriorityFactor: *int64 -> *int64
	var priorityFactor *int64
	if c.PriorityFactor != nil {
		v := *c.PriorityFactor
		priorityFactor = &v
	}

	// ConcurrencyKeys: []CustomConcurrency -> []*pb.ConcurrencyKey
	var concurrencyKeys []*pb.ConcurrencyKey
	for _, ck := range c.CustomConcurrencyKeys {
		concurrencyKeys = append(concurrencyKeys, &pb.ConcurrencyKey{
			Key:   ck.Key,
			Hash:  ck.Hash,
			Limit: int64(ck.Limit),
		})
	}

	// Context: map[string]any -> *structpb.Struct
	var pbCtx *structpb.Struct
	if len(c.Context) > 0 {
		var err error
		pbCtx, err = structpb.NewStruct(c.Context)
		if err != nil {
			return nil, fmt.Errorf("context to struct: %w", err)
		}
	}

	// Semaphores: []constraintapi.Semaphore -> JSON string
	var semaphoresJSON string
	if len(c.Semaphores) > 0 {
		byt, err := json.Marshal(c.Semaphores)
		if err != nil {
			return nil, fmt.Errorf("marshal semaphores: %w", err)
		}
		semaphoresJSON = string(byt)
	}

	return &pb.Config{
		FunctionVersion: int64(c.FunctionVersion),
		SpanId:          c.SpanID,
		BatchId:         batchID,
		StartedAt:       startedAt,
		EventIds:        eventIDs,
		RequestVersion:  int64(c.RequestVersion),
		Idempotency:     c.Idempotency,
		ReplayId:        replayID,
		OriginalRunId:   originalRunID,
		PriorityFactor:  priorityFactor,
		ConcurrencyKeys: concurrencyKeys,
		ForceStepPlan:   c.ForceStepPlan,
		Context:         pbCtx,
		HasAi:           c.HasAI,
		SemaphoresJson:  semaphoresJSON,
	}, nil
}

// ConfigFromProto converts a proto Config message to the internal Config type.
func ConfigFromProto(pbCfg *pb.Config) (Config, error) {
	if pbCfg == nil {
		return *InitConfig(&Config{}), nil
	}

	// EventIDs: string -> ulid.ULID
	eventIDs := make([]ulid.ULID, len(pbCfg.EventIds))
	for i, s := range pbCfg.EventIds {
		id, err := ulid.Parse(s)
		if err != nil {
			return Config{}, fmt.Errorf("parse event ID %q: %w", s, err)
		}
		eventIDs[i] = id
	}

	// BatchID: string -> *ulid.ULID
	var batchID *ulid.ULID
	if pbCfg.BatchId != "" {
		id, err := ulid.Parse(pbCfg.BatchId)
		if err != nil {
			return Config{}, fmt.Errorf("parse batch ID %q: %w", pbCfg.BatchId, err)
		}
		batchID = &id
	}

	// StartedAt: *timestamppb.Timestamp -> time.Time
	var startedAt = pbCfg.StartedAt.AsTime()

	// ReplayID: *string -> *uuid.UUID
	var replayID *uuid.UUID
	if pbCfg.ReplayId != nil {
		id, err := uuid.Parse(*pbCfg.ReplayId)
		if err != nil {
			return Config{}, fmt.Errorf("parse replay ID %q: %w", *pbCfg.ReplayId, err)
		}
		replayID = &id
	}

	// OriginalRunID: *string -> *ulid.ULID
	var originalRunID *ulid.ULID
	if pbCfg.OriginalRunId != nil {
		id, err := ulid.Parse(*pbCfg.OriginalRunId)
		if err != nil {
			return Config{}, fmt.Errorf("parse original run ID %q: %w", *pbCfg.OriginalRunId, err)
		}
		originalRunID = &id
	}

	// PriorityFactor: *int64 -> *int64
	var priorityFactor *int64
	if pbCfg.PriorityFactor != nil {
		v := *pbCfg.PriorityFactor
		priorityFactor = &v
	}

	// ConcurrencyKeys: []*pb.ConcurrencyKey -> []CustomConcurrency
	var concurrencyKeys []CustomConcurrency
	for _, ck := range pbCfg.ConcurrencyKeys {
		concurrencyKeys = append(concurrencyKeys, CustomConcurrency{
			Key:   ck.Key,
			Hash:  ck.Hash,
			Limit: int(ck.Limit),
		})
	}

	// Context: *structpb.Struct -> map[string]any
	var ctx map[string]any
	if pbCfg.Context != nil {
		ctx = pbCfg.Context.AsMap()
	}

	// Semaphores: JSON string -> []constraintapi.Semaphore
	var semaphores []constraintapi.Semaphore
	if pbCfg.SemaphoresJson != "" {
		if err := json.Unmarshal([]byte(pbCfg.SemaphoresJson), &semaphores); err != nil {
			return Config{}, fmt.Errorf("unmarshal semaphores_json: %w", err)
		}
	}

	return *InitConfig(&Config{
		FunctionVersion:       int(pbCfg.FunctionVersion),
		SpanID:                pbCfg.SpanId,
		BatchID:               batchID,
		StartedAt:             startedAt,
		EventIDs:              eventIDs,
		RequestVersion:        int(pbCfg.RequestVersion),
		Idempotency:           pbCfg.Idempotency,
		ReplayID:              replayID,
		OriginalRunID:         originalRunID,
		PriorityFactor:        priorityFactor,
		CustomConcurrencyKeys: concurrencyKeys,
		ForceStepPlan:         pbCfg.ForceStepPlan,
		Context:               ctx,
		HasAI:                 pbCfg.HasAi,
		Semaphores:            semaphores,
	}), nil
}

// IDToProto converts internal ID to the proto ID message.
func IDToProto(id ID) *pb.ID {
	return &pb.ID{
		RunId:      id.RunID.String(),
		FunctionId: id.FunctionID.String(),
		Tenant:     TenantToProto(id.Tenant),
	}
}

// IDFromProto converts a proto ID message to the internal ID type.
func IDFromProto(pbID *pb.ID) (ID, error) {
	if pbID == nil {
		return ID{}, nil
	}

	runID, err := ulid.Parse(pbID.RunId)
	if err != nil {
		return ID{}, fmt.Errorf("parse run ID %q: %w", pbID.RunId, err)
	}

	functionID, err := uuid.Parse(pbID.FunctionId)
	if err != nil {
		return ID{}, fmt.Errorf("parse function ID %q: %w", pbID.FunctionId, err)
	}

	tenant, err := TenantFromProto(pbID.Tenant)
	if err != nil {
		return ID{}, fmt.Errorf("tenant from proto: %w", err)
	}

	return ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant:     tenant,
	}, nil
}

// TenantToProto converts internal Tenant to the proto Tenant message.
func TenantToProto(t Tenant) *pb.Tenant {
	return &pb.Tenant{
		AppId:     t.AppID.String(),
		EnvId:     t.EnvID.String(),
		AccountId: t.AccountID.String(),
	}
}

// TenantFromProto converts a proto Tenant message to the internal Tenant type.
func TenantFromProto(pbT *pb.Tenant) (Tenant, error) {
	if pbT == nil {
		return Tenant{}, nil
	}

	appID, err := uuid.Parse(pbT.AppId)
	if err != nil {
		return Tenant{}, fmt.Errorf("parse app ID %q: %w", pbT.AppId, err)
	}

	envID, err := uuid.Parse(pbT.EnvId)
	if err != nil {
		return Tenant{}, fmt.Errorf("parse env ID %q: %w", pbT.EnvId, err)
	}

	accountID, err := uuid.Parse(pbT.AccountId)
	if err != nil {
		return Tenant{}, fmt.Errorf("parse account ID %q: %w", pbT.AccountId, err)
	}

	return Tenant{
		AppID:     appID,
		EnvID:     envID,
		AccountID: accountID,
	}, nil
}

// RunMetricsToProto converts internal RunMetrics to the proto RunMetrics message.
func RunMetricsToProto(rm RunMetrics) *pb.RunMetrics {
	return &pb.RunMetrics{
		StateSize: int64(rm.StateSize),
		EventSize: int64(rm.EventSize),
		StepCount: int64(rm.StepCount),
	}
}

// RunMetricsFromProto converts a proto RunMetrics message to the internal RunMetrics type.
func RunMetricsFromProto(pbRM *pb.RunMetrics) RunMetrics {
	if pbRM == nil {
		return RunMetrics{}
	}

	return RunMetrics{
		StateSize: int(pbRM.StateSize),
		EventSize: int(pbRM.EventSize),
		StepCount: int(pbRM.StepCount),
	}
}
