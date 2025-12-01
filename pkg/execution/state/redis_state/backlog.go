package redis_state

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math/rand"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"gonum.org/v1/gonum/stat/sampleuv"
)

// NOTE: there's no logic behind this number, it's just a random pick for now
var ThrottleBackoffMultiplierThreshold = 15 * time.Second

var (
	ErrBacklogNotFound = fmt.Errorf("backlog not found")

	ErrBacklogPeekMaxExceedsLimits = fmt.Errorf("backlog peek exceeded the maximum limit")

	ErrBacklogGarbageCollected = fmt.Errorf("backlog was garbage-collected")
)

// BacklogManager defines the interface for backlog operations
type BacklogManager interface {
	BacklogPeek(ctx context.Context, b *QueueBacklog, from time.Time, until time.Time, limit int64, opts ...PeekOpt) ([]*osqueue.QueueItem, int, error)
	BacklogRefill(ctx context.Context, b *QueueBacklog, sp *QueueShadowPartition, refillUntil time.Time, refillItems []string, latestConstraints PartitionConstraintConfig, options ...backlogRefillOptionFn) (*BacklogRefillResult, error)
	ItemBacklog(ctx context.Context, i osqueue.QueueItem) QueueBacklog
	ItemShadowPartition(ctx context.Context, i osqueue.QueueItem) QueueShadowPartition
}

type PartitionConstraintConfig struct {
	FunctionVersion int `json:"fv,omitempty,omitzero"`

	Concurrency PartitionConcurrency `json:"c,omitempty,omitzero"`

	// Throttle configuration, optionally specifying key. If no key is set, the throttle value will be the function ID.
	Throttle *PartitionThrottle `json:"t,omitempty,omitzero"`
}

type CustomConcurrencyLimit struct {
	Mode                enums.ConcurrencyMode  `json:"m"`
	Scope               enums.ConcurrencyScope `json:"s"`
	HashedKeyExpression string                 `json:"k"`
	Limit               int                    `json:"l"`
}

