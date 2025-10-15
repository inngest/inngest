package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/inngest/inngest/pkg/enums"
	pb "github.com/inngest/inngest/proto/gen/constraintapi/v1"
)

func RateLimitScopeToProto(scope enums.RateLimitScope) pb.ConstraintApiRateLimitScope {
	switch scope {
	case enums.RateLimitScopeFn:
		return pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION
	case enums.RateLimitScopeEnv:
		return pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ENV
	case enums.RateLimitScopeAccount:
		return pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT
	default:
		return pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_UNSPECIFIED
	}
}

func RateLimitScopeFromProto(scope pb.ConstraintApiRateLimitScope) enums.RateLimitScope {
	switch scope {
	case pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION:
		return enums.RateLimitScopeFn
	case pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ENV:
		return enums.RateLimitScopeEnv
	case pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT:
		return enums.RateLimitScopeAccount
	default:
		return enums.RateLimitScopeFn
	}
}

func ConcurrencyScopeToProto(scope enums.ConcurrencyScope) pb.ConstraintApiConcurrencyScope {
	switch scope {
	case enums.ConcurrencyScopeFn:
		return pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION
	case enums.ConcurrencyScopeEnv:
		return pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ENV
	case enums.ConcurrencyScopeAccount:
		return pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ACCOUNT
	default:
		return pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_UNSPECIFIED
	}
}

func ConcurrencyScopeFromProto(scope pb.ConstraintApiConcurrencyScope) enums.ConcurrencyScope {
	switch scope {
	case pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION:
		return enums.ConcurrencyScopeFn
	case pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ENV:
		return enums.ConcurrencyScopeEnv
	case pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ACCOUNT:
		return enums.ConcurrencyScopeAccount
	default:
		return enums.ConcurrencyScopeFn
	}
}

func ThrottleScopeToProto(scope enums.ThrottleScope) pb.ConstraintApiThrottleScope {
	switch scope {
	case enums.ThrottleScopeFn:
		return pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_FUNCTION
	case enums.ThrottleScopeEnv:
		return pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV
	case enums.ThrottleScopeAccount:
		return pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ACCOUNT
	default:
		return pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_UNSPECIFIED
	}
}

func ThrottleScopeFromProto(scope pb.ConstraintApiThrottleScope) enums.ThrottleScope {
	switch scope {
	case pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_FUNCTION:
		return enums.ThrottleScopeFn
	case pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV:
		return enums.ThrottleScopeEnv
	case pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ACCOUNT:
		return enums.ThrottleScopeAccount
	default:
		return enums.ThrottleScopeFn
	}
}

func ConcurrencyModeToProto(mode enums.ConcurrencyMode) pb.ConstraintApiConcurrencyMode {
	switch mode {
	case enums.ConcurrencyModeStep:
		return pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP
	case enums.ConcurrencyModeRun:
		return pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_RUN
	default:
		return pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_UNSPECIFIED
	}
}

func ConcurrencyModeFromProto(mode pb.ConstraintApiConcurrencyMode) enums.ConcurrencyMode {
	switch mode {
	case pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP:
		return enums.ConcurrencyModeStep
	case pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_RUN:
		return enums.ConcurrencyModeRun
	default:
		return enums.ConcurrencyModeStep
	}
}

func ConstraintKindToProto(kind ConstraintKind) pb.ConstraintApiConstraintKind {
	switch kind {
	case CapacityKindRateLimit:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT
	case CapacityKindConcurrency:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY
	case CapacityKindThrottle:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE
	default:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED
	}
}

func ConstraintKindFromProto(kind pb.ConstraintApiConstraintKind) ConstraintKind {
	switch kind {
	case pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT:
		return CapacityKindRateLimit
	case pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY:
		return CapacityKindConcurrency
	case pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE:
		return CapacityKindThrottle
	default:
		return CapacityKindRateLimit
	}
}

func RunProcessingModeToProto(mode RunProcessingMode) pb.ConstraintApiRunProcessingMode {
	switch mode {
	case RunProcessingModeBackground:
		return pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND
	case RunProcessingModeSync:
		return pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_SYNC
	default:
		return pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_UNSPECIFIED
	}
}

