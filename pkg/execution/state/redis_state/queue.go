package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"golang.org/x/exp/rand"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"gonum.org/v1/gonum/stat/sampleuv"
)

const (
	PartitionSelectionMax  int64 = 20
	PartitionPeekMax       int64 = PartitionSelectionMax * 3
	PartitionLeaseDuration       = 2 * time.Second
	QueueSelectionMax      int64 = 50
	QueuePeekMax           int64 = QueueSelectionMax * 3
	QueueLeaseDuration           = 10 * time.Second

	PriorityMax uint = 0
	PriorityMin uint = 9
)

var (
	ErrQueueItemExists        = fmt.Errorf("queue item already exists")
	ErrQueueItemNotFound      = fmt.Errorf("queue item not found")
	ErrQueueItemAlreadyLeased = fmt.Errorf("queue item already leased")
	ErrPriorityTooLow         = fmt.Errorf("priority is too low")
	ErrWeightedSampleRead     = fmt.Errorf("error reading from weighted sample")
	ErrPartitionAlreadyLeased = fmt.Errorf("partition already leased")
)

var (
	source rand.Source
	r      *rand.Rand
)

func init() {
	source = rand.NewSource(uint64(time.Now().UnixNano()))
	r = rand.New(source)
}

// PriorityFinder returns the priority for a given workflow.
type PriorityFinder func(ctx context.Context, workflowID uuid.UUID) uint

type queue struct {
	metrics any
	r       *redis.Client
	pf      PriorityFinder
}

// QueueItem represents an individually queued work scheduled for some time in the
// future.
type QueueItem struct {
	// ID represents a unique identifier for the queue item.  The ULID
	// stores when the QueueItem was created.
	ID          ulid.ULID `json:"id"`
	WorkflowID  uuid.UUID `json:"workflowID"`
	Attempt     int       `json:"attempt"`
	MaxAttempts int       `json:"maxAttempts"`
	// LeaseID is a ULID which embeds a timestamp denoting when the lease expires.
	LeaseID *ulid.ULID `json:"leaseID"`
	// Data represents the enqueued data, eg. the edge to process or the pause
	// to resume.
	Data any `json:"data"`
}

func (q QueueItem) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

// QueuePartition represents an individual queue for a workflow.  It stores the
// time of the earliest job within the workflow.
type QueuePartition struct {
	// QueuePartitionIndex embeds the workflow ID and priority within the
	// actual partition.
	QueuePartitionIndex

	// Earliest refers to the earliest QueueItem time within this partition.
	Earliest time.Time

	// LeaseID represents a lease on this partition.  If the LeaseID is not nil,
	// this partition can be claimed by a shared-nothing worker to work on the
	// queue items within this partition.
	//
	// A lease is shortly held (eg seconds).  It should last long enough for
	// workers to claim QueueItems only.
	LeaseID *ulid.ULID
}

func (q QueuePartition) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

// QueuePartitionIndex is an index for looking up queue partitions by time.  We store
// the priority within the index such that peeking on a queue is fast and can use
// priorities for random sampling.
type QueuePartitionIndex struct {
	WorkflowID uuid.UUID
	Priority   uint
}

func (q QueuePartitionIndex) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

