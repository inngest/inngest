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
	case ConstraintKindRateLimit:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT
	case ConstraintKindConcurrency:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY
	case ConstraintKindThrottle:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE
	default:
		return pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED
	}
}

func ConstraintKindFromProto(kind pb.ConstraintApiConstraintKind) ConstraintKind {
	switch kind {
	case pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT:
		return ConstraintKindRateLimit
	case pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY:
		return ConstraintKindConcurrency
	case pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE:
		return ConstraintKindThrottle
	default:
		return ConstraintKind("")
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
	case LeaseLocationUnknown:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_UNSPECIFIED
	case LeaseLocationScheduleRun:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN
	case LeaseLocationPartitionLease:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_PARTITION_LEASE
	case LeaseLocationItemLease:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE
	case LeaseLocationCheckpoint:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_CHECKPOINT
	default:
		return pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_UNSPECIFIED
	}
}

func LeaseLocationFromProto(location pb.ConstraintApiLeaseLocation) LeaseLocation {
	switch location {
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_UNSPECIFIED:
		return LeaseLocationUnknown
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN:
		return LeaseLocationScheduleRun
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_PARTITION_LEASE:
		return LeaseLocationPartitionLease
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE:
		return LeaseLocationItemLease
	case pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_CHECKPOINT:
		return LeaseLocationCheckpoint
	default:
		return LeaseLocationUnknown
	}
}

func LeaseServiceToProto(service LeaseService) pb.ConstraintApiLeaseService {
	switch service {
	case ServiceUnknown:
		return pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_UNSPECIFIED
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
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_UNSPECIFIED:
		return ServiceUnknown
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS:
		return ServiceNewRuns
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR:
		return ServiceExecutor
	case pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API:
		return ServiceAPI
	default:
		return ServiceUnknown
	}
}

