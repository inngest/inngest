package redis_state

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/cespare/xxhash/v2"
	"github.com/go-redis/redis/v8"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/uber-go/tally"
	"gonum.org/v1/gonum/stat/sampleuv"
	"lukechampine.com/frand"
)

const (
	PartitionSelectionMax int64 = 20
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
	PartitionRequeueExtension       = 30 * time.Second
	QueueSelectionMax         int64 = 50
	QueuePeekMax              int64 = 1000
	QueuePeekDefault          int64 = QueueSelectionMax * 3
	QueueLeaseDuration              = 10 * time.Second
	SequentialLeaseDuration         = 10 * time.Second
	SequentialLeaseMax              = 20 * time.Second

	PriorityMax     uint = 0
	PriorityDefault uint = 5
	PriorityMin     uint = 9

	defaultNumWorkers     = 100
	defaultPollTick       = 10 * time.Millisecond
	defaultIdempotencyTTL = 12 * time.Hour
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
	ErrSequentialAlreadyLeased       = fmt.Errorf("sequential scanner already leased")
	ErrSequentialLeaseExceedsLimits  = fmt.Errorf("sequential lease duration exceeds the maximum of %d seconds", int(SequentialLeaseMax.Seconds()))
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
		q.metrics = scope
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

// WithAllowQeueNames specifies that the worker that must only select jobs from the
// given queues.
//
// For speed, we only allow the allowlist to be a maximum of 100 queues long.  This
// list is passed to Redis when peeking partitions each time.
//
// NOTE: If this is set, this worker will never be allowed to claim sequential leases,
// as this assumes that the worker can only run a small subset of queues available. This
// means that the worker is not able to participate in fairness and latency for all queues.
//
// func WithAllowQueueNames(queues ...string) func(q *queue) {
// 	return func(q *queue) {
// 		q.allowQueues = queues
// 		q.allowQueuesJSON, _ = json.Marshal(queues)
// 	}
// }

func NewQueue(r redis.UniversalClient, opts ...QueueOpt) *queue {
	q := &queue{
		r: r,
		pf: func(ctx context.Context, item QueueItem) uint {
			return PriorityDefault
		},
		kg:             defaultQueueKey,
		numWorkers:     defaultNumWorkers,
		metrics:        tally.NewTestScope("queue", map[string]string{}),
		wg:             &sync.WaitGroup{},
		seqLeaseLock:   &sync.RWMutex{},
		pollTick:       defaultPollTick,
		idempotencyTTL: defaultIdempotencyTTL,
	}

	for _, opt := range opts {
		opt(q)
	}

	q.sem = &trackingSemaphore{Weighted: semaphore.NewWeighted(int64(q.numWorkers))}
	q.workers = make(chan QueueItem, q.numWorkers)

	return q
}

type queue struct {
	// name is the identifiable name for this worker, for logging.
	name string
	// redis stores the redis connection to use.
	r  redis.UniversalClient
	pf PriorityFinder
	kg QueueKeyGenerator
	// metrics allows reporting of metrics
	metrics tally.Scope

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
	workers chan QueueItem
	// sem stores a semaphore controlling the number of jobs currently
	// being processed.  This lets us check whether there's capacity in the queue
	// prior to leasing items.
	sem *trackingSemaphore

	// denyQueues provides a denylist ensuring that the queue will never claim
	// this partition, meaning that no jobs from this queue will run on this worker.
	denyQueues   []string
	denyQueueMap map[string]*struct{}

	// allowQueues provides an allowlist of queue names, ensuring that the worker only
	// peeks and claims queues with names from this list.
	allowQueues []string
	// allowQueuesJSON stores a JSON-encoded representation of allow queues.
	allowQueuesJSON []byte

	// seqLeaseID stores the lease ID if this queue is the sequential processor.
	// all runners attempt to claim this lease automatically.
	seqLeaseID *ulid.ULID
	// seqLeaseLock ensures that there are no data races writing to
	// or reading from seqLeaseID in parallel.
	seqLeaseLock *sync.RWMutex
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

	// Add the At timestamp.
	i.AtMS = at.UnixMilli()

	// Get the queue name from the queue item.  This allows utilization of
	// the partitioned queue for jobs with custom queue names, vs utilizing
	// workflow IDs in every case.
	qn := i.Queue()

	qp := QueuePartition{WorkflowID: i.WorkflowID, Priority: priority, AtS: at.Unix()}
	keys := []string{
		q.kg.QueueItem(),       // Queue item
		q.kg.QueueIndex(qn),    // Queue sorted set
		q.kg.PartitionItem(),   // Partition item, map
		q.kg.PartitionMeta(qn), // Partition item
		q.kg.PartitionIndex(),  // Global partition queue
		q.kg.Idempotency(i.ID),
	}
	status, err := scripts["queue/enqueue"].Run(
		ctx,
		q.r,
		keys,

		i,
		i.ID,
		at.UnixMilli(),
		qn,
		qp,
	).Int64()
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

	items, err := scripts["queue/peek"].Run(
		ctx,
		q.r,
		[]string{
			q.kg.QueueIndex(queueName),
			q.kg.QueueItem(),
		},
		until.UnixMilli(),
		limit,
	).StringSlice()
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
		if qi.LeaseID != nil && now.Before(ulid.Time(qi.LeaseID.Time())) {
			// Leased item, don't return.
			continue
		}
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
//
// itemID must be the hashed ID of the queue item.
func (q *queue) Lease(ctx context.Context, queueName string, itemID string, duration time.Duration) (*ulid.ULID, error) {
	// TODO: Add custom throttling here.
	leaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(duration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(queueName),
		q.kg.PartitionMeta(queueName),
	}
	status, err := scripts["queue/lease"].Run(
		ctx,
		q.r,
		keys,
		itemID,
		leaseID.String(),
		time.Now().UnixMilli(),
	).Int64()
	if err != nil {
		return nil, fmt.Errorf("error leasing pause: %w", err)
	}
	switch status {
	case 0:
		return &leaseID, nil
	case 1:
		return nil, ErrQueueItemNotFound
	case 2:
		return nil, ErrQueueItemAlreadyLeased
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
func (q *queue) ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	newLeaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(duration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(i.Queue()),
	}
	status, err := scripts["queue/extendLease"].Run(
		ctx,
		q.r,
		keys,
		i.ID,
		leaseID.String(),
		newLeaseID.String(),
	).Int64()
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
func (q *queue) Dequeue(ctx context.Context, i QueueItem) error {
	qn := i.Queue()
	keys := []string{
		q.kg.QueueItem(),
		q.kg.QueueIndex(qn),
		q.kg.PartitionMeta(qn),
		q.kg.Idempotency(i.ID),
	}

	idempotency := q.idempotencyTTL
	if q.idempotencyTTLFunc != nil {
		idempotency = q.idempotencyTTLFunc(ctx, i)
	}

	status, err := scripts["queue/dequeue"].Run(
		ctx,
		q.r,
		keys,

		i.ID,
		int(idempotency.Seconds()),
	).Int64()
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
func (q *queue) Requeue(ctx context.Context, i QueueItem, at time.Time) error {
	priority := PriorityMin
	if q.pf != nil {
		priority = q.pf(ctx, i)
	}

	if priority > PriorityMin {
		return ErrPriorityTooLow
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
	}
	status, err := scripts["queue/requeue"].Run(
		ctx,
		q.r,
		keys,

		i,
		i.ID,
		at.UnixMilli(),
		qn,
		qp,
	).Int64()
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
func (q *queue) PartitionLease(ctx context.Context, queueName string, duration time.Duration) (*ulid.ULID, error) {
	// XXX: Check for function throttling prior to leasing;  if it's throttled we can requeue
	// the pointer and back off.  A question here is enqueuing new items onto the partition
	// will reset the pointer update, leading to thrash.
	now := time.Now()
	leaseExpires := now.Add(duration).UTC().Truncate(time.Millisecond)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpires), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		q.kg.PartitionItem(),
		q.kg.PartitionIndex(),
	}
	status, err := scripts["queue/partitionLease"].Run(
		ctx,
		q.r,
		keys,

		queueName,
		leaseID.String(),
		now.UnixMilli(),
		leaseExpires.Unix(),
	).Int64()
	if err != nil {
		return nil, fmt.Errorf("error leasing partition: %w", err)
	}
	switch status {
	case 0:
		return &leaseID, nil
	case 1:
		return nil, ErrPartitionNotFound
	case 2:
		return nil, ErrPartitionAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response enqueueing item: %d", status)
	}
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

	// TODO: If this is an allowlist, only peek the given partitions.
	//       This requires ZMSCORE support within miniredis for testing;  we need
	//       to add this prior to implementation.

	unix := until.Unix()
	encoded, err := scripts["queue/partitionPeek"].Run(
		ctx,
		q.r,
		[]string{
			q.kg.PartitionIndex(),
			q.kg.PartitionItem(),
		},
		unix,
		limit,
	).StringSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking partition items: %w", err)
	}

	weights := []float64{}
	items := make([]*QueuePartition, len(encoded))

	ignored := 0
	for n, i := range encoded {
		item := &QueuePartition{}
		if err = json.Unmarshal([]byte(i), item); err != nil {
			return nil, fmt.Errorf("error reading partition item: %w", err)
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
func (q *queue) PartitionRequeue(ctx context.Context, queueName string, at time.Time) error {
	keys := []string{
		q.kg.PartitionItem(),
		q.kg.PartitionIndex(),
		q.kg.PartitionMeta(queueName),
		q.kg.QueueIndex(queueName),
		q.kg.QueueItem(),
	}
	status, err := scripts["queue/partitionRequeue"].Run(
		ctx,
		q.r,
		keys,

		queueName,
		at.Unix(),
		QueuePeekMax,
	).Int64()
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

	keys := []string{q.kg.PartitionItem()}
	status, err := scripts["queue/partitionReprioritize"].Run(
		ctx,
		q.r,
		keys,
		queueName,
		priority,
	).Int64()
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

// LeaseSequential allows a worker to lease sequential processing.  If leased, this allows a worker
// to peek partitions sequentially.  Leasing this key works similar to leasing partitions or queue items:
//
// - If the key isn't leased, a new lease is accepted.
// - If the lease is expired, a new lease is accepted.
// - If the key is leased, you must pass in the existing lease ID to renew the lease.  Mismatches do not
//   grant a lease.
//
// This returns the new lease ID on success.
func (q *queue) LeaseSequential(ctx context.Context, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	if duration > SequentialLeaseMax {
		return nil, ErrSequentialLeaseExceedsLimits
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

	status, err := scripts["queue/sequentialLease"].Run(
		ctx,
		q.r,
		[]string{q.kg.Sequential()},
		now.UnixMilli(),
		newLeaseID.String(),
		existing,
	).Int64()
	if err != nil {
		return nil, fmt.Errorf("error claiming sequential lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrSequentialAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response claiming sequential lease: %d", status)
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