func RunProcessingModeFromProto(mode pb.ConstraintApiRunProcessingMode) RunProcessingMode {
	switch mode {
	case pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND:
		return RunProcessingModeBackground
	case pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_SYNC:
		return RunProcessingModeSync
	default:
		return RunProcessingModeBackground
	}
}

func LeaseLocationToProto(location LeaseLocation) pb.ConstraintApiLeaseLocation {
	switch location {
	case LeaseLocationScheduleRun:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN
	case LeaseLocationPartitionLease:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_PARTITION_LEASE
	case LeaseLocationItemLease:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE
	default:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_UNSPECIFIED
	}
}

func LeaseLocationFromProto(location pb.ConstraintApiLeaseLocation) LeaseLocation {
	switch location {
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN:
		return LeaseLocationScheduleRun
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_PARTITION_LEASE:
		return LeaseLocationPartitionLease
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE:
		return LeaseLocationItemLease
	default:
		return LeaseLocationScheduleRun
	}
}

func LeaseServiceToProto(service LeaseService) pb.ConstraintApiLeaseService {
	switch service {
	case ServiceNewRuns:
		return pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS
	case ServiceExecutor:
		return pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR
	case ServiceAPI:
		return pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API
	default:
		return pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_UNSPECIFIED
	}
}

func LeaseServiceFromProto(service pb.ConstraintApiLeaseService) LeaseService {
	switch service {
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS:
		return ServiceNewRuns
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR:
		return ServiceExecutor
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API:
		return ServiceAPI
	default:
		return ServiceNewRuns
	}
}

func RateLimitConfigToProto(config RateLimitConfig) *pb.RateLimitConfig {
	return &pb.RateLimitConfig{
		Scope:             RateLimitScopeToProto(config.Scope),
		Limit:             int32(config.Limit),
		Period:            config.Period,
		KeyExpressionHash: config.KeyExpressionHash,
	}
}

func RateLimitConfigFromProto(pbConfig *pb.RateLimitConfig) RateLimitConfig {
	if pbConfig == nil {
		return RateLimitConfig{}
	}
	return RateLimitConfig{
		Scope:             RateLimitScopeFromProto(pbConfig.Scope),
		Limit:             int(pbConfig.Limit),
		Period:            pbConfig.Period,
		KeyExpressionHash: pbConfig.KeyExpressionHash,
	}
}

func CustomConcurrencyLimitToProto(limit CustomConcurrencyLimit) *pb.CustomConcurrencyLimit {
	return &pb.CustomConcurrencyLimit{
		Mode:              ConcurrencyModeToProto(limit.Mode),
		Scope:             ConcurrencyScopeToProto(limit.Scope),
		Limit:             int32(limit.Limit),
		KeyExpressionHash: limit.KeyExpressionHash,
	}
}

func CustomConcurrencyLimitFromProto(pbLimit *pb.CustomConcurrencyLimit) CustomConcurrencyLimit {
	if pbLimit == nil {
		return CustomConcurrencyLimit{}
	}
	return CustomConcurrencyLimit{
		Mode:              ConcurrencyModeFromProto(pbLimit.Mode),
		Scope:             ConcurrencyScopeFromProto(pbLimit.Scope),
		Limit:             int(pbLimit.Limit),
		KeyExpressionHash: pbLimit.KeyExpressionHash,
	}
}

func ConcurrencyConfigToProto(config ConcurrencyConfig) *pb.ConcurrencyConfig {
	customKeys := make([]*pb.CustomConcurrencyLimit, len(config.CustomConcurrencyKeys))
	for i, key := range config.CustomConcurrencyKeys {
		customKeys[i] = CustomConcurrencyLimitToProto(key)
	}

	return &pb.ConcurrencyConfig{
		AccountConcurrency:       int32(config.AccountConcurrency),
		FunctionConcurrency:      int32(config.FunctionConcurrency),
		AccountRunConcurrency:    int32(config.AccountRunConcurrency),
		FunctionRunConcurrency:   int32(config.FunctionRunConcurrency),
		CustomConcurrencyKeys:    customKeys,
	}
}