func RateLimitConfigToProto(config RateLimitConfig) *pb.RateLimitConfig {
	return &pb.RateLimitConfig{
		Scope:             RateLimitScopeToProto(config.Scope),
		Limit:             int32(config.Limit),
		Period:            int32(config.Period),
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
		Period:            int(pbConfig.Period),
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
		AccountConcurrency:     int32(config.AccountConcurrency),
		FunctionConcurrency:    int32(config.FunctionConcurrency),
		AccountRunConcurrency:  int32(config.AccountRunConcurrency),
		FunctionRunConcurrency: int32(config.FunctionRunConcurrency),
		CustomConcurrencyKeys:  customKeys,
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
		AccountConcurrency:     int(pbConfig.AccountConcurrency),
		FunctionConcurrency:    int(pbConfig.FunctionConcurrency),
		AccountRunConcurrency:  int(pbConfig.AccountRunConcurrency),
		FunctionRunConcurrency: int(pbConfig.FunctionRunConcurrency),
		CustomConcurrencyKeys:  customKeys,
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

// New constraint type conversions
func RateLimitConstraintToProto(constraint RateLimitConstraint) *pb.RateLimitConstraint {
	return &pb.RateLimitConstraint{
		Scope:             RateLimitScopeToProto(constraint.Scope),
		KeyExpressionHash: constraint.KeyExpressionHash,
		EvaluatedKeyHash:  constraint.EvaluatedKeyHash,
	}
}

func RateLimitConstraintFromProto(pbConstraint *pb.RateLimitConstraint) RateLimitConstraint {
	if pbConstraint == nil {
		return RateLimitConstraint{}
	}
	return RateLimitConstraint{
		Scope:             RateLimitScopeFromProto(pbConstraint.Scope),
		KeyExpressionHash: pbConstraint.KeyExpressionHash,
		EvaluatedKeyHash:  pbConstraint.EvaluatedKeyHash,
	}
}

func ConcurrencyConstraintToProto(constraint ConcurrencyConstraint) *pb.ConcurrencyConstraint {
	return &pb.ConcurrencyConstraint{
		Mode:              ConcurrencyModeToProto(constraint.Mode),
		Scope:             ConcurrencyScopeToProto(constraint.Scope),
		KeyExpressionHash: constraint.KeyExpressionHash,
		EvaluatedKeyHash:  constraint.EvaluatedKeyHash,
		InProgressItemKey: constraint.InProgressItemKey,
	}
}

func ConcurrencyConstraintFromProto(pbConstraint *pb.ConcurrencyConstraint) ConcurrencyConstraint {
	if pbConstraint == nil {
		return ConcurrencyConstraint{}
	}
	return ConcurrencyConstraint{
		Mode:              ConcurrencyModeFromProto(pbConstraint.Mode),
		Scope:             ConcurrencyScopeFromProto(pbConstraint.Scope),
		KeyExpressionHash: pbConstraint.KeyExpressionHash,
		EvaluatedKeyHash:  pbConstraint.EvaluatedKeyHash,
		InProgressItemKey: pbConstraint.InProgressItemKey,
	}
}

func ThrottleConstraintToProto(constraint ThrottleConstraint) *pb.ThrottleConstraint {
	return &pb.ThrottleConstraint{
		Scope:             ThrottleScopeToProto(constraint.Scope),
		KeyExpressionHash: constraint.KeyExpressionHash,
		EvaluatedKeyHash:  constraint.EvaluatedKeyHash,
	}
}

func ThrottleConstraintFromProto(pbConstraint *pb.ThrottleConstraint) ThrottleConstraint {
	if pbConstraint == nil {
		return ThrottleConstraint{}
	}
	return ThrottleConstraint{
		Scope:             ThrottleScopeFromProto(pbConstraint.Scope),
		KeyExpressionHash: pbConstraint.KeyExpressionHash,
		EvaluatedKeyHash:  pbConstraint.EvaluatedKeyHash,
	}
}

func ConstraintItemToProto(item ConstraintItem) *pb.ConstraintItem {
	kind := ConstraintKindToProto(item.Kind)

	pbItem := &pb.ConstraintItem{
		Kind: kind,
	}

	if item.Concurrency != nil {
		pbItem.Concurrency = ConcurrencyConstraintToProto(*item.Concurrency)
	}

	if item.Throttle != nil {
		pbItem.Throttle = ThrottleConstraintToProto(*item.Throttle)
	}

	if item.RateLimit != nil {
		pbItem.RateLimit = RateLimitConstraintToProto(*item.RateLimit)
	}

	return pbItem
}

func ConstraintItemFromProto(pbItem *pb.ConstraintItem) ConstraintItem {
	if pbItem == nil {
		return ConstraintItem{}
	}

	item := ConstraintItem{
		Kind: ConstraintKindFromProto(pbItem.Kind),
	}

	if pbItem.Concurrency != nil {
		concurrency := ConcurrencyConstraintFromProto(pbItem.Concurrency)
		item.Concurrency = &concurrency
	}

	if pbItem.Throttle != nil {
		throttle := ThrottleConstraintFromProto(pbItem.Throttle)
		item.Throttle = &throttle
	}

	if pbItem.RateLimit != nil {
		rateLimit := RateLimitConstraintFromProto(pbItem.RateLimit)
		item.RateLimit = &rateLimit
	}

	return item
}

func ConstraintUsageToProto(usage ConstraintUsage) *pb.ConstraintUsage {
	return &pb.ConstraintUsage{
		Constraint: ConstraintItemToProto(usage.Constraint),
		Used:       int32(usage.Used),
		Limit:      int32(usage.Limit),
	}
}

func ConstraintUsageFromProto(pbUsage *pb.ConstraintUsage) ConstraintUsage {
	if pbUsage == nil {
		return ConstraintUsage{}
	}
	return ConstraintUsage{
		Constraint: ConstraintItemFromProto(pbUsage.Constraint),
		Used:       int(pbUsage.Used),
		Limit:      int(pbUsage.Limit),
	}
}

func CapacityLeaseToProto(lease CapacityLease) *pb.CapacityLease {
	return &pb.CapacityLease{
		LeaseId:        lease.LeaseID.String(),
		IdempotencyKey: lease.IdempotencyKey,
	}
}

func CapacityLeaseFromProto(pbLease *pb.CapacityLease) (CapacityLease, error) {
	if pbLease == nil {
		return CapacityLease{}, nil
	}

	leaseID, err := ulid.Parse(pbLease.LeaseId)
	if err != nil {
		return CapacityLease{}, fmt.Errorf("invalid lease ID: %w", err)
	}

	return CapacityLease{
		LeaseID:        leaseID,
		IdempotencyKey: pbLease.IdempotencyKey,
	}, nil
}

func LeaseSourceToProto(source LeaseSource) *pb.LeaseSource {
	return &pb.LeaseSource{
		Service:           LeaseServiceToProto(source.Service),
		Location:          LeaseLocationToProto(source.Location),
		RunProcessingMode: RunProcessingModeToProto(source.RunProcessingMode),
	}
}

func LeaseSourceFromProto(pbSource *pb.LeaseSource) LeaseSource {
	if pbSource == nil {
		return LeaseSource{}
	}
	return LeaseSource{
		Service:           LeaseServiceFromProto(pbSource.Service),
		Location:          LeaseLocationFromProto(pbSource.Location),
		RunProcessingMode: RunProcessingModeFromProto(pbSource.RunProcessingMode),
	}
}

func MigrationIdentifierToProto(migration MigrationIdentifier) *pb.MigrationIdentifier {
	return &pb.MigrationIdentifier{
		IsRateLimit: migration.IsRateLimit,
		QueueShard:  migration.QueueShard,
	}
}

func MigrationIdentifierFromProto(pbMigration *pb.MigrationIdentifier) MigrationIdentifier {
	if pbMigration == nil {
		return MigrationIdentifier{}
	}
	return MigrationIdentifier{
		IsRateLimit: pbMigration.IsRateLimit,
		QueueShard:  pbMigration.QueueShard,
	}
}

func CapacityCheckRequestToProto(req *CapacityCheckRequest) *pb.CapacityCheckRequest {
	if req == nil {
		return nil
	}

	constraints := make([]*pb.ConstraintItem, len(req.Constraints))
	for i, constraint := range req.Constraints {
		constraints[i] = ConstraintItemToProto(constraint)
	}

	return &pb.CapacityCheckRequest{
		AccountId:     req.AccountID.String(),
		EnvId:         req.EnvID.String(),
		FunctionId:    req.FunctionID.String(),
		Configuration: ConstraintConfigToProto(req.Configuration),
		Constraints:   constraints,
		Migration:     MigrationIdentifierToProto(req.Migration),
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

	envID, err := uuid.Parse(pbReq.EnvId)
	if err != nil {
		return nil, fmt.Errorf("invalid env ID: %w", err)
	}

	var functionID uuid.UUID
	if pbReq.FunctionId != "" {
		functionID, err = uuid.Parse(pbReq.FunctionId)
		if err != nil {
			return nil, fmt.Errorf("invalid function ID: %w", err)
		}
	}

	constraints := make([]ConstraintItem, len(pbReq.Constraints))
	for i, constraint := range pbReq.Constraints {
		constraints[i] = ConstraintItemFromProto(constraint)
	}

	return &CapacityCheckRequest{
		AccountID:     accountID,
		EnvID:         envID,
		FunctionID:    functionID,
		Configuration: ConstraintConfigFromProto(pbReq.Configuration),
		Constraints:   constraints,
		Migration:     MigrationIdentifierFromProto(pbReq.Migration),
	}, nil
}

func CapacityCheckResponseToProto(resp *CapacityCheckResponse) *pb.CapacityCheckResponse {
	if resp == nil {
		return nil
	}

	limitingConstraints := make([]*pb.ConstraintItem, len(resp.LimitingConstraints))
	for i, constraint := range resp.LimitingConstraints {
		limitingConstraints[i] = ConstraintItemToProto(constraint)
	}

	usage := make([]*pb.ConstraintUsage, len(resp.Usage))
	for i, u := range resp.Usage {
		usage[i] = ConstraintUsageToProto(u)
	}

	var retryAfter *timestamppb.Timestamp
	if !resp.RetryAfter.IsZero() {
		retryAfter = timestamppb.New(resp.RetryAfter)
	}

	return &pb.CapacityCheckResponse{
		AvailableCapacity:   int32(resp.AvailableCapacity),
		LimitingConstraints: limitingConstraints,
		Usage:               usage,
		FairnessReduction:   int32(resp.FairnessReduction),
		RetryAfter:          retryAfter,
	}
}

func CapacityCheckResponseFromProto(pbResp *pb.CapacityCheckResponse) *CapacityCheckResponse {
	if pbResp == nil {
		return nil
	}

	limitingConstraints := make([]ConstraintItem, len(pbResp.LimitingConstraints))
	for i, constraint := range pbResp.LimitingConstraints {
		limitingConstraints[i] = ConstraintItemFromProto(constraint)
	}

	usage := make([]ConstraintUsage, len(pbResp.Usage))
	for i, u := range pbResp.Usage {
		usage[i] = ConstraintUsageFromProto(u)
	}

	var retryAfter time.Time
	if pbResp.RetryAfter != nil {
		retryAfter = pbResp.RetryAfter.AsTime()
	}

	return &CapacityCheckResponse{
		AvailableCapacity:   int(pbResp.AvailableCapacity),
		LimitingConstraints: limitingConstraints,
		Usage:               usage,
		FairnessReduction:   int(pbResp.FairnessReduction),
		RetryAfter:          retryAfter,
	}
}

func CapacityAcquireRequestToProto(req *CapacityAcquireRequest) *pb.CapacityAcquireRequest {
	if req == nil {
		return nil
	}

	constraints := make([]*pb.ConstraintItem, len(req.Constraints))
	for i, item := range req.Constraints {
		constraints[i] = ConstraintItemToProto(item)
	}

	leaseRunIDs := make(map[string]string)
	for leaseIdempotencyKey, runID := range req.LeaseRunIDs {
		leaseRunIDs[leaseIdempotencyKey] = runID.String()
	}

	return &pb.CapacityAcquireRequest{
		IdempotencyKey:       req.IdempotencyKey,
		AccountId:            req.AccountID.String(),
		EnvId:                req.EnvID.String(),
		FunctionId:           req.FunctionID.String(),
		Configuration:        ConstraintConfigToProto(req.Configuration),
		Constraints:          constraints,
		Amount:               int32(req.Amount),
		LeaseIdempotencyKeys: req.LeaseIdempotencyKeys,
		LeaseRunIds:          leaseRunIDs,
		CurrentTime:          timestamppb.New(req.CurrentTime),
		Duration:             durationpb.New(req.Duration),
		MaximumLifetime:      durationpb.New(req.MaximumLifetime),
		BlockingThreshold:    durationpb.New(req.BlockingThreshold),
		Source:               LeaseSourceToProto(req.Source),
		Migration:            MigrationIdentifierToProto(req.Migration),
	}
}

func CapacityAcquireRequestFromProto(pbReq *pb.CapacityAcquireRequest) (*CapacityAcquireRequest, error) {
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

	constraints := make([]ConstraintItem, len(pbReq.Constraints))
	for i, item := range pbReq.Constraints {
		constraints[i] = ConstraintItemFromProto(item)
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

	leaseRunIDs := make(map[string]ulid.ULID)
	for leaseIdempotencyKey, runID := range pbReq.LeaseRunIds {
		parsed, err := ulid.Parse(runID)
		if err != nil {
			return nil, fmt.Errorf("invalid run ID: %w", err)
		}
		leaseRunIDs[leaseIdempotencyKey] = parsed
	}

	return &CapacityAcquireRequest{
		IdempotencyKey:       pbReq.IdempotencyKey,
		AccountID:            accountID,
		EnvID:                envID,
		FunctionID:           functionID,
		Configuration:        ConstraintConfigFromProto(pbReq.Configuration),
		Constraints:          constraints,
		Amount:               int(pbReq.Amount),
		LeaseIdempotencyKeys: pbReq.LeaseIdempotencyKeys,
		LeaseRunIDs:          leaseRunIDs,
		CurrentTime:          currentTime,
		Duration:             duration,
		MaximumLifetime:      maximumLifetime,
		BlockingThreshold:    blockingThreshold,
		Source:               LeaseSourceFromProto(pbReq.Source),
		Migration:            MigrationIdentifierFromProto(pbReq.Migration),
	}, nil
}

func CapacityAcquireResponseToProto(resp *CapacityAcquireResponse) *pb.CapacityAcquireResponse {
	if resp == nil {
		return nil
	}

	leases := make([]*pb.CapacityLease, len(resp.Leases))
	for i, lease := range resp.Leases {
		leases[i] = CapacityLeaseToProto(lease)
	}

	limitingConstraints := make([]*pb.ConstraintItem, len(resp.LimitingConstraints))
	for i, constraint := range resp.LimitingConstraints {
		limitingConstraints[i] = ConstraintItemToProto(constraint)
	}

	return &pb.CapacityAcquireResponse{
		Leases:              leases,
		LimitingConstraints: limitingConstraints,
		RetryAfter:          timestamppb.New(resp.RetryAfter),
		FairnessReduction:   int32(resp.FairnessReduction),
	}
}

func CapacityAcquireResponseFromProto(pbResp *pb.CapacityAcquireResponse) (*CapacityAcquireResponse, error) {
	if pbResp == nil {
		return nil, nil
	}

	leases := make([]CapacityLease, len(pbResp.Leases))
	for i, pbLease := range pbResp.Leases {
		lease, err := CapacityLeaseFromProto(pbLease)
		if err != nil {
			return nil, fmt.Errorf("invalid lease at index %d: %w", i, err)
		}
		leases[i] = lease
	}

	limitingConstraints := make([]ConstraintItem, len(pbResp.LimitingConstraints))
	for i, constraint := range pbResp.LimitingConstraints {
		limitingConstraints[i] = ConstraintItemFromProto(constraint)
	}

	var retryAfter time.Time
	if pbResp.RetryAfter != nil {
		retryAfter = pbResp.RetryAfter.AsTime()
	}

	return &CapacityAcquireResponse{
		Leases:              leases,
		LimitingConstraints: limitingConstraints,
		RetryAfter:          retryAfter,
		FairnessReduction:   int(pbResp.FairnessReduction),
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
		Migration:      MigrationIdentifierToProto(req.Migration),
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
		Migration:      MigrationIdentifierFromProto(pbReq.Migration),
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

func CapacityReleaseRequestToProto(req *CapacityReleaseRequest) *pb.CapacityReleaseRequest {
	if req == nil {
		return nil
	}
	return &pb.CapacityReleaseRequest{
		IdempotencyKey: req.IdempotencyKey,
		AccountId:      req.AccountID.String(),
		LeaseId:        req.LeaseID.String(),
		Migration:      MigrationIdentifierToProto(req.Migration),
	}
}

func CapacityReleaseRequestFromProto(pbReq *pb.CapacityReleaseRequest) (*CapacityReleaseRequest, error) {
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

	return &CapacityReleaseRequest{
		IdempotencyKey: pbReq.IdempotencyKey,
		AccountID:      accountID,
		LeaseID:        leaseID,
		Migration:      MigrationIdentifierFromProto(pbReq.Migration),
	}, nil
}

func CapacityReleaseResponseToProto(resp *CapacityReleaseResponse) *pb.CapacityReleaseResponse {
	if resp == nil {
		return nil
	}
	return &pb.CapacityReleaseResponse{}
}

func CapacityReleaseResponseFromProto(pbResp *pb.CapacityReleaseResponse) *CapacityReleaseResponse {
	if pbResp == nil {
		return nil
	}
	return &CapacityReleaseResponse{}
}
