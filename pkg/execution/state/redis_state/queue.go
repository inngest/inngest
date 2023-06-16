package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/execution/concurrency"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"github.com/rueian/rueidis"
	"github.com/uber-go/tally/v4"
	"go.opentelemetry.io/otel/trace"
	"gonum.org/v1/gonum/stat/sampleuv"
	"lukechampine.com/frand"
)

const (
	PartitionSelectionMax int64 = 35
	PartitionPeekMax      int64 = PartitionSelectionMax * 3

	// PartitionLeaseDuration dictates how long a worker holds the lease for
	// a partition.  This gives the worker a right to scan all queue items
	// for that partition to schedule the execution of jobs.
	//
	// Right now, this must be short enough to reduce contention but long enough
	// to account for the latency of peeking QueuePeekMax jobs from Redis.
	PartitionLeaseDuration = 4 * time.Second
	// PartitionRequeueExtension is the length of time that we extend a partition's
	// vesting time when requeueing by default.
	PartitionRequeueExtension = 30 * time.Second

	// PartitionConcurrencyLimitRequeueExtension is the length of time that a partition
	// is requeued if there is no global or partition(function) capacity because of
	// concurrency limits.
	//
	// This is short, as there are still functions that are due to run (ie vesting < now),
	// but long enough to reduce thrash.
	//
	// This means that jobs not started because of concurrency limits incur up to this amount
	// of additional latency.
	PartitionConcurrencyLimitRequeueExtension = time.Second * 2

	QueueSelectionMax   int64 = 50
	QueuePeekMax        int64 = 1000
	QueuePeekDefault    int64 = QueueSelectionMax * 3
	QueueLeaseDuration        = 10 * time.Second
	ConfigLeaseDuration       = 10 * time.Second
	ConfigLeaseMax            = 20 * time.Second

	PriorityMax     uint = 0
	PriorityDefault uint = 5
	PriorityMin     uint = 9

	// FunctionStartScoreBufferTime is the grace period used to compare function start
	// times to edg enqueue times.
	FunctionStartScoreBufferTime = 10 * time.Second

	defaultNumWorkers           = 100
	defaultPollTick             = 10 * time.Millisecond
	defaultIdempotencyTTL       = 12 * time.Hour
	defaultPartitionConcurrency = 100 // TODO: add function to override.
)

var (
	ErrQueueItemExists               = fmt.Errorf("queue item already exists")
	ErrQueueItemNotFound             = fmt.Errorf("queue item not found")
	ErrQueueItemAlreadyLeased        = fmt.Errorf("queue item already leased")
	ErrQueueItemLeaseMismatch        = fmt.Errorf("item lease does not match")
	ErrQueueItemNotLeased            = fmt.Errorf("queue item is not leased")
	ErrQueuePeekMaxExceedsLimits     = fmt.Errorf("peek exceeded the maximum limit of %d", QueuePeekMax)
	ErrPriorityTooLow                = fmt.Errorf("priority is too low")
	ErrPriorityTooHigh               = fmt.Errorf("priority is too high")
	ErrWeightedSampleRead            = fmt.Errorf("error reading from weighted sample")
	ErrPartitionNotFound             = fmt.Errorf("partition not found")
	ErrPartitionAlreadyLeased        = fmt.Errorf("partition already leased")
	ErrPartitionPeekMaxExceedsLimits = fmt.Errorf("peek exceeded the maximum limit of %d", PartitionPeekMax)
	ErrPartitionGarbageCollected     = fmt.Errorf("partition garbage collected")
	ErrConfigAlreadyLeased           = fmt.Errorf("config scanner already leased")
	ErrConfigLeaseExceedsLimits      = fmt.Errorf("config lease duration exceeds the maximum of %d seconds", int(ConfigLeaseMax.Seconds()))
	ErrPartitionConcurrencyLimit     = fmt.Errorf("At partition concurrency limit")
	ErrConcurrencyLimit              = fmt.Errorf("At concurrency limit")
)

var (
	rnd *frandRNG
)

func init() {
	// For weighted shuffles generate a new rand.
	rnd = &frandRNG{RNG: frand.New(), lock: &sync.Mutex{}}
}

// PriorityFinder returns the priority for a given queue item.
type PriorityFinder func(ctx context.Context, item QueueItem) uint

type QueueOpt func(q *queue)

func WithName(name string) func(q *queue) {
	return func(q *queue) {
		q.name = name
	}
}

func WithMetricsScope(scope tally.Scope) func(q *queue) {
	return func(q *queue) {
		q.scope = scope
	}
}

func WithPriorityFinder(pf PriorityFinder) func(q *queue) {
	return func(q *queue) {
		q.pf = pf
	}
}

func WithQueueKeyGenerator(kg QueueKeyGenerator) func(q *queue) {
	return func(q *queue) {
		q.kg = kg
	}
}

func WithIdempotencyTTL(t time.Duration) func(q *queue) {
	return func(q *queue) {
		q.idempotencyTTL = t
	}
}

// WithIdempotencyTTLFunc returns custom idempotecy durations given a QueueItem.
// This allows customization of the idempotency TTL based off of specific jobs.
func WithIdempotencyTTLFunc(f func(context.Context, QueueItem) time.Duration) func(q *queue) {
	return func(q *queue) {
		q.idempotencyTTLFunc = f
	}
}