// Enqueue enqueues a QueueItem.  It creates a QueuePartition for the workspace
// if a partition does not exist.
//
// The QueueItem's ID can be a zero UUID;  if the ID is a zero value a new ID
// will be created for the queue item.
func (q queue) Enqueue(ctx context.Context, i QueueItem, at time.Time) (QueueItem, error) {
	if i.ID.Compare(ulid.ULID{}) == 0 {
		i.ID = ulid.MustNew(ulid.Now(), r)
	}

	priority := q.pf(ctx, i.WorkflowID)
	if priority > PriorityMin {
		return i, ErrPriorityTooLow
	}

	qpi := QueuePartitionIndex{i.WorkflowID, priority}

	keys := []string{
		fmt.Sprintf("queue:item:%s", i.ID),             // Queue item
		fmt.Sprintf("queue:sorted:%s", i.WorkflowID),   // Queue sorted set
		fmt.Sprintf("partition:item:%s", i.WorkflowID), // Partition item
		"partition:sorted",                             // Global partition queue
	}
	status, err := redis.NewScript(scripts["queue/enqueue"]).Eval(
		ctx,
		q.r,
		keys,

		i,
		i.ID.String(),
		at.Unix(),
		qpi,
		QueuePartition{
			QueuePartitionIndex: qpi,
			Earliest:            at,
		},
	).Int64()
	if err != nil {
		return i, fmt.Errorf("error leasing pause: %w", err)
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

// Peek takes QueuePeekMax items from a queue.
func (q queue) Peek(ctx context.Context, workflowID uuid.UUID) ([]*QueueItem, error) {
	return nil, nil
}

// Lease temporarily dequeues an item from the queue by obtaining a lease, preventing
// other workers from working on this queue item at the same time.
//
// Obtaining a lease updates the vesting time for the queue item until now() +
// lease duration.
func (q queue) Lease(ctx context.Context, workflowID uuid.UUID, itemID ulid.ULID, duration time.Duration) error {
	// TODO: Add custom throttling here.

	leaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(duration).UTC()), r)
	if err != nil {
		return fmt.Errorf("error generating id: %w", err)
	}

	keys := []string{
		fmt.Sprintf("queue:item:%s", itemID),         // Queue item
		fmt.Sprintf("queue:sorted:%s", workflowID),   // Queue sorted set
		fmt.Sprintf("partition:item:%s", workflowID), // Partition item
	}
	status, err := redis.NewScript(scripts["queue/lease"]).Eval(
		ctx,
		q.r,
		keys,
		leaseID.String(),
		time.Now().UnixMilli(),
	).Int64()
	if err != nil {
		return fmt.Errorf("error leasing pause: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return ErrQueueItemNotFound
	case 2:
		return ErrQueueItemAlreadyLeased
	default:
		return fmt.Errorf("unknown response enqueueing item: %d", status)
	}
}

// ExtendLease extens the lease for a given queue item, given the queue item is currently
// leased with the given ID.  This returns a new lease ID if the lease is successfully ended.
func (q queue) ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	return nil, nil
}

// Dequeue removes an item from the queue entirely.
func (q queue) Dequeue(ctx context.Context, i QueueItem) error {
	return nil
}

// Partition
func (q queue) PartitionLease(ctx context.Context, qpi QueuePartitionIndex) (*QueuePartition, error) {
	// TODO: Fetch partition item in lua, check partition lease, update item and index if available
	return nil, nil
}

// PartitionReprioritize reprioritizes a workflow's QueueItems within the queue.
func (q queue) PartitionReprioritize(ctx context.Context, workflowID uuid.UUID, priority uint) error {
	// TODO: Remove the partition and index, and re-add atomically.
	// We must remove the partition entirely as it's used within a ZSET,
	// and when the structure changes we can no longer update it via ZADD.
	return nil
}

// PartitionPeek returns up to PartitionSelectionMax partition items from the queue. This
// returns the indexes of partitions.
//
// If sequential is set to true this returns partitions in order from earliest to latest
// available lease times. Otherwise, this shuffles all partitions and picks partitions
// randomly, with higher priority partitions more likely to be selected.  This reduces
// lease contention amongst multiple shared-nothing workers.
func (q queue) PartitionPeek(ctx context.Context, sequential bool) ([]*QueuePartitionIndex, error) {
	encoded, err := q.r.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     "partition:sorted",
		Start:   "-inf",
		Stop:    time.Now().Unix(),
		ByScore: true,
		Count:   PartitionPeekMax,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("error peeking partition: %w", err)
	}

	weights := []float64{}
	items := make([]*QueuePartitionIndex, len(encoded))
	for n, i := range encoded {
		item := &QueuePartitionIndex{}
		if err = json.Unmarshal([]byte(i), item); err != nil {
			return nil, fmt.Errorf("error reading partition item: %w", err)
		}
		items[n] = item
		weights = append(weights, float64(10-item.Priority))
	}

	// Some scanners run sequentially, ensuring we always work on the functions with
	// the oldest run at times in order, no matter the priority.
	if sequential {
		n := int(math.Min(float64(len(items)), float64(PartitionSelectionMax)))
		return items[0:n], nil
	}

	if len(items) <= int(PartitionSelectionMax) {
		// Just return them all, shuffled.  We don't need to do priority shuffling
		// or blending as there aren't enough items to pick from.
		rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
		return items, nil
	}

	// Otherwise, we want to weighted shuffle the resulting array random.  This means that
	// many shared nothing scanners can query for outstanding partitions and receive a
	// randomized order favouring higher-priority queue items.  This reduces the chances
	// of contention when leasing.
	w := sampleuv.NewWeighted(weights, r)
	result := make([]*QueuePartitionIndex, PartitionSelectionMax)
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return nil, ErrWeightedSampleRead
		}
		result[n] = items[idx]
	}

	return result, nil
}
