package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

const (
	// KindStart represents a queue state that the function state has been created but not started yet.
	// Essentially a status that represents the backlog.
	KindStart         = "start"
	KindEdge          = "edge"
	KindSleep         = "sleep"
	KindPause         = "pause"
	KindDebounce      = "debounce"
	KindScheduleBatch = "schedule-batch"
	KindEdgeError     = "edge-error" // KindEdgeError is used to indicate a final step error attempting a graceful save.
	KindQueueMigrate  = "queue-migrate"
)

type jobIDValType struct{}

var (
	jobCtxVal = jobIDValType{}
)

// WithJobID returns a context that stores the given job ID inside.
func WithJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobCtxVal, jobID)
}

// JobIDFromContext returns the job ID given the current context, or an
// empty string if there's no job ID.
func JobIDFromContext(ctx context.Context) string {
	str, _ := ctx.Value(jobCtxVal).(string)
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
}

func (q *QueueItem) SetID(ctx context.Context, str string) {
	q.ID = HashID(ctx, str)
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

	// If this is not a start/simple edge/edge error, we can ignore this.
	if (q.Data.Kind != KindStart &&
		q.Data.Kind != KindEdge &&
		q.Data.Kind != KindEdgeError) || q.Data.Attempt > 0 {
		return q.AtMS
	}

	// If this is > 2 seconds in the future, don't mess with the time.
	// This prevents any accidental fudging of future run times, even if the
	// kind is edge (which should never exist... but, better to be safe).
	if q.AtMS > now.Add(consts.FutureAtLimit).UnixMilli() {
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
	Metadata map[string]string `json:"metadata,omitempty"`
	// QueueName allows control over the queue name.  If not provided, this falls
	// back to the queue mapping defined on the queue or the workflow ID of the fn.
	QueueName *string `json:"qn,omitempty"`
	// RunInfo shows additional runtime information for the item like delays.
	RunInfo *RunInfo `json:"runinfo,omitempty"`
	// Throttle represents GCRA rate limiting for the queue item, which is applied when
	// attempting to lease the item from the queue.
	Throttle *Throttle `json:"throttle,omitempty"`
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

func (i Item) GetMaxAttempts() int {
	if i.MaxAttempts == nil {
		return consts.DefaultRetryCount
	}
	return *i.MaxAttempts
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
		Metadata              map[string]string         `json:"metadata"`
		QueueName             *string                   `json:"qn,omitempty"`
		RunInfo               *RunInfo                  `json:"runinfo,omitempty"`
		Throttle              *Throttle                 `json:"throttle"`
		CustomConcurrencyKeys []state.CustomConcurrency `json:"cck,omitempty"`
		PriorityFactor        *int64                    `json:"pf,omitempty"`
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
	i.RunInfo = temp.RunInfo
	i.CustomConcurrencyKeys = temp.CustomConcurrencyKeys
	i.PriorityFactor = temp.PriorityFactor
	i.QueueName = temp.QueueName

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

// PayloadPauseTimeout is the payload stored when enqueueing a pause timeout, eg.
// a future task to check whether an event has been received yet.
//
// This is always enqueued from any async match;  we must correctly decrement the
// pending count in cases where the event is not received.
type PayloadPauseTimeout struct {
	PauseID   uuid.UUID `json:"pauseID"`
	OnTimeout bool      `json:"onTimeout"`
}

func HashID(_ context.Context, id string) string {
	ui := xxhash.Sum64String(id)
	return strconv.FormatUint(ui, 36)
}