func WithNumWorkers(n int32) func(q *queue) {
	return func(q *queue) {
		q.numWorkers = n
	}
}

// WithPollTick specifies the interval at which the queue will poll the backing store
// for available partitions.
func WithPollTick(t time.Duration) func(q *queue) {
	return func(q *queue) {
		q.pollTick = t
	}
}

func WithTracer(t trace.Tracer) func(q *queue) {
	return func(q *queue) {
		q.tracer = t
	}
}

// WithDenyQueueNames specifies that the worker cannot select jobs from queue partitions
// within the given list of names.  This means that the worker will never work on jobs
// in the specified queues.
//
// NOTE: If this is set and this worker claims the sequential lease, there is no guarantee
// on latency or fairness in the denied queue partitions.
func WithDenyQueueNames(queues ...string) func(q *queue) {
	return func(q *queue) {
		q.denyQueues = queues
		q.denyQueueMap = make(map[string]*struct{})
		for _, i := range queues {
			q.denyQueueMap[i] = &struct{}{}
		}
	}
}

// WithAllowQueueNames specifies that the worker can only select jobs from queue partitions
// within the given list of names.  This means that the worker will never work on jobs in
// other queues.
func WithAllowQueueNames(queues ...string) func(q *queue) {
	return func(q *queue) {
		q.allowQueues = queues
		q.allowQueueMap = make(map[string]*struct{})
		for _, i := range queues {
			q.allowQueueMap[i] = &struct{}{}
		}
	}
}

// WithKindToQueueMapping maps queue.Item.Kind strings to queue names.  For example,
// when pushing a queue.Item with a kind of PayloadEdge, this job can be mapped to
// a specific queue name here.
//
// The mapping must be provided in terms of item kind to queue name.  If the item
// kind doesn't exist in the mapping the job's queue name will be left nil.  This
// means that the item will be placed in the workflow ID's queue.
func WithKindToQueueMapping(mapping map[string]string) func(q *queue) {
	// XXX: Refactor osqueue.Item and this package to resolve these interfaces
	// and clean up this function.
	return func(q *queue) {
		q.queueKindMapping = mapping
	}
}

func WithLogger(l *zerolog.Logger) func(q *queue) {
	return func(q *queue) {
		q.logger = l
	}
}

// WithCustomConcurrencyKeyGenerator assigns a function that returns concurrency keys
// for a given queue item, eg. a step in a function.
func WithCustomConcurrencyKeyGenerator(f QueueItemConcurrencyKeyGenerator) func(q *queue) {
	return func(q *queue) {
		q.customConcurrencyGen = f
	}
}

// WithPartitionConcurrencyKeyGenerator assigns a function that returns concurrency keys
// for a given partition.
func WithPartitionConcurrencyKeyGenerator(f PartitionConcurrencyKeyGenerator) func(q *queue) {
	return func(q *queue) {
		q.partitionConcurrencyGen = f
	}
}

func WithAccountConcurrencyKeyGenerator(f QueueItemConcurrencyKeyGenerator) func(q *queue) {
	return func(q *queue) {
		q.accountConcurrencyGen = f
	}
}

func WithConcurrencyService(s concurrency.ConcurrencyService) func(q *queue) {
	return func(q *queue) {
		q.concurrencyService = s
	}
}

// QueueItemConcurrencyKeyGenerator returns concurrenc keys given a queue item to limits.
//
// Each queue item can have its own concurrency keys.  For example, you can define
// concurrency limits for steps within a function.  This ensures that there will never be
// more than N concurrent items running at once.
type QueueItemConcurrencyKeyGenerator func(ctx context.Context, i QueueItem) (string, int)

// PartitionConcurrencyKeyGenerator returns a concurrency key and limit for a given partition
// (function).
//
// This allows partitions (read: functions) to set their own concurrency limits.
type PartitionConcurrencyKeyGenerator func(ctx context.Context, p QueuePartition) (string, int)

func NewQueue(r rueidis.Client, opts ...QueueOpt) *queue {
	q := &queue{
		r: r,
		pf: func(ctx context.Context, item QueueItem) uint {
			return PriorityDefault
		},
		kg:                 defaultQueueKey,
		numWorkers:         defaultNumWorkers,
		wg:                 &sync.WaitGroup{},
		seqLeaseLock:       &sync.RWMutex{},
		scavengerLeaseLock: &sync.RWMutex{},
		pollTick:           defaultPollTick,
		idempotencyTTL:     defaultIdempotencyTTL,
		queueKindMapping:   make(map[string]string),
		scope:              tally.NoopScope,
		tracer:             trace.NewNoopTracerProvider().Tracer("redis_queue"),
		logger:             logger.From(context.Background()),
		partitionConcurrencyGen: func(ctx context.Context, p QueuePartition) (string, int) {
			return p.Queue(), 10_000
		},
	}

	for _, opt := range opts {
		opt(q)
	}

	q.sem = &trackingSemaphore{Weighted: semaphore.NewWeighted(int64(q.numWorkers))}
	q.workers = make(chan processItem, q.numWorkers)

	return q
}