type PartitionThrottle struct {
	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`

	// Limit is the actual rate limit
	Limit int `json:"l"`
	// Burst is the busrsable capacity of the rate limit
	Burst int `json:"b"`
	// Period is the rate limit period, in seconds
	Period int `json:"p"`
}

type PartitionConcurrency struct {
	// SystemConcurrency represents the concurrency limit to apply to system queues. Unset on regular function partitions.
	SystemConcurrency int `json:"sc,omitempty"`

	// AccountConcurrency represents the global account concurrency limit. This is unset on system queues.
	AccountConcurrency int `json:"ac,omitempty"`

	// FunctionConcurrency represents the function concurrency limit.
	FunctionConcurrency int `json:"fc,omitempty"`

	// AccountRunConcurrency represents the global account run concurrency limit (how many active runs per account). This is unset on system queues.
	AccountRunConcurrency int `json:"arc,omitempty"`

	// FunctionRunConcurrency represents the function run concurrency limit (how many active runs allowed per function).
	FunctionRunConcurrency int `json:"frc,omitempty"`

	// Up to two custom concurrency keys on user-defined scopes, optionally specifying a key. The key is required
	// on env or account level scopes.
	CustomConcurrencyKeys []CustomConcurrencyLimit `json:"cck,omitempty"`
}

type QueueShadowPartition struct {
	// PartitionID is the function ID or system queue name. The shadow partition
	// ID is the same as the partition ID used across the queue.
	PartitionID string `json:"id,omitempty"`

	// FunctionVersion represents the current function version represented by this shadow partition.
	// Whenever a newer function version is enqueued, the concurrency keys and limits in here will be adjusted
	// accordingly as part of enqueue_to_backlog().
	// System queues do not have function versions.
	FunctionVersion int `json:"fv"`

	// LeaseID represents a lease on this shadow partition.  If the LeaseID is not nil,
	// this partition can be claimed by a shared-nothing refill worker to work on the
	// backlogs within this shadow partition.
	LeaseID *ulid.ULID `json:"leaseID,omitempty"`

	FunctionID      *uuid.UUID `json:"fid,omitempty"`
	EnvID           *uuid.UUID `json:"eid,omitempty"`
	AccountID       *uuid.UUID `json:"aid,omitempty"`
	SystemQueueName *string    `json:"queueName,omitempty"`
}

func (sp QueueShadowPartition) Identifier() PartitionIdentifier {
	fnID := uuid.Nil
	if sp.FunctionID != nil {
		fnID = *sp.FunctionID
	}

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	envID := uuid.Nil
	if sp.EnvID != nil {
		envID = *sp.EnvID
	}

	return PartitionIdentifier{
		SystemQueueName: sp.SystemQueueName,

		AccountID:  accountID,
		EnvID:      envID,
		FunctionID: fnID,
	}
}

func (sp QueueShadowPartition) GetAccountID() uuid.UUID {
	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	return accountID
}

// readyQueueKey returns the ZSET key to the ready queue
func (sp QueueShadowPartition) readyQueueKey(kg QueueKeyGenerator) string {
	return kg.PartitionQueueSet(enums.PartitionTypeDefault, sp.PartitionID, "")
}

// inProgressKey returns the key storing the in progress set for the shadow partition
func (sp QueueShadowPartition) inProgressKey(kg QueueKeyGenerator) string {
	return kg.Concurrency("p", sp.PartitionID)
}

// activeKey returns the key storing the active set for the shadow partition
func (sp QueueShadowPartition) activeKey(kg QueueKeyGenerator) string {
	return kg.ActiveSet("p", sp.PartitionID)
}

func (sp QueueShadowPartition) keyQueuesEnabled(ctx context.Context, q *queue) bool {
	if sp.SystemQueueName != nil {
		return false
	}

	if sp.AccountID == nil || q.allowKeyQueues == nil {
		return false
	}

	return q.allowKeyQueues(ctx, *sp.AccountID)
}

func (q *PartitionConstraintConfig) CustomConcurrencyLimit(n int) int {
	if n < 0 || n > len(q.Concurrency.CustomConcurrencyKeys) {
		return 0
	}

	key := q.Concurrency.CustomConcurrencyKeys[n-1]

	return key.Limit
}

func (q PartitionConstraintConfig) CustomConcurrencyKey(kg QueueKeyGenerator, b *QueueBacklog, n int) (string, int) {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.Concurrency("", ""), 0
	}

	backlogKey := b.ConcurrencyKeys[n-1]

	for _, key := range q.Concurrency.CustomConcurrencyKeys {
		if key.Scope == backlogKey.Scope && key.HashedKeyExpression == backlogKey.HashedKeyExpression {
			// Return concrete key with latest limit from shadow partition
			return backlogKey.concurrencyKey(kg), key.Limit
		}
	}

	return kg.Concurrency("", ""), 0
}

// accountInProgressKey returns the key storing the in progress set for the shadow partition's account
func (sp QueueShadowPartition) accountInProgressKey(kg QueueKeyGenerator) string {
	// Do not track account concurrency for system queues
	if sp.SystemQueueName != nil {
		return kg.Concurrency("", "")
	}

	// This should never be unset
	if sp.AccountID == nil {
		return kg.Concurrency("account", "")
	}

	return kg.Concurrency("account", sp.AccountID.String())
}

// accountActiveKey returns the key storing the active set for the shadow partition's account
func (sp QueueShadowPartition) accountActiveKey(kg QueueKeyGenerator) string {
	// Do not track account concurrency for system queues
	if sp.SystemQueueName != nil {
		return kg.ActiveSet("", "")
	}

	// This should never be unset
	if sp.AccountID == nil {
		return kg.ActiveSet("account", "")
	}

	return kg.ActiveSet("account", sp.AccountID.String())
}

func (sp QueueShadowPartition) accountActiveRunKey(kg QueueKeyGenerator) string {
	// Do not track account run concurrency for system queues
	if sp.SystemQueueName != nil {
		return kg.ActiveRunsSet("", "")
	}

	// This should never be unset
	if sp.AccountID == nil {
		return kg.ActiveRunsSet("account", "")
	}

	return kg.ActiveRunsSet("account", sp.AccountID.String())
}

func (sp QueueShadowPartition) activeRunKey(kg QueueKeyGenerator) string {
	return kg.ActiveRunsSet("p", sp.PartitionID)
}

// BacklogConcurrencyKey represents a custom concurrency key, which can be scoped to the function, environment, or account.
//
// Note: BacklogConcurrencyKey is only used for custom concurrency keys with a defined `key`.
// In the case of configuring concurrency on the function scope without providing a `key`, the default backlog will be used.
type BacklogConcurrencyKey struct {
	// CanonicalKeyID is the combined concurrency key (e.g. a:<account ID>:hash("customer1"))
	CanonicalKeyID string `json:"kid"`

	Scope enums.ConcurrencyScope `json:"cs"`

	// EntityID stores the accountID, envID, or fnID for the respective concurrency scope
	EntityID uuid.UUID `json:"cse"`

	// HashedKeyExpression is the hashed concurrency key expression (e.g. hash("event.data.customerId"))
	HashedKeyExpression string `json:"cke"`

	// HashedValue is the hashed concurrency key value (e.g. hash("customer1"))
	HashedValue string `json:"ckv"`

	// UnhashedValue is the unhashed evaluated key (e.g. "customer1")
	// This may be truncated for long values and may only be used for observability and debugging.
	UnhashedValue string `json:"ckuv"`

	// ConcurrencyMode represents the concurrency mode.
	ConcurrencyMode enums.ConcurrencyMode `json:"mode"`
}

type BacklogThrottle struct {
	// ThrottleKey is the hashed evaluated throttle key (e.g. hash("customer1")) or function ID (e.g. hash(fnID))
	ThrottleKey string `json:"tk,omitempty"`

	// ThrottleKeyRawValue is the unhashed evaluated throttle key (e.g. "customer1") or function ID.
	// This may be truncated for long values and may only be used for observability and debugging.
	ThrottleKeyRawValue string `json:"tkv,omitempty"`

	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`
}

