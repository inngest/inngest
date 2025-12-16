package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/util"
	"github.com/xhit/go-str2duration/v2"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

type jobIDValType struct{ int }

var (
	jobCtxVal   = jobIDValType{0}
	shardCtxVal = jobIDValType{1}
)

// WithJobID returns a context that stores the given job ID inside.
func WithJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobCtxVal, jobID)
}

// WithShardID returns a context that stores the shard ID for the current job.
func WithShardID(ctx context.Context, shardID string) context.Context {
	return context.WithValue(ctx, shardCtxVal, shardID)
}

// JobIDFromContext returns the job ID given the current context, or an
// empty string if there's no job ID.
func JobIDFromContext(ctx context.Context) string {
	str, _ := ctx.Value(jobCtxVal).(string)
	return str
}

func ShardIDFromContext(ctx context.Context) string {
	str, _ := ctx.Value(shardCtxVal).(string)
	return str
}

// QueueItem represents an individually queued work scheduled for some time in the
// future.
type QueueItem struct {
	// ID represents a unique identifier for the queue item.  This can be any
	// unique string and will be hashed.  Using the same ID provides idempotency
	// guarantees within the queue's IdempotencyTTL.
	ID string `json:"id"`
	// EarliestPeekTime stores the earliest time that the job was peeked as a
	// millisecond epoch timestamp.
	//
	// This lets us easily track sojourn latency.
	EarliestPeekTime int64 `json:"pt,omitempty"`
	// AtMS represents the score for the queue item - usually, the current time
	// that this QueueItem needs to be executed at, as a millisecond epoch.
	//
	// Note that due to priority factors and function FIFO manipulation, if we're
	// scheduling a job to run at `Now()` AtMS may be a time in the past to bump
	// the item in the queue.
	//
	// This is necessary for rescoring partitions and checking latencies.
	AtMS int64 `json:"at"`

	// WallTimeMS represents the actual wall time in which the job should run, used to
	// check latencies.  This is NOT used for scoring or ordering and is for internal
	// accounting only.
	//
	// This is set when enqueueing or requeueing a job.
	WallTimeMS int64 `json:"wt"`

	// FunctionID is the workflow ID that this job belongs to.
	FunctionID uuid.UUID `json:"wfID"`
	// WorkspaceID is the workspace that this job belongs to.
	WorkspaceID uuid.UUID `json:"wsID"`
	// LeaseID is a ULID which embeds a timestamp denoting when the lease expires.
	LeaseID *ulid.ULID `json:"leaseID,omitempty"`
	// Data represents the enqueued data, eg. the edge to process or the pause
	// to resume.
	Data Item `json:"data"`
	// QueueName allows placing this job into a specific queue name. This is exclusively
	// used for system-specific queues for handling pauses, recovery, and other features.
	// If unset, the workflow-specific partitions for key queues will be used.
	//
	// This should almost always be nil.
	QueueName *string `json:"queueID,omitempty"`
	// IdempotencyPerioud allows customizing the idempotency period for this queue
	// item.  For example, after a debounce queue has been consumed we want to remove
	// the idempotency key immediately;  the same debounce key should become available
	// for another debounced function run.
	IdempotencyPeriod *time.Duration `json:"ip,omitempty"`

	// Refilled from backlog ID, set if item was refilled from backlog. Removed during requeue to partition.
	RefilledFrom string `json:"rf,omitempty"`
	RefilledAt   int64  `json:"rat,omitempty"`

	// EnqueuedAt tracks the unix timestamp of enqueueing the queue item (to the backlog or directly to
	// the partition). This is not the same as AtMS for items scheduled in the future or past.
	EnqueuedAt int64 `json:"eat"`

	// CapacityLease is the optional capacity lease for this queue item.
	// This is set when the Constraint API feature flag is enabled and the item was refilled.
	CapacityLease *CapacityLease `json:"cl,omitempty"`
}

type CapacityLease struct {
	LeaseID ulid.ULID `json:"l,omitempty"`
}

func (q *QueueItem) SetID(ctx context.Context, str string) {
	q.ID = HashID(ctx, str)
}

// IsPromotableScore returns whether a score can be fudged.
func (q QueueItem) IsPromotableScore() bool {
	switch q.Data.Kind {
	case KindStart, KindSleep, KindEdge, KindPause, KindEdgeError:
		// All user jobs can be fudged.
		return true
	}
	return false
}