type queue struct {
	// name is the identifiable name for this worker, for logging.
	name string
	// redis stores the redis connection to use.
	r  rueidis.Client
	pf PriorityFinder
	kg QueueKeyGenerator

	accountConcurrencyGen   QueueItemConcurrencyKeyGenerator
	partitionConcurrencyGen PartitionConcurrencyKeyGenerator
	customConcurrencyGen    QueueItemConcurrencyKeyGenerator
	// concurrencyService is an external concurrency limiter used when pulling
	// jobs off of the queue.  It is only invoked for jobs with a non-zero function ID,
	// eg. for jobs that run a function.
	concurrencyService concurrency.ConcurrencyService

	// idempotencyTTL is the default or static idempotency duration apply to jobs,
	// if idempotencyTTLFunc is not defined.
	idempotencyTTL time.Duration
	// idempotencyTTLFunc returns an time.Duration representing how long job IDs
	// remain idempotent.
	idempotencyTTLFunc func(context.Context, QueueItem) time.Duration
	// pollTick is the interval between each scan for jobs.
	pollTick time.Duration
	// quit is a channel that any method can send on to trigger termination
	// of the Run loop.  This typically accepts an error, but a nil error
	// will still quit the runner.
	quit chan error
	// wg stores a waitgroup for all in-progress jobs
	wg *sync.WaitGroup
	// numWorkers stores the number of workers available to concurrently process jobs.
	numWorkers int32
	// workers is a buffered channel which allows scanners to send queue items
	// to workers to be processed
	workers chan processItem
	// sem stores a semaphore controlling the number of jobs currently
	// being processed.  This lets us check whether there's capacity in the queue
	// prior to leasing items.
	sem *trackingSemaphore
	// queueKindMapping stores a map of job kind => queue names
	queueKindMapping map[string]string
	logger           *zerolog.Logger

	// denyQueues provides a denylist ensuring that the queue will never claim
	// this partition, meaning that no jobs from this queue will run on this worker.
	denyQueues   []string
	denyQueueMap map[string]*struct{}

	// allowQueues provides an allowlist, ensuring that the queue only peeks the specified
	// partitions.  jobs from other partitions will never be scanned or processed.
	allowQueues   []string
	allowQueueMap map[string]*struct{}

	// seqLeaseID stores the lease ID if this queue is the sequential processor.
	// all runners attempt to claim this lease automatically.
	seqLeaseID *ulid.ULID
	// seqLeaseLock ensures that there are no data races writing to
	// or reading from seqLeaseID in parallel.
	seqLeaseLock *sync.RWMutex

	// scavengerLeaseID stores the lease ID if this queue is the scavenger processor.
	// all runners attempt to claim this lease automatically.
	scavengerLeaseID *ulid.ULID
	// scavengerLeaseLock ensures that there are no data races writing to
	// or reading from scavengerLeaseID in parallel.
	scavengerLeaseLock *sync.RWMutex

	// metrics allows reporting of metrics
	scope tally.Scope
	// tracer is the tracer to use for opentelemetry tracing.
	tracer trace.Tracer
}

// processItem references the queue partition and queue item to be processed by a worker.
// both items need to be passed to a worker as both items are needed to generate concurrency
// keys to extend leases and dequeue.
type processItem struct {
	P QueuePartition
	I QueueItem
}