type QueueBacklog struct {
	BacklogID               string `json:"id,omitempty"`
	ShadowPartitionID       string `json:"sid,omitempty"`
	EarliestFunctionVersion int    `json:"fv,omitempty"`

	// Start marks backlogs representing items with KindStart.
	Start bool `json:"start,omitempty"`

	// Set for backlogs representing custom concurrency keys
	ConcurrencyKeys []BacklogConcurrencyKey `json:"ck,omitempty"`

	// Set for backlogs containing start items only for a given throttle configuration
	Throttle *BacklogThrottle `json:"t,omitempty"`

	SuccessiveThrottleConstrained          int `json:"stc,omitzero"`
	SuccessiveCustomConcurrencyConstrained int `json:"sccc,omitzero"`
}

// ItemBacklog creates a backlog for the given item. The returned backlog may represent current _or_ past
// configurations, in case the queue item has existed for some time and the function was updated in the meantime.
//
// For the sake of consistency and cleanup, ItemBacklog *must* always return the same configuration,
// over the complete lifecycle of a queue item. To this end, the function exclusively retrieves data
// from the queue item, has no side effects, and does not make any calls to external data stores.
func (q *queue) ItemBacklog(ctx context.Context, i osqueue.QueueItem) QueueBacklog {
	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		q.log.Warn("backlogs encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set", "item", i)
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		q.log.Error("backlogs encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName", "item", i)
	}

	if queueName != nil {
		// Simply use default backlog for system queues - there shouldn't be any concurrency or throttle keys involved.
		return QueueBacklog{
			ShadowPartitionID: *queueName,
			BacklogID:         fmt.Sprintf("system:%s", *queueName),
		}
	}

	b := QueueBacklog{
		BacklogID:         fmt.Sprintf("fn:%s", i.FunctionID),
		ShadowPartitionID: i.FunctionID.String(),

		// Store earliest function version. Since we do not update backlog metadata,
		// this may be older than the latest items in the backlog.
		EarliestFunctionVersion: i.Data.Identifier.WorkflowVersion,

		// Start items should be moved into their own backlog. This is useful for
		// function run concurrency: To determine how many new runs can start, we can
		// calculate the remaining run capacity and refill as many items from the start backlog.
		Start: i.Data.Kind == osqueue.KindStart,
	}
	if b.Start {
		b.BacklogID += ":start"
	}

	// Enqueue start items to throttle backlog if throttle is configured
	if i.Data.Throttle != nil && b.Start {
		// This is always specified, even if no key was configured in the function definition.
		// In that case, the Throttle Key is the hashed function ID. See Schedule() for more details.
		b.Throttle = &BacklogThrottle{
			ThrottleKey:               i.Data.Throttle.Key,
			ThrottleKeyExpressionHash: i.Data.Throttle.KeyExpressionHash,
		}

		b.BacklogID += fmt.Sprintf(":t<%s:%s>", i.Data.Throttle.KeyExpressionHash, i.Data.Throttle.Key)

		if i.Data.Throttle.UnhashedThrottleKey != "" {
			unhashedKey := i.Data.Throttle.UnhashedThrottleKey
			// truncate - just in case
			if len(unhashedKey) > 512 {
				unhashedKey = unhashedKey[:512]
			}
			b.Throttle.ThrottleKeyRawValue = unhashedKey
		}
	}

	concurrencyKeys := i.Data.GetConcurrencyKeys()
	if len(concurrencyKeys) > 0 {
		// Create custom concurrency key backlog
		b.ConcurrencyKeys = make([]BacklogConcurrencyKey, len(concurrencyKeys))

		for i, key := range concurrencyKeys {
			scope, entityID, checksum, _ := key.ParseKey()

			b.BacklogID += fmt.Sprintf(":c%d<%s:%s>", i+1, key.Hash, util.XXHash(key.Key))

			b.ConcurrencyKeys[i] = BacklogConcurrencyKey{
				CanonicalKeyID: key.Key,

				Scope: scope,

				// Account ID, Env ID, or Function ID to apply to the concurrency key to
				EntityID: entityID,

				// Hashed expression to identify which key this is in the shadow partition concurrency key list
				HashedKeyExpression: key.Hash, // hash("event.data.customerID")

				// Evaluated hashed and unhashed values
				HashedValue: checksum, // hash("customer1")

				// Just for debugging purposes (only passed on Enqueue after Schedule or backlog normalization)
				UnhashedValue: key.UnhashedEvaluatedKeyValue, // "customer1"
			}
		}
	}

	return b
}