func ConcurrencyConfigFromProto(pbConfig *pb.ConcurrencyConfig) ConcurrencyConfig {
	if pbConfig == nil {
		return ConcurrencyConfig{}
	}

	customKeys := make([]CustomConcurrencyLimit, len(pbConfig.CustomConcurrencyKeys))
	for i, key := range pbConfig.CustomConcurrencyKeys {
		customKeys[i] = CustomConcurrencyLimitFromProto(key)
	}

	return ConcurrencyConfig{
		AccountConcurrency:       int(pbConfig.AccountConcurrency),
		FunctionConcurrency:      int(pbConfig.FunctionConcurrency),
		AccountRunConcurrency:    int(pbConfig.AccountRunConcurrency),
		FunctionRunConcurrency:   int(pbConfig.FunctionRunConcurrency),
		CustomConcurrencyKeys:    customKeys,
	}
}

func ThrottleConfigToProto(config ThrottleConfig) *pb.ThrottleConfig {
	return &pb.ThrottleConfig{
		Scope:                     ThrottleScopeToProto(config.Scope),
		ThrottleKeyExpressionHash: config.ThrottleKeyExpressionHash,
		Limit:                     int32(config.Limit),
		Burst:                     int32(config.Burst),
		Period:                    int32(config.Period),
	}
}

func ThrottleConfigFromProto(pbConfig *pb.ThrottleConfig) ThrottleConfig {
	if pbConfig == nil {
		return ThrottleConfig{}
	}
	return ThrottleConfig{
		Scope:                     ThrottleScopeFromProto(pbConfig.Scope),
		ThrottleKeyExpressionHash: pbConfig.ThrottleKeyExpressionHash,
		Limit:                     int(pbConfig.Limit),
		Burst:                     int(pbConfig.Burst),
		Period:                    int(pbConfig.Period),
	}
}

func ConstraintConfigToProto(config ConstraintConfig) *pb.ConstraintConfig {
	rateLimits := make([]*pb.RateLimitConfig, len(config.RateLimit))
	for i, rl := range config.RateLimit {
		rateLimits[i] = RateLimitConfigToProto(rl)
	}

	throttles := make([]*pb.ThrottleConfig, len(config.Throttle))
	for i, th := range config.Throttle {
		throttles[i] = ThrottleConfigToProto(th)
	}

	return &pb.ConstraintConfig{
		FunctionVersion: int32(config.FunctionVersion),
		RateLimit:       rateLimits,
		Concurrency:     ConcurrencyConfigToProto(config.Concurrency),
		Throttle:        throttles,
	}
}

func ConstraintConfigFromProto(pbConfig *pb.ConstraintConfig) ConstraintConfig {
	if pbConfig == nil {
		return ConstraintConfig{}
	}

	rateLimits := make([]RateLimitConfig, len(pbConfig.RateLimit))
	for i, rl := range pbConfig.RateLimit {
		rateLimits[i] = RateLimitConfigFromProto(rl)
	}

	throttles := make([]ThrottleConfig, len(pbConfig.Throttle))
	for i, th := range pbConfig.Throttle {
		throttles[i] = ThrottleConfigFromProto(th)
	}

	return ConstraintConfig{
		FunctionVersion: int(pbConfig.FunctionVersion),
		RateLimit:       rateLimits,
		Concurrency:     ConcurrencyConfigFromProto(pbConfig.Concurrency),
		Throttle:        throttles,
	}
}

func ConstraintCapacityItemToProto(item ConstraintCapacityItem) *pb.ConstraintCapacityItem {
	var kind pb.ConstraintApiConstraintKind
	if item.Kind != nil {
		kind = ConstraintKindToProto(*item.Kind)
	} else {
		kind = pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED
	}

	return &pb.ConstraintCapacityItem{
		Kind:   kind,
		Amount: int32(item.Amount),
	}
}

func ConstraintCapacityItemFromProto(pbItem *pb.ConstraintCapacityItem) ConstraintCapacityItem {
	if pbItem == nil {
		return ConstraintCapacityItem{}
	}

	var kind *ConstraintKind
	if pbItem.Kind != pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED {
		k := ConstraintKindFromProto(pbItem.Kind)
		kind = &k
	}

	return ConstraintCapacityItem{
		Kind:   kind,
		Amount: int(pbItem.Amount),
	}
}