// QueueItem represents an individually queued work scheduled for some time in the
// future.
type QueueItem struct {
	// ID represents a unique identifier for the queue item.  This can be any
	// unique string and will be hashed.  Using the same ID provides idempotency
	// guarantees within the queue's IdempotencyTTL.
	ID string `json:"id"`
	// At represents the current time that this QueueItem needs to be executed at,
	// as a millisecond epoch.  This is the millisecond-level granularity score of
	// the item.  Note that the score in Redis is second-level, ie this field / 1000.
	//
	// This is necessary for rescoring partitions and checking latencies.
	AtMS int64 `json:"at"`
	// WorkflowID is the workflow ID that this job belongs to.
	WorkflowID uuid.UUID `json:"wfID"`
	// WorkspaceID is the workspace that this job belongs to.
	WorkspaceID uuid.UUID `json:"wsID"`
	// LeaseID is a ULID which embeds a timestamp denoting when the lease expires.
	LeaseID *ulid.ULID `json:"leaseID,omitempty"`
	// Data represents the enqueued data, eg. the edge to process or the pause
	// to resume.
	Data osqueue.Item `json:"data"`
	// QueueName allows placing this job into a specific queue name.  If the QueueName
	// is nil, the WorkflowID will be used as the queue name.  This allows us to
	// automatically create partitioned queues for each function within Inngest.
	//
	// This should almost always be nil.
	QueueName *string `json:"queueID,omitempty"`
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
func (q QueueItem) Score() int64 {
	// If this is not an edge, we can ignore this.
	if q.Data.Kind != osqueue.KindEdge || q.Data.Attempt > 0 {
		return q.AtMS
	}

	// If this is > 2 seconds in the future, don't mess with the time.
	// This prevents any accidental fudging of future run times, even if the
	// kind is edge (which should never exist... but, better to be safe).
	if q.AtMS > time.Now().Add(2*time.Second).UnixMilli() {
		return q.AtMS
	}

	// Only fudge the numbers if the run is older than the buffer time.
	startAt := int64(q.Data.Identifier.RunID.Time())
	if q.AtMS-startAt > FunctionStartScoreBufferTime.Milliseconds() {
		return startAt
	}
	return q.AtMS
}

func (q QueueItem) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

// Queue returns the queue name for this queue item.  This is the
// workflow ID of the QueueItem unless the QueueName is specifically
// set.
func (q QueueItem) Queue() string {
	if q.QueueName == nil {
		return q.WorkflowID.String()
	}
	return *q.QueueName
}

// IsLeased checks if the QueueItem is currently already leased or not
// based on the time passed in.
func (q QueueItem) IsLeased(time time.Time) bool {
	return q.LeaseID != nil && ulid.Time(q.LeaseID.Time()).After(time)
}

// QueuePartition represents an individual queue for a workflow.  It stores the
// time of the earliest job within the workflow.
type QueuePartition struct {
	QueueName *string `json:"queue,omitempty"`

	WorkflowID uuid.UUID `json:"wid"`

	Priority uint `json:"p"`

	// AtS refers to the earliest QueueItem time within this partition, in
	// seconds as a unix epoch.
	//
	// This is updated when taking a lease, requeuing items, etc.
	//
	// The S suffix differentiates between a QueueItem;  we only need second-
	// level granularity here as queue partitions work to the nearest second.
	AtS int64 `json:"at"`

	// Last represents the time that this QueueItem was last leased, as a unix
	// epoch.
	//
	// This lets us inspect the time a QueuePartition was last leased, and figure
	// out whether we should garbage collect the partition index.
	Last int64 `json:"last"`

	// LeaseID represents a lease on this partition.  If the LeaseID is not nil,
	// this partition can be claimed by a shared-nothing worker to work on the
	// queue items within this partition.
	//
	// A lease is shortly held (eg seconds).  It should last long enough for
	// workers to claim QueueItems only.
	LeaseID *ulid.ULID `json:"leaseID"`
}

func (q QueuePartition) Queue() string {
	if q.QueueName == nil {
		return q.WorkflowID.String()
	}
	return *q.QueueName
}

func (q QueuePartition) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

// EnqueueItem enqueues a QueueItem.  It creates a QueuePartition for the workspace
// if a partition does not exist.
//
// The QueueItem's ID can be a zero UUID;  if the ID is a zero value a new ID
// will be created for the queue item.
//
// The queue score must be added in milliseconds to process sub-second items in order.
func (q *queue) EnqueueItem(ctx context.Context, i QueueItem, at time.Time) (QueueItem, error) {
	ctx, span := q.tracer.Start(ctx, "EnqueueItem")
	defer span.End()

	if len(i.ID) == 0 {
		i.ID = ulid.MustNew(ulid.Now(), rnd).String()
	}

	// Hash the ID.
	i.ID = hashID(ctx, i.ID)

	priority := PriorityMin
	if q.pf != nil {
		priority = q.pf(ctx, i)
	}

	if priority > PriorityMin {
		return i, ErrPriorityTooLow
	}
	if priority < PriorityMax {
		return i, ErrPriorityTooHigh
	}

	// Add the At timestamp, if not included.
	if i.AtMS == 0 {
		i.AtMS = at.UnixMilli()
	}

	if i.Data.JobID == nil {
		i.Data.JobID = &i.ID
	}

	// Get the queue name from the queue item.  This allows utilization of
	// the partitioned queue for jobs with custom queue names, vs utilizing
	// workflow IDs in every case.
	qn := i.Queue()

	qp := QueuePartition{
		QueueName:  i.QueueName,
		WorkflowID: i.WorkflowID,
		Priority:   priority,
		AtS:        at.Unix(),
	}

	keys := []string{
		q.kg.QueueItem(),       // Queue item
		q.kg.QueueIndex(qn),    // Queue sorted set
		q.kg.PartitionItem(),   // Partition item, map
		q.kg.PartitionMeta(qn), // Partition item
		q.kg.PartitionIndex(),  // Global partition queue
		q.kg.Idempotency(i.ID),
	}

	args, err := StrSlice([]any{
		i,
		i.ID,
		at.UnixMilli(),
		qn,
		qp,
	})
	if err != nil {
		return i, err
	}
	status, err := scripts["queue/enqueue"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return i, fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		return i, nil
	case 1:
		return i, ErrQueueItemExists
	default:
		return i, fmt.Errorf("unknown response enqueueing item: %d", status)
	}
}

// Peek takes n items from a queue, up until QueuePeekMax.  For peeking workflow/
// function jobs the queue name must be the ID of the workflow;  each workflow has
// its own queue of jobs using its ID as the queue name.
//
// If limit is -1, this will return the first unleased item - representing the next available item in the
// queue.
func (q *queue) Peek(ctx context.Context, queueName string, until time.Time, limit int64) ([]*QueueItem, error) {
	ctx, span := q.tracer.Start(ctx, "Peek")
	defer span.End()

	// Check whether limit is -1, peeking next available time
	isPeekNext := limit == -1

	if limit > QueuePeekMax {
		// Lua's max unpack() length is 8000; don't allow users to peek more than
		// 1k at a time regardless.
		return nil, ErrQueuePeekMaxExceedsLimits
	}
	if limit <= 0 {
		limit = QueuePeekMax
	}

	args, err := StrSlice([]any{
		until.UnixMilli(),
		limit,
	})
	if err != nil {
		return nil, err
	}
	items, err := scripts["queue/peek"].Exec(
		ctx,
		q.r,
		[]string{
			q.kg.QueueIndex(queueName),
			q.kg.QueueItem(),
		},
		args,
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking queue items: %w", err)
	}

	// Create a slice up to items in length.  We're going to remove any items that are
	// leased here, so we may end up returning less than the total length.
	result := make([]*QueueItem, len(items))
	n := 0
	now := time.Now()

	for _, str := range items {
		qi := &QueueItem{}
		if err := json.Unmarshal([]byte(str), qi); err != nil {
			return nil, fmt.Errorf("error unmarshalling peeked queue item: %w", err)
		}
		if qi.IsLeased(now) {
			// Leased item, don't return.
			continue
		}
		// The nested osqueue.Item never has an ID set;  always re-set it
		qi.Data.JobID = &qi.ID
		result[n] = qi
		n++

		if isPeekNext {
			return []*QueueItem{qi}, nil
		}
	}

	return result[0:n], nil
}

// Lease temporarily dequeues an item from the queue by obtaining a lease, preventing
// other workers from working on this queue item at the same time.
//
// Obtaining a lease updates the vesting time for the queue item until now() +
// lease duration. This returns the newly acquired lease ID on success.
func (q *queue) Lease(ctx context.Context, p QueuePartition, item QueueItem, duration time.Duration) (*ulid.ULID, error) {
	ctx, span := q.tracer.Start(ctx, "Lease")
	defer span.End()

	var (
		ak, pk, ck string // account, partition, custom concurrency key
		ac, pc, cc int    // account, partiiton, custom concurrency max
	)

	// required
	pk, pc = q.partitionConcurrencyGen(ctx, p)
	// optional
	if q.accountConcurrencyGen != nil {
		ak, ac = q.accountConcurrencyGen(ctx, item)
	}
	if q.customConcurrencyGen != nil {
		ck, cc = q.customConcurrencyGen(ctx, item) // Get the custom concurrency key, if available.
	}

	leaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(duration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(item.Queue()),
		q.kg.PartitionMeta(item.Queue()),
		q.kg.Concurrency("account", ak),
		q.kg.Concurrency("p", pk),
		q.kg.Concurrency("custom", ck),
		q.kg.ConcurrencyIndex(),
	}
	args, err := StrSlice([]any{
		item.ID,
		leaseID.String(),
		time.Now().UnixMilli(),
		ac,
		pc,
		cc,
		p.Queue(),
	})
	if err != nil {
		return nil, err
	}
	status, err := scripts["queue/lease"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing queue item: %w", err)
	}
	switch status {
	case 0:
		return &leaseID, nil
	case 1:
		return nil, ErrQueueItemNotFound
	case 2:
		return nil, ErrQueueItemAlreadyLeased
	case 3:
		return nil, ErrConcurrencyLimit
	default:
		return nil, fmt.Errorf("unknown response enqueueing item: %d", status)
	}
}

// ExtendLease extens the lease for a given queue item, given the queue item is currently
// leased with the given ID.  This returns a new lease ID if the lease is successfully ended.
//
// The existing lease ID must be passed in so that we can guarantee that the worker
// renewing the lease still owns the lease.
//
// Renewing a lease updates the vesting time for the queue item until now() +
// lease duration. This returns the newly acquired lease ID on success.
func (q *queue) ExtendLease(ctx context.Context, p QueuePartition, i QueueItem, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	ctx, span := q.tracer.Start(ctx, "ExtendLease")
	defer span.End()

	var (
		ak, pk, ck string // account, partition, custom concurrency key
	)
	// required
	pk, _ = q.partitionConcurrencyGen(ctx, p)
	// optional
	if q.accountConcurrencyGen != nil {
		ak, _ = q.accountConcurrencyGen(ctx, i)
	}
	if q.customConcurrencyGen != nil {
		ck, _ = q.customConcurrencyGen(ctx, i)
	}

	newLeaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(duration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(i.Queue()),
		q.kg.PartitionIndex(),
		q.kg.Concurrency("account", ak),
		q.kg.Concurrency("p", pk),
		q.kg.Concurrency("custom", ck),
	}

	args, err := StrSlice([]any{
		i.ID,
		leaseID.String(),
		newLeaseID.String(),
		p.Queue(),
	})
	if err != nil {
		return nil, err
	}
	status, err := scripts["queue/extendLease"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error extending lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrQueueItemNotFound
	case 2:
		return nil, ErrQueueItemNotLeased
	case 3:
		return nil, ErrQueueItemLeaseMismatch
	default:
		return nil, fmt.Errorf("unknown response enqueueing item: %d", status)
	}
}

// Dequeue removes an item from the queue entirely.
func (q *queue) Dequeue(ctx context.Context, p QueuePartition, i QueueItem) error {
	var (
		ak, pk, ck string // account, partition, custom concurrency key
	)
	// required
	pk, _ = q.partitionConcurrencyGen(ctx, p)
	// optional
	if q.accountConcurrencyGen != nil {
		ak, _ = q.accountConcurrencyGen(ctx, i)
	}
	if q.customConcurrencyGen != nil {
		ck, _ = q.customConcurrencyGen(ctx, i)
	}

	qn := i.Queue()
	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(qn),
		q.kg.PartitionMeta(qn),
		q.kg.Idempotency(i.ID),
		q.kg.Concurrency("account", ak),
		q.kg.Concurrency("p", pk),
		q.kg.Concurrency("custom", ck),
		q.kg.ConcurrencyIndex(),
	}

	idempotency := q.idempotencyTTL
	if q.idempotencyTTLFunc != nil {
		idempotency = q.idempotencyTTLFunc(ctx, i)
	}

	args, err := StrSlice([]any{
		i.ID,
		int(idempotency.Seconds()),
		p.Queue(),
	})
	if err != nil {
		return err
	}
	status, err := scripts["queue/dequeue"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error dequeueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response dequeueing item: %d", status)
	}
}