func (q *queue) ItemShadowPartition(ctx context.Context, i osqueue.QueueItem) QueueShadowPartition {
	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		q.log.Warn("shadow partitions encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set", "item", i)
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		q.log.Error("shadow partitions encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName", "item", i)
	}

	accountID := i.Data.Identifier.AccountID

	// The only case when we manually set a queueName is for system partitions
	if queueName != nil {
		var aID *uuid.UUID
		if accountID != uuid.Nil {
			aID = &accountID
		}

		return QueueShadowPartition{
			PartitionID:     *queueName,
			SystemQueueName: queueName,

			AccountID: aID,
		}
	}

	if accountID == uuid.Nil {
		stack := string(debug.Stack())
		q.log.Error("unexpected missing accountID in ItemShadowPartition call", "item", i, "stack", stack)
	}

	fnID := i.FunctionID
	if fnID == uuid.Nil {
		stack := string(debug.Stack())
		q.log.Error("unexpected missing functionID in ItemShadowPartition call", "item", i, "stack", stack)
	}

	return QueueShadowPartition{
		PartitionID:     fnID.String(),
		FunctionVersion: i.Data.Identifier.WorkflowVersion,

		// Identifiers
		FunctionID: &fnID,
		EnvID:      &i.WorkspaceID,
		AccountID:  &accountID,
	}
}

func (b QueueBacklog) isDefault() bool {
	return b.Throttle == nil && len(b.ConcurrencyKeys) == 0
}

func (b QueueBacklog) isOutdated(constraints PartitionConstraintConfig) enums.QueueNormalizeReason {
	// If the backlog represents newer items than the constraints we're working on,
	// do not attempt to mark the backlog as outdated. Constraints MUST be >= backlog function version at all times.
	if b.EarliestFunctionVersion > 0 && constraints.FunctionVersion > 0 && b.EarliestFunctionVersion > constraints.FunctionVersion {
		return enums.QueueNormalizeReasonUnchanged
	}

	// If this is the default backlog, don't normalize.
	// If custom concurrency keys were added, previously-enqueued items
	// in the default backlog do not have custom concurrency keys set.
	if b.isDefault() {
		return enums.QueueNormalizeReasonUnchanged
	}

	// Throttle removed - move items back to default backlog
	if b.Throttle != nil && constraints.Throttle == nil {
		return enums.QueueNormalizeReasonThrottleRemoved
	}

	// Throttle key changed - move from old throttle key backlogs to the new throttle key backlogs
	if b.Throttle != nil && constraints.Throttle != nil && b.Throttle.ThrottleKeyExpressionHash != constraints.Throttle.ThrottleKeyExpressionHash {
		return enums.QueueNormalizeReasonThrottleKeyChanged
	}

	// Concurrency key count does not match
	if len(b.ConcurrencyKeys) != len(constraints.Concurrency.CustomConcurrencyKeys) {
		return enums.QueueNormalizeReasonCustomConcurrencyKeyCountMismatch
	}

	// All concurrency keys on backlog must be found on partition
	// This is quadratic but each backlog and shadow partition can only have up to 2 keys, so it's bounded.
	for _, backlogKey := range b.ConcurrencyKeys {
		hasKey := false
		for _, shadowPartitionKey := range constraints.Concurrency.CustomConcurrencyKeys {
			if shadowPartitionKey.Mode == backlogKey.ConcurrencyMode && shadowPartitionKey.Scope == backlogKey.Scope && shadowPartitionKey.HashedKeyExpression == backlogKey.HashedKeyExpression {
				hasKey = true
				break
			}
		}

		if !hasKey {
			return enums.QueueNormalizeReasonCustomConcurrencyKeyNotFoundOnShadowPartition
		}
	}

	// We don't have to check that all keys on the shadow partition must be found on
	// the backlog as we've compared the length, so the previous check will account for
	// missing/different keys.

	return enums.QueueNormalizeReasonUnchanged
}