// RequiresPromotion returns true if the score needs future promotion (fudging the numbers!)
// This is the case when a workflow job is enqueued in the future:
//
// - Workflow T0 runs a job which retries at T10
// - 1,000,000 steps are enqueued for other workflows at T5
// - At T10...
//   - In order to run workflow T0's retry, we have to complete all 1M jobs from
//     other *later* workflows before attempting the retry, meaning that we do not
//     run older run's jobs before newer runs jobs.
//
// Future fudging allows us to reschedule jobs at an aprpoproate time through some other
// queueing means.
func (q QueueItem) RequiresPromotionJob(now time.Time) bool {
	if !q.IsPromotableScore() {
		// If this doesn't have fudging enabled, ignore.
		return false
	}

	if now.IsZero() {
		now = time.Now()
	}

	// If this is > 2 seconds in the future, don't mess with the time.
	// This prevents any accidental fudging of future run times, even if the
	// kind is edge (which should never exist... but, better to be safe).
	if q.AtMS > now.Add(consts.FutureAtLimit).UnixMilli() {
		return true
	}

	return false
}

// Score returns the score (time that the item should run) for the queue item.
//
// NOTE: In order to prioritize finishing older function runs with a busy function
// queue, we sometimes use the function run's "started at" time to enqueue edges which
// run steps.  This lets us push older function steps to the beginning of the queue,
// ensuring they run before other newer function runs.
//
// We can ONLY do this for the first attempt, and we can ONLY do this for edges that
// are not sleeps (eg. immediate runs)
func (q QueueItem) Score(now time.Time) int64 {
	if now.IsZero() {
		now = time.Now()
	}

	if !q.IsPromotableScore() || q.RequiresPromotionJob(now) {
		return q.AtMS
	}

	// Get the time for the function, based off of the run ID.
	startAt := int64(q.Data.Identifier.RunID.Time())

	if startAt == 0 {
		return q.AtMS
	}

	// Remove the PriorityFactor from the time to push higher priority work
	// earlier.
	return startAt - q.Data.GetPriorityFactor()
}

func (q QueueItem) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

// IsLeased checks if the QueueItem is currently already leased or not
// based on the time passed in.
func (q QueueItem) IsLeased(time time.Time) bool {
	return q.LeaseID != nil && ulid.Time(q.LeaseID.Time()).After(time)
}

// SojournLatency is the delay due to concurrency limits, throttle, or other user-defined concurrency.
func (q QueueItem) SojournLatency(now time.Time) time.Duration {
	if q.RefilledAt == 0 {
		var sojourn time.Duration
		if q.EarliestPeekTime > 0 {
			sojourn = now.Sub(time.UnixMilli(q.EarliestPeekTime))
		}

		return sojourn
	}

	// Track the entire time from enqueueing an item to refilling, including
	// expected static (item in the future) and dynamic (time spent waiting due to concurrency limits) delays.
	// note: System delays may be included in this.
	return q.RefillDelay() + q.ExpectedDelay()
}

// Latency represents the processing delay excluding sojourn latency.
func (q QueueItem) Latency(now time.Time) time.Duration {
	if q.RefilledAt == 0 {
		sojourn := q.SojournLatency(now)

		return now.Sub(time.UnixMilli(q.WallTimeMS)) - sojourn
	}

	// Time between refill and lease/processing
	return q.LeaseDelay(now)
}

// ExpectedDelay returns the expected delay for a queue item (usually 0, positive if scheduled into the future).
// This is based on static information and thus does _not_ capture the time spent waiting due to concurrency constraints, etc.
func (q QueueItem) ExpectedDelay() time.Duration {
	if q.EnqueuedAt == 0 {
		return 0
	}

	delayMS := q.AtMS - q.EnqueuedAt
	delayMS = int64(math.Max(float64(delayMS), 0)) // ignore negative delays (item was planned earlier than enqueued)
	itemDelay := time.Duration(delayMS) * time.Millisecond

	return itemDelay
}