// Requeue requeues an item in the future.
func (q *queue) Requeue(ctx context.Context, p QueuePartition, i QueueItem, at time.Time) error {
	ctx, span := q.tracer.Start(ctx, "Requeue")
	defer span.End()

	priority := PriorityMin
	if q.pf != nil {
		priority = q.pf(ctx, i)
	}

	if priority > PriorityMin {
		return ErrPriorityTooLow
	}

	var (
		ak, pk, ck string // account, partition, custom concurrency key
	)

	// required
	pk, _ = q.partitionConcurrencyGen(ctx, p)
	// optional
	if q.accountConcurrencyGen != nil {
		ak, _ = q.accountConcurrencyGen(ctx, i)
	}
	if q.customConcurrencyGen != nil {
		ck, _ = q.customConcurrencyGen(ctx, i) // Get the custom concurrency key, if available.
	}

	// Unset any lease ID as this is requeued.
	i.LeaseID = nil
	// Update the At timestamp.
	i.AtMS = at.UnixMilli()

	qn := i.Queue()

	qp := QueuePartition{
		QueueName:  i.QueueName,
		WorkflowID: i.WorkflowID,
		Priority:   priority,
		AtS:        at.Unix(),
	}
	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(qn),
		q.kg.PartitionMeta(qn),
		q.kg.PartitionIndex(),
		q.kg.Concurrency("account", ak),
		q.kg.Concurrency("p", pk),
		q.kg.Concurrency("custom", ck),
		q.kg.ConcurrencyIndex(),
	}

	args, err := StrSlice([]any{
		i,
		i.ID,
		at.UnixMilli(),
		qn,
		qp,
	})
	if err != nil {
		return err
	}
	status, err := scripts["queue/requeue"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error requeueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	default:
		return fmt.Errorf("unknown response enqueueing item: %d", status)
	}
}