// customKeyInProgress returns the key to the "in progress" ZSET
func (b QueueBacklog) customKeyInProgress(kg QueueKeyGenerator, n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.Concurrency("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return key.concurrencyKey(kg)
}

func (b BacklogConcurrencyKey) concurrencyKey(kg QueueKeyGenerator) string {
	// Concurrency accounting keys are made up of three parts:
	// - The scope (account, environment, function) to apply the concurrency limit on
	// - The entity (account ID, envID, or function ID) based on the scope
	// - The dynamic key value (hashed evaluated expression)
	return kg.Concurrency("custom", b.CanonicalKeyID)
}

// customKeyActive returns the key to the active set for the given custom concurrency key
func (b QueueBacklog) customKeyActive(kg QueueKeyGenerator, n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.ActiveSet("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return key.activeKey(kg)
}

// customKeyActiveRuns returns the key to the active runs counter for the given custom concurrency key
func (b QueueBacklog) customKeyActiveRuns(kg QueueKeyGenerator, n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.ActiveRunsSet("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return key.activeRunsKey(kg)
}

func (b QueueBacklog) inProgressLeasesCustomKey(cm constraintapi.RolloutKeyGenerator, kg QueueKeyGenerator, accountID *uuid.UUID, n int) string {
	if cm == nil {
		return kg.Concurrency("", "")
	}

	if accountID == nil {
		return kg.Concurrency("", "")
	}

	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.Concurrency("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return key.inProgressLeasesKey(cm, *accountID)
}

func (b BacklogConcurrencyKey) activeKey(kg QueueKeyGenerator) string {
	// Concurrency accounting keys are made up of three parts:
	// - The scope (account, environment, function) to apply the concurrency limit on
	// - The entity (account ID, envID, or function ID) based on the scope
	// - The dynamic key value (hashed evaluated expression)
	return kg.ActiveSet("custom", b.CanonicalKeyID)
}

func (b BacklogConcurrencyKey) activeRunsKey(kg QueueKeyGenerator) string {
	return kg.ActiveRunsSet("custom", b.CanonicalKeyID)
}

func (b BacklogConcurrencyKey) inProgressLeasesKey(cm constraintapi.RolloutKeyGenerator, accountID uuid.UUID) string {
	return cm.KeyInProgressLeasesCustom(accountID, b.Scope, b.EntityID, b.HashedKeyExpression, b.HashedValue)
}

// activeKey returns backlog compound active key
func (b QueueBacklog) activeKey(kg QueueKeyGenerator) string {
	return kg.ActiveSet("compound", b.BacklogID)
}

func (b QueueBacklog) customConcurrencyKeyID(n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return ""
	}

	key := b.ConcurrencyKeys[n-1]
	return key.CanonicalKeyID
}

func (b QueueBacklog) requeueBackOff(now time.Time, constraint enums.QueueConstraint) time.Time {
	switch constraint {
	case enums.QueueConstraintCustomConcurrencyKey1, enums.QueueConstraintCustomConcurrencyKey2:
		next := time.Duration(b.SuccessiveCustomConcurrencyConstrained) * time.Second

		if next > PartitionConcurrencyLimitRequeueExtension {
			next = PartitionConcurrencyLimitRequeueExtension
		}

		return now.Add(next)
	default:
		return now.Add(BacklogDefaultRequeueExtension)
	}
}

type BacklogRefillResult struct {
	Constraint        enums.QueueConstraint
	Refilled          int
	TotalBacklogCount int
	BacklogCountUntil int
	Capacity          int
	Refill            int
	RefilledItems     []string
	RetryAt           time.Time
}

type backlogRefillOptions struct {
	constraintCheckIdempotencyKey string
	disableConstraintChecks       bool
	capacityLeaseIDs              []ulid.ULID
}

type backlogRefillOptionFn func(o *backlogRefillOptions)

func WithBacklogRefillConstraintCheckIdempotencyKey(idempotencyKey string) backlogRefillOptionFn {
	return func(o *backlogRefillOptions) {
		o.constraintCheckIdempotencyKey = idempotencyKey
	}
}

func WithBacklogRefillDisableConstraintChecks(disableConstraintChecks bool) backlogRefillOptionFn {
	return func(o *backlogRefillOptions) {
		o.disableConstraintChecks = disableConstraintChecks
	}
}

func WithBacklogRefillItemCapacityLeaseIDs(itemCapacityLeaseIDs []ulid.ULID) backlogRefillOptionFn {
	return func(o *backlogRefillOptions) {
		o.capacityLeaseIDs = itemCapacityLeaseIDs
	}
}

func (q *queue) BacklogRefill(
	ctx context.Context,
	b *QueueBacklog,
	sp *QueueShadowPartition,
	refillUntil time.Time,
	refillItems []string,
	latestConstraints PartitionConstraintConfig,
	options ...backlogRefillOptionFn,
) (*BacklogRefillResult, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "BacklogRefill"), redis_telemetry.ScopeQueue)

	o := &backlogRefillOptions{}
	for _, opt := range options {
		opt(o)
	}

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for BacklogRefill: %s", q.primaryQueueShard.Kind)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	nowMS := q.clock.Now().UnixMilli()

	var (
		throttleKey                                  string
		throttleLimit, throttleBurst, throttlePeriod int
	)
	if latestConstraints.Throttle != nil && b.Throttle != nil {
		throttleKey = b.Throttle.ThrottleKey
		throttleLimit = latestConstraints.Throttle.Limit
		throttleBurst = latestConstraints.Throttle.Burst
		throttlePeriod = latestConstraints.Throttle.Period
	}

	keys := []string{
		kg.ShadowPartitionMeta(),
		kg.BacklogMeta(),

		kg.BacklogSet(b.BacklogID),
		kg.ShadowPartitionSet(sp.PartitionID),
		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(accountID),

		sp.readyQueueKey(kg),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(accountID),

		kg.QueueItem(),

		// Constraint-related accounting keys
		sp.accountActiveKey(kg),  // account active
		sp.activeKey(kg),         // partition active
		b.customKeyActive(kg, 1), // custom key 1
		b.customKeyActive(kg, 2), // custom key 2
		b.activeKey(kg),          // compound key (active for this backlog)

		// Active run sets
		// kg.RunActiveSet(i.Data.Identifier.RunID), -> dynamically constructed in script for each item
		sp.accountActiveRunKey(kg),   // Set for active runs in account
		sp.activeRunKey(kg),          // Set for active runs in partition
		b.customKeyActiveRuns(kg, 1), // Set for active runs with custom concurrency key 1
		b.customKeyActiveRuns(kg, 2), // Set for active runs with custom concurrency key 2

		kg.BacklogActiveCheckSet(),
		kg.BacklogActiveCheckCooldown(b.BacklogID),

		kg.PartitionNormalizeSet(sp.PartitionID),

		// Constraint API rollout
		q.keyConstraintCheckIdempotency(sp.AccountID, o.constraintCheckIdempotencyKey),
	}

	// Don't check constraints if
	// - key queues have been disabled for this function (refill as quickly as possible)
	// - capacity leases were successfully acquired
	checkConstraints := sp.keyQueuesEnabled(ctx, q)
	if o.disableConstraintChecks {
		checkConstraints = false
	}

	checkConstraintsVal := "1"
	if !checkConstraints {
		checkConstraintsVal = "0"
	}

	// Enable conditional spot checking (probability in queue settings + feature flag)
	refillProbability, _ := q.activeSpotCheckProbability(ctx, accountID)
	shouldSpotCheckActiveSet := checkConstraints && rand.Intn(100) <= refillProbability

	// Ensure capacityLeaseIDs is never nil to avoid JSON marshaling to "null"
	capacityLeaseIDs := o.capacityLeaseIDs
	if capacityLeaseIDs == nil {
		capacityLeaseIDs = []ulid.ULID{}
	}

	args, err := StrSlice([]any{
		b.BacklogID,
		sp.PartitionID,
		accountID,
		refillUntil.UnixMilli(),
		refillItems,
		nowMS,

		latestConstraints.Concurrency.AccountConcurrency,
		latestConstraints.Concurrency.FunctionConcurrency,
		latestConstraints.CustomConcurrencyLimit(1),
		latestConstraints.CustomConcurrencyLimit(2),

		throttleKey,
		throttleLimit,
		throttleBurst,
		throttlePeriod,

		kg.QueuePrefix(),
		checkConstraintsVal,
		shouldSpotCheckActiveSet,

		capacityLeaseIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	res, err := scripts["queue/backlogRefill"].Exec(
		redis_telemetry.WithScriptName(ctx, "backlogRefill"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).ToAny()
	if err != nil {
		return nil, fmt.Errorf("error refilling backlog: %w", err)
	}

	returnTuple, ok := res.([]any)
	if !ok || len(returnTuple) != 8 {
		return nil, fmt.Errorf("expected return tuple to include 7 items")
	}

	status, ok := returnTuple[0].(int64)
	if !ok {
		return nil, fmt.Errorf("missing status in returned tuple")
	}

	refillCount, ok := returnTuple[1].(int64)
	if !ok {
		return nil, fmt.Errorf("missing refillCount in returned tuple")
	}

	backlogCountUntil, ok := returnTuple[2].(int64)
	if !ok {
		return nil, fmt.Errorf("missing backlogCount in returned tuple")
	}

	backlogCountTotal, ok := returnTuple[3].(int64)
	if !ok {
		return nil, fmt.Errorf("missing backlogCount in returned tuple")
	}

	capacity, ok := returnTuple[4].(int64)
	if !ok {
		return nil, fmt.Errorf("missing capacity in returned tuple")
	}

	refill, ok := returnTuple[5].(int64)
	if !ok {
		return nil, fmt.Errorf("missing refill in returned tuple")
	}

	rawRefilledItemIDs, ok := returnTuple[6].([]any)
	if !ok {
		return nil, fmt.Errorf("missing refilled item IDs in returned tuple")
	}

	refilledItemIDs := make([]string, len(rawRefilledItemIDs))
	for i, d := range rawRefilledItemIDs {
		itemID, ok := d.(string)
		if ok {
			refilledItemIDs[i] = itemID
		}
	}

	var retryAt time.Time
	retryAtMillis, ok := returnTuple[7].(int64)
	if !ok {
		return nil, fmt.Errorf("missing retryAt in returned tuple")
	}

	if retryAtMillis > nowMS {
		retryAt = time.UnixMilli(retryAtMillis)
	}

	refillResult := &BacklogRefillResult{
		Refilled:          int(refillCount),
		TotalBacklogCount: int(backlogCountTotal),
		BacklogCountUntil: int(backlogCountUntil),
		Capacity:          int(capacity),
		Refill:            int(refill),
		RefilledItems:     refilledItemIDs,
		RetryAt:           retryAt,
	}

	switch status {
	case 0:
		return refillResult, nil
	case 1:
		refillResult.Constraint = enums.QueueConstraintAccountConcurrency
		return refillResult, nil
	case 2:
		refillResult.Constraint = enums.QueueConstraintFunctionConcurrency
		return refillResult, nil
	case 3:
		refillResult.Constraint = enums.QueueConstraintCustomConcurrencyKey1
		return refillResult, nil
	case 4:
		refillResult.Constraint = enums.QueueConstraintCustomConcurrencyKey2
		return refillResult, nil
	case 5:
		refillResult.Constraint = enums.QueueConstraintThrottle
		return refillResult, nil
	default:
		return nil, fmt.Errorf("unknown status refilling backlog: %v (%T)", status, status)
	}
}

func (q *queue) BacklogRequeue(ctx context.Context, backlog *QueueBacklog, sp *QueueShadowPartition, requeueAt time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "BacklogRequeue"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for BacklogRequeue: %s", q.primaryQueueShard.Kind)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		kg.ShadowPartitionMeta(),
		kg.BacklogMeta(),
		kg.ShadowPartitionMeta(),

		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(accountID),
		kg.ShadowPartitionSet(sp.PartitionID),
		kg.BacklogSet(backlog.BacklogID),

		kg.PartitionNormalizeSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		accountID,
		sp.PartitionID,
		backlog.BacklogID,
		requeueAt.UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/backlogRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "backlogRequeue"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("could not requeue backlog: %w", err)
	}

	q.log.Trace("requeued backlog",
		"id", backlog.BacklogID,
		"partition", sp.PartitionID,
		"time", requeueAt.Format(time.StampMilli),
		"successive_throttle", backlog.SuccessiveThrottleConstrained,
		"successive_concurrency", backlog.SuccessiveCustomConcurrencyConstrained,
		"status", status,
	)

	switch status {
	case 0, 1:
		return nil
	case -1:
		return ErrBacklogNotFound
	default:
		return fmt.Errorf("unknown response requeueing backlog: %v (%T)", status, status)
	}
}

func (q *queue) BacklogPrepareNormalize(ctx context.Context, b *QueueBacklog, sp *QueueShadowPartition) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "BacklogPrepareNormalize"), redis_telemetry.ScopeQueue)

	shard := q.primaryQueueShard

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for BacklogPrepareNormalize: %s", shard.Kind)
	}
	kg := shard.RedisClient.kg

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		kg.BacklogMeta(),
		kg.ShadowPartitionMeta(),

		kg.BacklogSet(b.BacklogID),
		kg.ShadowPartitionSet(sp.PartitionID),
		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(accountID),

		kg.GlobalAccountNormalizeSet(),
		kg.AccountNormalizeSet(accountID),
		kg.PartitionNormalizeSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		b.BacklogID,
		sp.PartitionID,
		accountID,
		// order normalize by timestamp
		q.clock.Now().UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/backlogPrepareNormalize"].Exec(
		redis_telemetry.WithScriptName(ctx, "backlogPrepareNormalize"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).ToInt64()
	if err != nil {
		return fmt.Errorf("error preparing backlog normalization: %w", err)
	}

	switch status {
	case 1:
		return nil
	case -1:
		return ErrBacklogGarbageCollected
	default:
		return fmt.Errorf("unknown status preparing backlog normalization: %v (%T)", status, status)
	}
}