// RefillDelay returns the time it took from enqueueing to refilling (minus expected delays)
func (q QueueItem) RefillDelay() time.Duration {
	if q.RefilledAt == 0 || q.EnqueuedAt == 0 {
		return 0
	}
	refilledAt := time.UnixMilli(q.RefilledAt)
	enqueuedAt := time.UnixMilli(q.EnqueuedAt)

	refillDelay := refilledAt.Sub(enqueuedAt)

	// ignore expected delay (if item was scheduled in the future)
	// note: this does not account for time spent waiting due to hitting concurrency limits, etc.
	refillDelay = refillDelay - q.ExpectedDelay()

	return refillDelay
}

// LeaseDelay returns the time between refilling and leasing
func (q QueueItem) LeaseDelay(now time.Time) time.Duration {
	if q.RefilledAt == 0 {
		return 0
	}

	return now.Sub(time.UnixMilli(q.RefilledAt))
}

// Item represents an item stored within a queue.
//
// Note that each individual implementation may wrap this to add their own fields,
// such as a job identifier.
//
// TODO: Refactor this with the QueueItem in redis state to remove duplicates.
type Item struct {
	// JobID is an internal ID used to deduplicate queue items.
	JobID *string `json:"-"`
	// GroupID allows tracking step history across many jobs;  if a step is scheduled,
	// then runs and fails, it's rescheduled.  We want the same group ID to be stored
	// across the lifetime of a step so that we can correlate all history entries across
	// a specific step.
	GroupID string `json:"groupID,omitempty"`
	// Workspace is the ID that this workspace job belongs to
	WorkspaceID uuid.UUID `json:"wsID"`
	// Kind represents the job type and payload kind stored within Payload.
	Kind string `json:"kind"`
	// Identifier represents the unique workflow ID and run ID for the current job.
	Identifier state.Identifier `json:"identifier"`

	// Attempt stores the zero index attempt counter
	Attempt int `json:"atts"`
	// MaxAttempts is the maximum number of attempts we can retry.  When attempts == this,
	// do not schedule another try.  If nil, use queue.DefaultRetryCount.
	MaxAttempts *int `json:"maxAtts,omitempty"`
	// Payload stores item-specific data for use when processing the item.  For example,
	// this may contain the function's edge for running a step.
	Payload any `json:"payload,omitempty"`
	// Metadata is used for storing additional metadata related to the queue item.
	// e.g. tracing data
	Metadata map[string]any `json:"metadata,omitempty"`
	// QueueName allows control over the queue name.  If not provided, this falls
	// back to the queue mapping defined on the queue or the workflow ID of the fn.
	QueueName *string `json:"qn,omitempty"`
	// RunInfo shows additional runtime information for the item like delays.
	RunInfo *RunInfo `json:"runinfo,omitempty"`
	// Throttle represents GCRA rate limiting for the queue item, which is applied when
	// attempting to lease the item from the queue.
	Throttle *Throttle `json:"throttle,omitempty"`
	// Singleton represents a singleton key for the queue item, which is used to
	// not allow multiple start items to be scheduled at the same time.
	Singleton *Singleton `json:"singleton,omitempty"`
	// CustomConcurrencyKeys stores custom concurrency keys for this function run.  This
	// allows us to use custom concurrency keys for each job when processing steps for
	// the function, with cached expression results.
	//
	// NOTE: This was added as Identifier is being deprecated as of 2024-04-09.  Items added
	// to the queue prior to this date may have item.Identifier.CustomConcurrencyKeys added.
	CustomConcurrencyKeys []state.CustomConcurrency `json:"cck,omitempty"`
	// PriorityFactor is the overall priority factor for this particular function
	// run.  This allows individual runs to take precedence within the same queue.
	// The higher the number (up to consts.PriorityFactorMax), the higher priority
	// this run has.  All next steps will use this as the factor when scheduling
	// future edge jobs (on their first attempt).
	PriorityFactor *int64 `json:"pf,omitempty"`

	// ParallelMode controls discovery step scheduling after a parallel step
	// ends
	ParallelMode enums.ParallelMode `json:"pm,omitempty"`
}

func (i Item) GetMaxAttempts() int {
	if i.MaxAttempts == nil {
		return consts.DefaultRetryCount
	}
	return *i.MaxAttempts
}