// PartitionLease leases a parititon for a given workflow ID.  It returns the new lease ID.
//
// NOTE: This does not check the queue/partition name against allow or denylists;  it assumes
// that the worker always wants to lease the given queue.  Filtering must be done when peeking
// when running a worker.
func (q *queue) PartitionLease(ctx context.Context, p QueuePartition, duration time.Duration) (*ulid.ULID, int64, error) {
	ctx, span := q.tracer.Start(ctx, "PartitionLease")
	defer span.End()

	var (
		concurrencyKey string
		concurrency    = defaultPartitionConcurrency
	)
	if q.partitionConcurrencyGen != nil {
		concurrencyKey, concurrency = q.partitionConcurrencyGen(ctx, p)
	}

	// XXX: Check for function throttling prior to leasing;  if it's throttled we can requeue
	// the pointer and back off.  A question here is enqueuing new items onto the partition
	// will reset the pointer update, leading to thrash.
	now := time.Now()
	leaseExpires := now.Add(duration).UTC().Truncate(time.Millisecond)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpires), rnd)
	if err != nil {
		return nil, 0, fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		q.kg.PartitionItem(),
		q.kg.PartitionIndex(),
		q.kg.Concurrency("p", concurrencyKey),
	}

	args, err := StrSlice([]any{
		p.Queue(),
		leaseID.String(),
		now.UnixMilli(),
		leaseExpires.Unix(),
		concurrency,
	})
	if err != nil {
		return nil, 0, err
	}
	result, err := scripts["queue/partitionLease"].Exec(
		ctx,
		q.r,
		keys,
		args,
		// TODO: Partition concurrency defer amount

	).AsInt64()
	if err != nil {
		return nil, 0, fmt.Errorf("error leasing partition: %w", err)
	}
	switch result {
	case -1:
		return nil, 0, ErrPartitionConcurrencyLimit
	case -2:
		return nil, 0, ErrPartitionNotFound
	case -3:
		return nil, 0, ErrPartitionAlreadyLeased
	default:
		// If there's no concurrency limit for this partition, return a default
		// amount so that processing the partition has reasonable limits.
		if concurrency == 0 {
			return &leaseID, QueuePeekDefault, nil
		}

		// result is the available concurrency within this partition
		return &leaseID, result, nil
	}
}

func (q *queue) PartitionLeaseByID(ctx context.Context, id string, duration time.Duration) (*ulid.ULID, int64, error) {
	// Fetch the partition.
	return nil, 0, nil
}