// BacklogPeek is the public interface to peek items from a backlog
func (q *queue) BacklogPeek(ctx context.Context, b *QueueBacklog, from time.Time, until time.Time, limit int64, opts ...PeekOpt) ([]*osqueue.QueueItem, int, error) {
	return q.backlogPeek(ctx, b, from, until, limit, opts...)
}

// backlogPeek peeks item from the given backlog.
//
// Pointers to missing items will be removed from the backlog.
func (q *queue) backlogPeek(ctx context.Context, b *QueueBacklog, from time.Time, until time.Time, limit int64, opts ...PeekOpt) ([]*osqueue.QueueItem, int, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "backlogPeek"), redis_telemetry.ScopeQueue)

	opt := peekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	if !q.isPermittedQueueKind() {
		return nil, 0, fmt.Errorf("unsupported queue shared kind for backlogPeek: %s", q.primaryQueueShard.Kind)
	}

	if b == nil {
		return nil, 0, fmt.Errorf("expected backlog to be provided")
	}

	if limit > AbsoluteQueuePeekMax || limit > q.peekMax {
		limit = q.peekMax
	}
	if limit <= 0 {
		limit = q.peekMin
	}

	var fromTime *time.Time
	if !from.IsZero() {
		fromTime = &from
	}

	l := q.log.With(
		"method", "backlogPeek",
		"backlog", b,
		"from", from,
		"until", until,
		"limit", limit,
	)

	rc := q.primaryQueueShard.RedisClient
	if opt.Shard != nil {
		rc = opt.Shard.RedisClient
	}

	backlogSet := rc.kg.BacklogSet(b.BacklogID)

	p := peeker[osqueue.QueueItem]{
		q:               q,
		opName:          "backlogPeek",
		keyMetadataHash: rc.kg.QueueItem(),
		max:             q.peekMax,
		maker: func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		handleMissingItems:     CleanupMissingPointers(ctx, backlogSet, rc.Client(), l),
		isMillisecondPrecision: true,
		fromTime:               fromTime,
	}

	res, err := p.peek(ctx, backlogSet, true, until, limit, opts...)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, 0, ErrBacklogPeekMaxExceedsLimits
		}
		return nil, 0, fmt.Errorf("error peeking backlog queue items, %w", err)
	}

	return res.Items, res.TotalCount, nil
}