type Throttle struct {
	// Key is the unique throttling key that's used to group queue items when
	// processing rate limiting/throttling.
	Key string `json:"k"`
	// Limit is the actual rate limit
	Limit int `json:"l"`
	// Burst is the busrsable capacity of the rate limit
	Burst int `json:"b"`
	// Period is the rate limit period, in seconds
	Period int `json:"p"`

	// UnhashedThrottleKey is the raw value returned after evaluating the key expression, if configured.
	// Otherwise, this is the function ID. In the case of evaluated keys, this may be large and should be truncated before usage.
	UnhashedThrottleKey string `json:"-"`

	KeyExpressionHash string `json:"keh"`
}

type Singleton struct {
	// Key is the unique singleton key that's used to group queue items when
	// processing singleton items.
	Key string `json:"k"`

	// Mode defines the behavior when a new singleton run is queued while another is active.
	// It determines whether to skip the new run or cancel the current one and replace it.
	Mode enums.SingletonMode `json:"m"`
}

// SpanID generates a spanID based on the combination the jobID and attempt
func (i Item) SpanID() (*trace.SpanID, error) {
	if i.JobID == nil {
		return nil, fmt.Errorf("no job ID for item")
	}

	data := map[string]any{
		"id":      *i.JobID,
		"attempt": i.Attempt,
	}
	byt, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	hash := xxhash.New()
	_, _ = hash.Write(byt)
	sum := hash.Sum(nil)
	spanID := trace.SpanID(sum[:8])
	return &spanID, nil
}

func (i Item) GetConcurrencyKeys() []state.CustomConcurrency {
	if len(i.Identifier.CustomConcurrencyKeys) > 0 {
		// Only use this if specified.
		return i.Identifier.CustomConcurrencyKeys
	}
	return i.CustomConcurrencyKeys
}

// GetPriorityFactor returns the priority factor for the queue item.  This fudges the job item's
// visibility time on enqueue, allowing fair prioritization.
//
// For example, a job with a PriorityFactor of 100 will be inserted 100 seconds prior to the job's
// actual RunAt time.  This pushes the job ahead of other work, except for work older than 100 seconds.
//
// Therefore, when two jobs are enqueued at the same time with differeng factors the job with the higher
// factor will always run first (without a queue backlog).
//
// Note: the returned time is the factor in milliseconds.
func (i Item) GetPriorityFactor() int64 {
	switch i.Kind {
	case KindStart, KindEdge, KindEdgeError:
		if i.PriorityFactor != nil {
			// This takes precedence.
			return int64(*i.PriorityFactor * 1000)
		}
		// Only support edges right now.  We don't account for the factor on other queue entries,
		// else eg. sleeps would wake up at the wrong time.
		if i.Identifier.PriorityFactor != nil {
			return int64(*i.Identifier.PriorityFactor * 1000)
		}
	}
	return 0
}

// IsStepKind determines if the item is considered a step
func (i Item) IsStepKind() bool {
	return i.Kind == KindStart || i.Kind == KindEdge || i.Kind == KindSleep || i.Kind == KindEdgeError
}