// PartitionPeek returns up to PartitionSelectionMax partition items from the queue. This
// returns the indexes of partitions.
//
// If sequential is set to true this returns partitions in order from earliest to latest
// available lease times. Otherwise, this shuffles all partitions and picks partitions
// randomly, with higher priority partitions more likely to be selected.  This reduces
// lease contention amongst multiple shared-nothing workers.
func (q *queue) PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error) {
	if limit > PartitionPeekMax {
		return nil, ErrPartitionPeekMaxExceedsLimits
	}
	if limit <= 0 {
		limit = PartitionPeekMax
	}

	// TODO: If this is an allowlist, only peek the given partitions.  Use ZMSCORE
	// to fetch the scores for all allowed partitions, then filter where score <= until.
	// Call an HMGET to get the partitions.

	unix := until.Unix()

	args, err := StrSlice([]any{
		unix,
		limit,
	})
	if err != nil {
		return nil, err
	}

	encoded, err := scripts["queue/partitionPeek"].Exec(
		ctx,
		q.r,
		[]string{
			q.kg.PartitionIndex(),
			q.kg.PartitionItem(),
		},
		args,
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking partition items: %w", err)
	}

	weights := []float64{}
	items := make([]*QueuePartition, len(encoded))

	ignored := 0
	for n, i := range encoded {
		if i == "" {
			ignored++
			continue
		}

		item := &QueuePartition{}
		if err = json.Unmarshal([]byte(i), item); err != nil {
			return nil, fmt.Errorf("error reading partition item: %w", err)
		}

		// If we have an allowlist, only accept this partition if its in the allowlist.
		if len(q.allowQueues) > 0 && q.allowQueueMap[item.Queue()] == nil {
			ignored++
			continue
		}

		// Ignore any denied queues if they're explicitly in the denylist.  Because
		// we allocate the len(encoded) amount, we also want to track the number of
		// ignored queues to use the correct index when setting our items;  this ensures
		// that we don't access items with an index and get nil pointers.
		if len(q.denyQueues) > 0 && q.denyQueueMap[item.Queue()] != nil {
			ignored++
			continue
		}

		items[n-ignored] = item
		weights = append(weights, float64(10-item.Priority))
	}

	// Remove any ignored items from the slice.
	items = items[0 : len(items)-ignored]

	// Some scanners run sequentially, ensuring we always work on the functions with
	// the oldest run at times in order, no matter the priority.
	if sequential {
		n := int(math.Min(float64(len(items)), float64(PartitionSelectionMax)))
		return items[0:n], nil
	}

	// We want to weighted shuffle the resulting array random.  This means that many
	// shared nothing scanners can query for outstanding partitions and receive a
	// randomized order favouring higher-priority queue items.  This reduces the chances
	// of contention when leasing.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*QueuePartition, len(items))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return nil, ErrWeightedSampleRead
		}
		result[n] = items[idx]
	}

	return result, nil
}

// PartitionRequeue requeues a parition with a new score, ensuring that the partition will be
// read at (or very close to) the given time.
//
// This is used after peeking and passing all queue items onto workers; we then take the next
// unleased available time for the queue item and requeue the partition.
//
// forceAt is used to enforce the given queue time.  This is used when partitions are at a
// concurrency limit;  we don't want to scan the partition next time, so we force the partition
// to be at a specific time instead of taking the earliest available queue item time
func (q *queue) PartitionRequeue(ctx context.Context, queueName string, at time.Time, forceAt bool) error {
	ctx, span := q.tracer.Start(ctx, "PartitionRequeue")
	defer span.End()

	keys := []string{
		q.kg.PartitionItem(),
		q.kg.PartitionIndex(),
		q.kg.PartitionMeta(queueName),
		q.kg.QueueIndex(queueName),
		q.kg.QueueItem(),
		q.kg.Concurrency("p", queueName),
	}
	force := 0
	if forceAt {
		force = 1
	}
	args, err := StrSlice([]any{

		queueName,
		at.Unix(),
		force,
	})
	if err != nil {
		return err
	}
	status, err := scripts["queue/partitionRequeue"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error requeueing partition: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return ErrPartitionNotFound
	case 2:
		return ErrPartitionGarbageCollected
	default:
		return fmt.Errorf("unknown response requeueing item: %d", status)
	}
}

// PartitionDequeue removes a partition pointer from the queue.  This is used when peeking and
// receiving zero items to run.
func (q *queue) PartitionDequeue(ctx context.Context, queueName string, at time.Time) error {
	panic("unimplemented: requeueing partitions handles this.")
}

// PartitionReprioritize reprioritizes a workflow's QueueItems within the queue.
func (q *queue) PartitionReprioritize(ctx context.Context, queueName string, priority uint) error {
	if priority > PriorityMin {
		return ErrPriorityTooLow
	}
	if priority < PriorityMax {
		return ErrPriorityTooHigh
	}

	args, err := StrSlice([]any{
		queueName,
		priority,
	})
	if err != nil {
		return err
	}

	keys := []string{q.kg.PartitionItem()}
	status, err := scripts["queue/partitionReprioritize"].Exec(
		ctx,
		q.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return ErrPartitionNotFound
	default:
		return fmt.Errorf("unknown response reprioritizing partition: %d", status)
	}
}