// NOTE: this function only work with key queues
func (q *queue) BacklogsByPartition(ctx context.Context, queueShard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error) {
	opt := queueIterOpt{
		batchSize:                 1000,
		interval:                  50 * time.Millisecond,
		enableMillisecondIncrease: true,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	l := q.log.With(
		"method", "BacklogsByPartition",
		"partitionID", partitionID,
		"from", from,
		"until", until,
	)

	kg := queueShard.RedisClient.kg

	return func(yield func(*QueueBacklog) bool) {
		hashKey := kg.BacklogMeta()
		ptFrom := from

		for {
			var iterated int

			peeker := peeker[QueueBacklog]{
				q:                      q,
				max:                    opt.batchSize,
				opName:                 "backlogsByPartition",
				isMillisecondPrecision: true,
				handleMissingItems: func(pointers []string) error {
					// don't interfere, clean up will happen in normal processing anyways
					return nil
				},
				maker: func() *QueueBacklog {
					return &QueueBacklog{}
				},
				keyMetadataHash: hashKey,
				fromTime:        &ptFrom,
			}

			isSequential := true
			res, err := peeker.peek(ctx, kg.ShadowPartitionSet(partitionID), isSequential, until, opt.batchSize,
				WithPeekOptQueueShard(&queueShard),
			)
			if err != nil {
				l.Error("error peeking backlogs for partition", "partition_id", partitionID, "err", err)
				return
			}

			for _, bl := range res.Items {
				if bl == nil {
					continue
				}

				if !yield(bl) {
					return
				}

				iterated++
			}

			ptFrom = time.UnixMilli(res.Cursor)

			l.Trace("iterated backlogs in partition", "count", iterated)

			// didn't process anything, exit loop
			if iterated == 0 {
				break
			}

			if opt.enableMillisecondIncrease {
				// shift the starting point 1ms so it doesn't try to grab the same stuff again
				// NOTE: this could result skipping items if the previous batch of items are all on
				// the same millisecond
				ptFrom = ptFrom.Add(time.Millisecond)
			}

			// wait a little before processing the next batch
			<-time.After(opt.interval)
		}
	}, nil
}

func (q *queue) PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "partitionBacklogSize"), redis_telemetry.ScopeQueue)

	if q.queueShardClients == nil {
		return 0, nil
	}

	l := q.log.With(
		"method", "PartitionBacklogSize",
		"partition_id", partitionID,
	)

	var (
		wg    sync.WaitGroup
		count int64
	)
	until := q.clock.Now().Add(24 * time.Hour * 365) // 1y ahead

	for _, shard := range q.queueShardClients {
		shard := shard

		wg.Add(1)
		go func() {
			defer wg.Done()

			log := l.With("shard", shard.Name)

			backlogs, err := q.BacklogsByPartition(ctx, shard, partitionID, time.Time{}, until)
			if err != nil {
				log.ReportError(err, "error preparing backlog iterator")
				return
			}

			bwg := sync.WaitGroup{}
			for bl := range backlogs {
				bwg.Add(1)
				backlogID := bl.BacklogID

				go func() {
					defer bwg.Done()

					size, err := q.BacklogSize(ctx, shard, backlogID)
					if err != nil {
						log.ReportError(err, "error retrieving backlog size",
							logger.WithErrorReportTags(map[string]string{
								"backlog":   bl.BacklogID,
								"partition": bl.ShadowPartitionID,
							}),
						)
						return
					}
					atomic.AddInt64(&count, size)
				}()
			}
			bwg.Wait()
		}()
	}
	wg.Wait()

	return count, nil
}

func (q *queue) BacklogSize(ctx context.Context, queueShard QueueShard, backlogID string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "backlogSize"), redis_telemetry.ScopeQueue)

	rc := queueShard.RedisClient.Client()
	cmd := rc.B().Zcard().Key(queueShard.RedisClient.kg.BacklogSet(backlogID)).Build()
	count, err := rc.Do(ctx, cmd).AsInt64()
	if rueidis.IsRedisNil(err) {
		return 0, nil
	}
	return count, err
}

func shuffleBacklogs(b []*QueueBacklog) []*QueueBacklog {
	weights := make([]float64, len(b))
	for i, backlog := range b {
		if backlog.Start {
			weights[i] = 1.0
		} else {
			weights[i] = 10.0
		}
	}

	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*QueueBacklog, len(b))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return b
		}
		result[n] = b[idx]
	}

	return result
}
