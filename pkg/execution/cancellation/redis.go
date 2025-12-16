package cancellation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/redis/rueidis"
)

const (
	DefaultPrefix = "{cancel}"
	pkgName       = "execution.cancellation"
)

// NewRedisWriter writes cancellations to Redis.
func NewRedisWriter(r rueidis.Client, q queue.Producer, prefix string) cqrs.CancellationWriter {
	if prefix == "" {
		prefix = DefaultPrefix
	}
	return redisReadWriter{r: r, q: q, prefix: prefix}
}

// NewRedisReader loads cancellations from Redis.
func NewRedisReader(r rueidis.Client, prefix string) Reader {
	if prefix == "" {
		prefix = DefaultPrefix
	}
	return redisReadWriter{r: r, q: nil, prefix: prefix}
}

type redisReadWriter struct {
	r      rueidis.Client
	q      queue.Producer
	prefix string
}

type redisWrapper struct {
	Version      int               `json:"v"`
	Cancellation cqrs.Cancellation `json:"c"`
}

func (r redisReadWriter) CreateCancellation(ctx context.Context, c cqrs.Cancellation) error {
	if c.ID.IsZero() {
		return fmt.Errorf("A cancellation ID must be created before writing")
	}

	// TODO: Use msgpack, capnproto, protobuf, or some other fast parser here.
	byt, err := json.Marshal(redisWrapper{Version: 1, Cancellation: c})
	if err != nil {
		return err
	}

	key := r.key(c.WorkspaceID, c.FunctionID)

	// TODO: We want to store overlapping indexes of cancellations
	// using zsets which store beginning and end times of when cancellations are valid.
	//
	// This lets us query for valid cancellations using a fast lookup, instead of
	// loading them all.
	//
	// For now, though, we're adding cancellations to a hashmap of each given workspace/
	// function combination.
	cmd := r.r.B().Hset().Key(key).FieldValue().FieldValue(c.ID.String(), string(byt)).Build()
	if err := r.r.Do(ctx, cmd).Error(); err != nil {
		return err
	}

	// Enqueue to system queue for eager cancellation
	queueName := queue.KindCancel
	maxAttempts := consts.MaxRetries + 1
	jobID := c.ID.String()
	err = r.q.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		GroupID:     uuid.New().String(),
		WorkspaceID: c.WorkspaceID,
		Kind:        queue.KindCancel,
		Identifier: state.Identifier{
			AccountID:   c.AccountID,
			WorkspaceID: c.WorkspaceID,
			AppID:       c.AppID,
			WorkflowID:  c.FunctionID,
			Key:         fmt.Sprintf("cancel:%s", c.ID),
		},
		MaxAttempts: &maxAttempts,
		Payload:     c,
		QueueName:   &queueName,
	}, time.Now(), queue.EnqueueOpts{})
	if err == redis_state.ErrQueueItemExists {
		return nil
	}
	return err
}

func (r redisReadWriter) DeleteCancellation(ctx context.Context, c cqrs.Cancellation) error {
	key := r.key(c.WorkspaceID, c.FunctionID)
	cmd := r.r.B().Hdel().Key(key).Field(c.ID.String()).Build()
	err := r.r.Do(ctx, cmd).Error()
	if err == nil || rueidis.IsRedisNil(err) {
		return nil
	}
	return fmt.Errorf("error deleting cancellation: %w", err)
}

func (r redisReadWriter) ReadAt(ctx context.Context, wsID uuid.UUID, fnID uuid.UUID, at time.Time) ([]cqrs.Cancellation, error) {
	// XXX: Cancellations need to be fast.  They're loaded in the critical path on each step
	// execution so that cancellations are immedaite.
	//
	// Ideally, we'd store these in-memory for instant loading.  We'd need to notify
	// notify executors when cancellations have been modified, allowing us to only fetch
	// items from the datastore when necessary.  Our executors are shared-nothing and do
	// not communicate amongst each other;  this needs us to start centralizing executors
	// to a messaging system (NATS) and ensuring heartbeats for executors so that we can
	// continuously communicate with them and prevent them from executing in the case of
	// networking errors.
	//
	// Our state store is fast enough to perform lookups to make this quick, and this doesn't
	// increase latency beyond a few milliseconds.
	key := r.key(wsID, fnID)

	start := time.Now()

	cmd := r.r.B().Hgetall().Key(key).Build()
	all, err := r.r.Do(ctx, cmd).AsMap()
	if err != nil {
		return nil, err
	}

	{
		dur := time.Since(start)

		tags := map[string]any{}
		if len(all) > 100 {
			tags["workspace_id"] = wsID
			tags["fn_id"] = fnID
		}

		metrics.HistogramCancellationReadSize(ctx, int64(len(all)), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    tags,
		})

		metrics.HistogramCancellationReadDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    tags,
		})
	}

	result := []cqrs.Cancellation{}
	for _, item := range all {
		found := &redisWrapper{}
		if err := item.DecodeJSON(found); err != nil {
			return nil, err
		}
		// XXX: Right now there's only one version of a cancellation stored in
		// the state store, so we don't need to handle found.Version differently.
		c := found.Cancellation
		if at.After(c.StartedBefore) {
			// This cancellation is only for functions prior to the given point
			// in time, so ignore.
			continue
		}
		if c.StartedAfter != nil && at.Before(*c.StartedAfter) {
			continue
		}
		result = append(result, c)
	}

	return result, nil
}

func (r redisReadWriter) key(wsID uuid.UUID, fnID uuid.UUID) string {
	// We currently don't hash the workspace or function here.  It would cut down
	// on key size, but we store cancellations in a map anyway.
	return fmt.Sprintf("%s:%s:%s", r.prefix, wsID, fnID)
}