func LeaseSourceToProto(source LeaseSource) *pb.LeaseSource {
	return &pb.LeaseSource{
		Service:            LeaseServiceToProto(source.Service),
		Location:           LeaseLocationToProto(source.Location),
		RunProcessingMode: RunProcessingModeToProto(source.RunProcessingMode),
	}
}

func LeaseSourceFromProto(pbSource *pb.LeaseSource) LeaseSource {
	if pbSource == nil {
		return LeaseSource{}
	}
	return LeaseSource{
		Service:            LeaseServiceFromProto(pbSource.Service),
		Location:           LeaseLocationFromProto(pbSource.Location),
		RunProcessingMode: RunProcessingModeFromProto(pbSource.RunProcessingMode),
	}
}

func CapacityCheckRequestToProto(req *CapacityCheckRequest) *pb.CapacityCheckRequest {
	if req == nil {
		return nil
	}
	return &pb.CapacityCheckRequest{
		AccountId: req.AccountID.String(),
	}
}

func CapacityCheckRequestFromProto(pbReq *pb.CapacityCheckRequest) (*CapacityCheckRequest, error) {
	if pbReq == nil {
		return nil, nil
	}

	accountID, err := uuid.Parse(pbReq.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	return &CapacityCheckRequest{
		AccountID: accountID,
	}, nil
}

func CapacityCheckResponseToProto(resp *CapacityCheckResponse) *pb.CapacityCheckResponse {
	if resp == nil {
		return nil
	}
	return &pb.CapacityCheckResponse{}
}

func CapacityCheckResponseFromProto(pbResp *pb.CapacityCheckResponse) *CapacityCheckResponse {
	if pbResp == nil {
		return nil
	}
	return &CapacityCheckResponse{}
}

func CapacityLeaseRequestToProto(req *CapacityLeaseRequest) *pb.CapacityLeaseRequest {
	if req == nil {
		return nil
	}

	requestedCapacity := make([]*pb.ConstraintCapacityItem, len(req.RequestedCapacity))
	for i, item := range req.RequestedCapacity {
		requestedCapacity[i] = ConstraintCapacityItemToProto(item)
	}

	return &pb.CapacityLeaseRequest{
		IdempotencyKey:    req.IdempotencyKey,
		AccountId:         req.AccountID.String(),
		EnvId:             req.EnvID.String(),
		FunctionId:        req.FunctionID.String(),
		Configuration:     ConstraintConfigToProto(req.Configuration),
		RequestedCapacity: requestedCapacity,
		CurrentTime:       timestamppb.New(req.CurrentTime),
		Duration:          durationpb.New(req.Duration),
		MaximumLifetime:   durationpb.New(req.MaximumLifetime),
		BlockingThreshold: durationpb.New(req.BlockingThreshold),
		Source:            LeaseSourceToProto(req.Source),
	}
}

func CapacityLeaseRequestFromProto(pbReq *pb.CapacityLeaseRequest) (*CapacityLeaseRequest, error) {
	if pbReq == nil {
		return nil, nil
	}

	accountID, err := uuid.Parse(pbReq.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	envID, err := uuid.Parse(pbReq.EnvId)
	if err != nil {
		return nil, fmt.Errorf("invalid env ID: %w", err)
	}

	functionID, err := uuid.Parse(pbReq.FunctionId)
	if err != nil {
		return nil, fmt.Errorf("invalid function ID: %w", err)
	}

	requestedCapacity := make([]ConstraintCapacityItem, len(pbReq.RequestedCapacity))
	for i, item := range pbReq.RequestedCapacity {
		requestedCapacity[i] = ConstraintCapacityItemFromProto(item)
	}

	var currentTime time.Time
	if pbReq.CurrentTime != nil {
		currentTime = pbReq.CurrentTime.AsTime()
	}

	var duration time.Duration
	if pbReq.Duration != nil {
		duration = pbReq.Duration.AsDuration()
	}

	var maximumLifetime time.Duration
	if pbReq.MaximumLifetime != nil {
		maximumLifetime = pbReq.MaximumLifetime.AsDuration()
	}

	var blockingThreshold time.Duration
	if pbReq.BlockingThreshold != nil {
		blockingThreshold = pbReq.BlockingThreshold.AsDuration()
	}

	return &CapacityLeaseRequest{
		IdempotencyKey:    pbReq.IdempotencyKey,
		AccountID:         accountID,
		EnvID:             envID,
		FunctionID:        functionID,
		Configuration:     ConstraintConfigFromProto(pbReq.Configuration),
		RequestedCapacity: requestedCapacity,
		CurrentTime:       currentTime,
		Duration:          duration,
		MaximumLifetime:   maximumLifetime,
		BlockingThreshold: blockingThreshold,
		Source:            LeaseSourceFromProto(pbReq.Source),
	}, nil
}

func CapacityLeaseResponseToProto(resp *CapacityLeaseResponse) *pb.CapacityLeaseResponse {
	if resp == nil {
		return nil
	}

	reservedCapacity := make([]*pb.ConstraintCapacityItem, len(resp.ReservedCapacity))
	for i, item := range resp.ReservedCapacity {
		reservedCapacity[i] = ConstraintCapacityItemToProto(item)
	}

	insufficientCapacity := make([]*pb.ConstraintCapacityItem, len(resp.InsufficientCapacity))
	for i, item := range resp.InsufficientCapacity {
		insufficientCapacity[i] = ConstraintCapacityItemToProto(item)
	}

	var leaseID *string
	if resp.LeaseID != nil {
		s := resp.LeaseID.String()
		leaseID = &s
	}

	return &pb.CapacityLeaseResponse{
		LeaseId:              leaseID,
		ReservedCapacity:     reservedCapacity,
		InsufficientCapacity: insufficientCapacity,
		RetryAfter:           timestamppb.New(resp.RetryAfter),
	}
}

func CapacityLeaseResponseFromProto(pbResp *pb.CapacityLeaseResponse) (*CapacityLeaseResponse, error) {
	if pbResp == nil {
		return nil, nil
	}

	reservedCapacity := make([]ConstraintCapacityItem, len(pbResp.ReservedCapacity))
	for i, item := range pbResp.ReservedCapacity {
		reservedCapacity[i] = ConstraintCapacityItemFromProto(item)
	}

	insufficientCapacity := make([]ConstraintCapacityItem, len(pbResp.InsufficientCapacity))
	for i, item := range pbResp.InsufficientCapacity {
		insufficientCapacity[i] = ConstraintCapacityItemFromProto(item)
	}

	var leaseID *ulid.ULID
	if pbResp.LeaseId != nil {
		parsed, err := ulid.Parse(*pbResp.LeaseId)
		if err != nil {
			return nil, fmt.Errorf("invalid lease ID: %w", err)
		}
		leaseID = &parsed
	}

	var retryAfter time.Time
	if pbResp.RetryAfter != nil {
		retryAfter = pbResp.RetryAfter.AsTime()
	}

	return &CapacityLeaseResponse{
		LeaseID:              leaseID,
		ReservedCapacity:     reservedCapacity,
		InsufficientCapacity: insufficientCapacity,
		RetryAfter:           retryAfter,
	}, nil
}

func CapacityExtendLeaseRequestToProto(req *CapacityExtendLeaseRequest) *pb.CapacityExtendLeaseRequest {
	if req == nil {
		return nil
	}
	return &pb.CapacityExtendLeaseRequest{
		IdempotencyKey: req.IdempotencyKey,
		AccountId:      req.AccountID.String(),
		LeaseId:        req.LeaseID.String(),
		Duration:       durationpb.New(req.Duration),
	}
}

func CapacityExtendLeaseRequestFromProto(pbReq *pb.CapacityExtendLeaseRequest) (*CapacityExtendLeaseRequest, error) {
	if pbReq == nil {
		return nil, nil
	}

	accountID, err := uuid.Parse(pbReq.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	leaseID, err := ulid.Parse(pbReq.LeaseId)
	if err != nil {
		return nil, fmt.Errorf("invalid lease ID: %w", err)
	}

	var duration time.Duration
	if pbReq.Duration != nil {
		duration = pbReq.Duration.AsDuration()
	}

	return &CapacityExtendLeaseRequest{
		IdempotencyKey: pbReq.IdempotencyKey,
		AccountID:      accountID,
		LeaseID:        leaseID,
		Duration:       duration,
	}, nil
}

func CapacityExtendLeaseResponseToProto(resp *CapacityExtendLeaseResponse) *pb.CapacityExtendLeaseResponse {
	if resp == nil {
		return nil
	}

	var leaseID *string
	if resp.LeaseID != nil {
		s := resp.LeaseID.String()
		leaseID = &s
	}

	return &pb.CapacityExtendLeaseResponse{
		LeaseId: leaseID,
	}
}

func CapacityExtendLeaseResponseFromProto(pbResp *pb.CapacityExtendLeaseResponse) (*CapacityExtendLeaseResponse, error) {
	if pbResp == nil {
		return nil, nil
	}

	var leaseID *ulid.ULID
	if pbResp.LeaseId != nil {
		parsed, err := ulid.Parse(*pbResp.LeaseId)
		if err != nil {
			return nil, fmt.Errorf("invalid lease ID: %w", err)
		}
		leaseID = &parsed
	}

	return &CapacityExtendLeaseResponse{
		LeaseID: leaseID,
	}, nil
}

func CapacityCommitRequestToProto(req *CapacityCommitRequest) *pb.CapacityCommitRequest {
	if req == nil {
		return nil
	}
	return &pb.CapacityCommitRequest{
		IdempotencyKey: req.IdempotencyKey,
		AccountId:      req.AccountID.String(),
		LeaseId:        req.LeaseID.String(),
	}
}

func CapacityCommitRequestFromProto(pbReq *pb.CapacityCommitRequest) (*CapacityCommitRequest, error) {
	if pbReq == nil {
		return nil, nil
	}

	accountID, err := uuid.Parse(pbReq.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	leaseID, err := ulid.Parse(pbReq.LeaseId)
	if err != nil {
		return nil, fmt.Errorf("invalid lease ID: %w", err)
	}

	return &CapacityCommitRequest{
		IdempotencyKey: pbReq.IdempotencyKey,
		AccountID:      accountID,
		LeaseID:        leaseID,
	}, nil
}

func CapacityCommitResponseToProto(resp *CapacityCommitResponse) *pb.CapacityCommitResponse {
	if resp == nil {
		return nil
	}
	return &pb.CapacityCommitResponse{}
}

func CapacityCommitResponseFromProto(pbResp *pb.CapacityCommitResponse) *CapacityCommitResponse {
	if pbResp == nil {
		return nil
	}
	return &CapacityCommitResponse{}
}

func CapacityRollbackRequestToProto(req *CapacityRollbackRequest) *pb.CapacityRollbackRequest {
	if req == nil {
		return nil
	}
	return &pb.CapacityRollbackRequest{
		IdempotencyKey: req.IdempotencyKey,
		AccountId:      req.AccountID.String(),
		LeaseId:        req.LeaseID.String(),
	}
}

func CapacityRollbackRequestFromProto(pbReq *pb.CapacityRollbackRequest) (*CapacityRollbackRequest, error) {
	if pbReq == nil {
		return nil, nil
	}

	accountID, err := uuid.Parse(pbReq.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	leaseID, err := ulid.Parse(pbReq.LeaseId)
	if err != nil {
		return nil, fmt.Errorf("invalid lease ID: %w", err)
	}

	return &CapacityRollbackRequest{
		IdempotencyKey: pbReq.IdempotencyKey,
		AccountID:      accountID,
		LeaseID:        leaseID,
	}, nil
}

func CapacityRollbackResponseToProto(resp *CapacityRollbackResponse) *pb.CapacityRollbackResponse {
	if resp == nil {
		return nil
	}
	return &pb.CapacityRollbackResponse{}
}

func CapacityRollbackResponseFromProto(pbResp *pb.CapacityRollbackResponse) *CapacityRollbackResponse {
	if pbResp == nil {
		return nil
	}
	return &CapacityRollbackResponse{}
}