func (i *Item) UnmarshalJSON(b []byte) error {
	type kind struct {
		GroupID               string                    `json:"groupID"`
		WorkspaceID           uuid.UUID                 `json:"wsID"`
		Kind                  string                    `json:"kind"`
		Identifier            state.Identifier          `json:"identifier"`
		Attempt               int                       `json:"atts"`
		MaxAttempts           *int                      `json:"maxAtts,omitempty"`
		Payload               json.RawMessage           `json:"payload"`
		Metadata              map[string]any            `json:"metadata"`
		QueueName             *string                   `json:"qn,omitempty"`
		RunInfo               *RunInfo                  `json:"runinfo,omitempty"`
		Throttle              *Throttle                 `json:"throttle"`
		Singleton             *Singleton                `json:"singleton"`
		CustomConcurrencyKeys []state.CustomConcurrency `json:"cck,omitempty"`
		PriorityFactor        *int64                    `json:"pf,omitempty"`
		ParallelMode          enums.ParallelMode        `json:"pm,omitempty"`
	}
	temp := &kind{}
	err := json.Unmarshal(b, temp)
	if err != nil {
		return fmt.Errorf("error unmarshalling queue item: %w", err)
	}

	i.GroupID = temp.GroupID
	i.WorkspaceID = temp.WorkspaceID
	i.Kind = temp.Kind
	i.Identifier = temp.Identifier
	i.Attempt = temp.Attempt
	i.MaxAttempts = temp.MaxAttempts
	i.Metadata = temp.Metadata
	i.Throttle = temp.Throttle
	i.Singleton = temp.Singleton
	i.RunInfo = temp.RunInfo
	i.CustomConcurrencyKeys = temp.CustomConcurrencyKeys
	i.PriorityFactor = temp.PriorityFactor
	i.QueueName = temp.QueueName
	i.ParallelMode = temp.ParallelMode

	// Save this for custom unmarshalling of other jobs.  This is overwritten
	// for known queue kinds.
	if len(temp.Payload) > 0 {
		i.Payload = temp.Payload
	}

	switch temp.Kind {
	case KindStart, KindEdge, KindSleep, KindEdgeError:
		// Edge and Sleep are the same;  the only difference is that the executor
		// runner should always save nil to the state store using the outgoing edge's
		// ID when processing a sleep so that the state + stack are updated properly.
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadEdge{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindPause:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadPauseTimeout{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindPauseBlockFlush:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadPauseBlockFlush{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindJobPromote:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadJobPromote{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindFunctionPause:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadPauseFunction{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindFunctionUnpause:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadUnpauseFunction{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	}
	return nil
}

// GetEdge returns the edge from the enqueued item, if the payload is of type PayloadEdge.
func GetEdge(i Item) (*PayloadEdge, error) {
	switch v := i.Payload.(type) {
	case PayloadEdge:
		return &v, nil
	default:
		return nil, fmt.Errorf("unable to get edge from payload type: %T", v)
	}
}

// PayloadEdge is the payload stored when enqueueing an edge traversal to execute
// the incoming step of the edge.
type PayloadEdge struct {
	Edge inngest.Edge `json:"edge"`
}

type PayloadPauseBlockFlush struct {
	EventName string `json:"e"`
}

type PayloadJobPromote struct {
	PromoteJobID string `json:"sjid"`
	ScheduledAt  int64  `json:"su"`
}

// PayloadPauseTimeout is the payload stored when enqueueing a pause timeout, eg.
// a future task to check whether an event has been received yet.
//
// This is always enqueued from any async match;  we must correctly decrement the
// pending count in cases where the event is not received.
type PayloadPauseTimeout struct {
	// PauseID is the ID of the pause that the timeout job will resume.  This has
	// existed since the beginning of Inngest, and is included for backcompat for
	// future jobs.
	PauseID uuid.UUID `json:"pauseID"`
	// Pause is the full pause struct for the timeout job.  Note that the identifier
	// should not exist in the pause, as it already exists in the queue item.
	Pause state.Pause `json:"pause"`
}

// PayloadPauseFunction represents the queue item payload for the internal system queue for
// pausing functions reliably. The IDs are retrieved from the identifier.
type PayloadPauseFunction struct {
	// PausedAt represents the unix timestamp in milliseconds when the user requested to pause the function.
	PausedAt int64 `json:"pat"`

	// CancelRunningImmediately determines whether pending jobs should be cancelled immediately or after a set duration.
	CancelRunningImmediately bool `json:"cri,omitempty"`
}

// PayloadUnpauseFunction represents the queue item payload for the internal system queue for
// unpausing functions reliably. The IDs are retrieved from the identifier.
type PayloadUnpauseFunction struct {
	// PausedAt represents the unix timestamp in milliseconds when the user originally requested to pause the function.
	// This is included in the unpause job to create a consistent identifier for pause periods and make unpausing idempotent.
	PausedAt int64 `json:"pat"`
	// UnpausedAt represents the unix timestamp in milliseconds when the user requested to unpause the function.
	UnpausedAt int64 `json:"upat"`
}

func HashID(_ context.Context, id string) string {
	ui := xxhash.Sum64String(id)
	return strconv.FormatUint(ui, 36)
}

func GetThrottleConfig(ctx context.Context, fnID uuid.UUID, throttle *inngest.Throttle, evtMap map[string]any) *Throttle {
	if throttle == nil {
		return nil
	}

	unhashedThrottleKey := fnID.String()
	throttleKey := HashID(ctx, unhashedThrottleKey)
	var throttleExpr string
	if throttle.Key != nil {
		val, _ := expressions.Evaluate(ctx, *throttle.Key, map[string]any{
			"event": evtMap,
		})
		unhashedThrottleKey = fmt.Sprintf("%v", val)
		throttleKey = throttleKey + "-" + HashID(ctx, unhashedThrottleKey)
		throttleExpr = *throttle.Key
	}

	return &Throttle{
		Key:                 throttleKey,
		Limit:               int(throttle.Limit),
		Burst:               int(throttle.Burst),
		Period:              int(throttle.Period.Seconds()),
		UnhashedThrottleKey: unhashedThrottleKey,
		KeyExpressionHash:   util.XXHash(throttleExpr),
	}
}

func GetCustomConcurrencyKeys(ctx context.Context, id sv2.ID, customConcurrency []inngest.Concurrency, evtMap map[string]any) []state.CustomConcurrency {
	if len(customConcurrency) == 0 {
		return nil
	}

	var keys []state.CustomConcurrency

	// Ensure we evaluate concurrency keys when scheduling the function.
	for _, limit := range customConcurrency {
		if !limit.IsCustomLimit() {
			continue
		}

		// Ensure we bind the limit to the correct scope.
		scopeID := id.FunctionID
		switch limit.Scope {
		case enums.ConcurrencyScopeAccount:
			scopeID = id.Tenant.AccountID
		case enums.ConcurrencyScopeEnv:
			scopeID = id.Tenant.EnvID
		}

		evaluated := limit.Evaluate(ctx, evtMap)
		key := util.ConcurrencyKey(limit.Scope, scopeID, evaluated)

		// Store the concurrency limit in the function.  By copying in the raw expression hash,
		// we can update the concurrency limits for in-progress runs as new function versions
		// are stored.
		//
		// The raw keys are stored in the function state so that we don't need to re-evaluate
		// keys and input each time, as they're constant through the function run.
		keys = append(
			keys,
			sv2.CustomConcurrency{
				Key:                       key,
				Hash:                      limit.Hash,
				Limit:                     limit.Limit,
				UnhashedEvaluatedKeyValue: evaluated,
			},
		)
	}

	return keys
}

func ConvertToConstraintConfiguration(accountConcurrency int, fn inngest.Function) (constraintapi.ConstraintConfig, error) {
	var rateLimit []constraintapi.RateLimitConfig
	if fn.RateLimit != nil {
		var rateLimitKey string
		if fn.RateLimit.Key != nil {
			rateLimitKey = *fn.RateLimit.Key
		}

		dur, err := str2duration.ParseDuration(fn.RateLimit.Period)
		if err != nil {
			return constraintapi.ConstraintConfig{}, fmt.Errorf("invalid rate limit period: %w", err)
		}

		rateLimit = append(rateLimit, constraintapi.RateLimitConfig{
			Scope:             enums.RateLimitScopeFn,
			Limit:             int(fn.RateLimit.Limit),
			Period:            int(dur.Seconds()),
			KeyExpressionHash: util.XXHash(rateLimitKey),
		})
	}

	var customConcurrency []constraintapi.CustomConcurrencyLimit
	if fn.Concurrency != nil {
		for _, c := range fn.Concurrency.Limits {
			if !c.IsCustomLimit() {
				continue
			}

			customConcurrency = append(customConcurrency, constraintapi.CustomConcurrencyLimit{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             c.Scope,
				Limit:             c.Limit,
				KeyExpressionHash: c.Hash,
			})
		}
	}

	var throttles []constraintapi.ThrottleConfig
	if fn.Throttle != nil {
		var throttleKey string
		if fn.Throttle.Key != nil {
			throttleKey = *fn.Throttle.Key
		}

		throttles = append(throttles, constraintapi.ThrottleConfig{
			Limit:             int(fn.Throttle.Limit),
			Burst:             int(fn.Throttle.Burst),
			Period:            int(fn.Throttle.Period.Seconds()),
			Scope:             enums.ThrottleScopeFn,
			KeyExpressionHash: util.XXHash(throttleKey),
		})
	}

	functionConcurrency := 0
	if fn.Concurrency != nil {
		functionConcurrency = fn.Concurrency.PartitionConcurrency()
	}

	return constraintapi.ConstraintConfig{
		FunctionVersion: fn.FunctionVersion,
		RateLimit:       rateLimit,
		Concurrency: constraintapi.ConcurrencyConfig{
			AccountConcurrency:    accountConcurrency,
			FunctionConcurrency:   functionConcurrency,
			CustomConcurrencyKeys: customConcurrency,
		},
		Throttle: throttles,
	}, nil
}