func (q *queue) InProgress(ctx context.Context, prefix string, concurrencyKey string) (int64, error) {
	s := time.Now().UnixMilli()
	cmd := q.r.B().Zcount().
		Key(q.kg.Concurrency(prefix, concurrencyKey)).
		Min(fmt.Sprintf("%d", s)).
		Max("+inf").
		Build()
	return q.r.Do(ctx, cmd).AsInt64()
}

// Scavenge attempts to find jobs that may have been lost due to killed workers.  Workers are shared
// nothing, and each item in a queue has a lease.  If a worker dies, it will not finish the job and
// cannot renew the item's lease.
//
// We scan all partition concurrency queues - queues of leases - to find leases that have expired.
func (q *queue) Scavenge(ctx context.Context) (int, error) {
	// Find all items that have an expired lease - eg. where the min time for a lease is between
	// (0-now] in unix milliseconds.
	now := fmt.Sprintf("%d", time.Now().UnixMilli())

	cmd := q.r.B().Zrange().
		Key(q.kg.ConcurrencyIndex()).
		Min("-inf").
		Max(now).
		Byscore().
		Limit(0, 100).
		Build()

	pKeys, err := q.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("error scavenging for lost items: %w", err)
	}

	counter := 0

	// Each of the items is a concurrency queue with lost items.
	var resultErr error
	for _, partition := range pKeys {
		// Fetch the partition.  This uses the concurrency:p: prefix,
		// so remove the prefix from the item.
		partitionJSON, err := q.r.Do(ctx, q.r.B().Hget().Key(q.kg.PartitionItem()).Field(partition).Build()).AsBytes()
		if err == rueidis.Nil {
			continue
		}
		if err != nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("error finding partition '%s' during scavenge: %w", partition, err))
			continue
		}

		cmd := q.r.B().Zrange().
			Key(q.kg.Concurrency("p", partition)).
			Min("-inf").
			Max(now).
			Byscore().
			Limit(0, 100).
			Build()
		itemIDs, err := q.r.Do(ctx, cmd).AsStrSlice()
		if err != nil && err != rueidis.Nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("error querying partition concurrency queue '%s' during scavenge: %w", partition, err))
			continue
		}
		if len(itemIDs) == 0 {
			continue
		}

		p := QueuePartition{}
		if err := json.Unmarshal([]byte(partitionJSON), &p); err != nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("error unmarshalling partition '%s': %w", partitionJSON, err))
			continue
		}

		// Fetch the queue item, then requeue.
		cmd = q.r.B().Hmget().Key(q.kg.QueueItem()).Field(itemIDs...).Build()
		jobs, err := q.r.Do(ctx, cmd).AsStrSlice()
		if err != nil && err != rueidis.Nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("error fetching jobs for concurrency queue '%s' during scavenge: %w", partition, err))
			continue
		}
		for _, item := range jobs {
			qi := QueueItem{}
			if err := json.Unmarshal([]byte(item), &qi); err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error unmarshalling job '%s': %w", item, err))
				continue
			}
			if err := q.Requeue(ctx, p, qi, time.Now()); err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error requeueing job '%s': %w", item, err))
				continue
			}
			counter++
		}
	}

	return counter, resultErr
}

// ConfigLease allows a worker to lease config keys for sequential or scavenger processing.
// Leasing this key works similar to leasing partitions or queue items:
//
//   - If the key isn't leased, a new lease is accepted.
//   - If the lease is expired, a new lease is accepted.
//   - If the key is leased, you must pass in the existing lease ID to renew the lease.  Mismatches do not
//     grant a lease.
//
// This returns the new lease ID on success.
//
// If the sequential key is leased, this allows a worker to peek partitions sequentially.
func (q *queue) ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	if duration > ConfigLeaseMax {
		return nil, ErrConfigLeaseExceedsLimits
	}

	now := time.Now()
	newLeaseID, err := ulid.New(ulid.Timestamp(now.Add(duration)), rnd)
	if err != nil {
		return nil, err
	}

	var existing string
	if len(existingLeaseID) > 0 && existingLeaseID[0] != nil {
		existing = existingLeaseID[0].String()
	}

	args, err := StrSlice([]any{
		now.UnixMilli(),
		newLeaseID.String(),
		existing,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/configLease"].Exec(
		ctx,
		q.r,
		[]string{key},
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error claiming config lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrConfigAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response claiming config lease: %d", status)
	}
}

func hashID(ctx context.Context, id string) string {
	ui := xxhash.Sum64String(id)
	return strconv.FormatUint(ui, 36)
}

// frandRNG is a fast crypto-secure prng which uses a mutex to guard
// parallel reads.  It also implements the x/exp/rand.Source interface
// by adding a Seed() method which does nothing.
type frandRNG struct {
	*frand.RNG
	lock *sync.Mutex
}

func (f *frandRNG) Read(b []byte) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.RNG.Read(b)
}

func (f *frandRNG) Uint64() uint64 {
	return f.Uint64n(math.MaxUint64)
}

func (f *frandRNG) Uint64n(n uint64) uint64 {
	// sampled.Take calls Uint64n, which must be guarded by a lock in order
	// to be thread-safe.
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.RNG.Uint64n(n)
}

func (f *frandRNG) Float64() float64 {
	// sampled.Take also calls Float64, which must be guarded by a lock in order
	// to be thread-safe.
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.RNG.Float64()
}

func (f *frandRNG) Seed(seed uint64) {
	// Do nothing.
}
