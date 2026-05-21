package redis_state

import (
	"context"
	"fmt"
	"strconv"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// DebounceCreate implements queue.DebounceOperations.
func (q *queue) DebounceCreate(ctx context.Context, scope osqueue.Scope, key string, debounceID ulid.ULID, item []byte, ttl time.Duration) (*ulid.ULID, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	out, err := scripts["debounce/newDebounce"].Exec(
		ctx,
		client,
		[]string{kg.DebouncePointer(ctx, scope.FunctionID, key), kg.Debounce(ctx)},
		[]string{debounceID.String(), string(item), strconv.Itoa(int(ttl.Seconds()))},
	).ToString()
	if err != nil {
		return nil, fmt.Errorf("error creating debounce: %w", err)
	}

	if out == "0" {
		return nil, nil
	}

	existingID, err := ulid.Parse(out)
	if err != nil {
		return nil, fmt.Errorf("unknown new debounce return value: %s", out)
	}
	return &existingID, nil
}

// DebounceUpdate implements queue.DebounceOperations.
func (q *queue) DebounceUpdate(
	ctx context.Context,
	scope osqueue.Scope,
	key string,
	debounceID ulid.ULID,
	item []byte,
	ttl time.Duration,
	jobID string,
	now time.Time,
	eventTimestamp int64,
) (int64, osqueue.DebounceUpdateStatus, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	out, err := scripts["debounce/updateDebounce"].Exec(
		ctx,
		client,
		[]string{
			kg.DebouncePointer(ctx, scope.FunctionID, key),
			kg.Debounce(ctx),
			kg.QueueItem(),
		},
		[]string{
			debounceID.String(),
			string(item),
			strconv.Itoa(int(ttl.Seconds())),
			jobID,
			strconv.Itoa(int(now.UnixMilli())),
			strconv.Itoa(int(eventTimestamp)),
		},
	).AsInt64()
	if err != nil {
		return 0, 0, fmt.Errorf("error updating debounce: %w", err)
	}

	switch out {
	case -1:
		return 0, osqueue.DebounceUpdateInProgress, nil
	case -2:
		return 0, osqueue.DebounceUpdateOutOfOrder, nil
	case -3:
		return 0, osqueue.DebounceUpdateNotFound, nil
	default:
		return out, osqueue.DebounceUpdateOK, nil
	}
}

// DebounceStartExecution implements queue.DebounceOperations.
func (q *queue) DebounceStartExecution(ctx context.Context, scope osqueue.Scope, key string, newDebounceID, debounceID ulid.ULID) (osqueue.DebounceStartStatus, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	res, err := scripts["debounce/start"].Exec(
		ctx,
		client,
		[]string{
			kg.DebouncePointer(ctx, scope.FunctionID, key),
			kg.DebounceMigrating(ctx),
		},
		[]string{
			newDebounceID.String(),
			debounceID.String(),
		},
	).AsInt64()
	if err != nil {
		return 0, err
	}

	switch res {
	case -1:
		return osqueue.DebounceStartMigrating, nil
	case 0, 1:
		return osqueue.DebounceStartStarted, nil
	default:
		return 0, fmt.Errorf("invalid status returned when starting debounce: %d", res)
	}
}

// DebouncePrepareMigration implements queue.DebounceOperations.
func (q *queue) DebouncePrepareMigration(ctx context.Context, scope osqueue.Scope, key string, fakeDebounceID ulid.ULID) (*ulid.ULID, int64, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	out, err := scripts["debounce/prepareMigration"].Exec(
		ctx,
		client,
		[]string{
			kg.DebouncePointer(ctx, scope.FunctionID, key),
			kg.Debounce(ctx),
			kg.DebounceMigrating(ctx),
		},
		[]string{fakeDebounceID.String()},
	).ToAny()
	if err != nil {
		return nil, 0, fmt.Errorf("error running prepareMigration script: %w", err)
	}

	returnedSet, ok := out.([]any)
	if !ok {
		return nil, 0, fmt.Errorf("expected to receive one or more set items")
	}
	if len(returnedSet) < 1 {
		return nil, 0, fmt.Errorf("expected at least one item")
	}

	status, ok := returnedSet[0].(int64)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected return value, expected status int")
	}

	if status == 0 {
		return nil, 0, nil
	}

	if status != 1 || len(returnedSet) < 2 {
		return nil, 0, fmt.Errorf("expected status 1 with at least two return items")
	}

	debounceIdStr, ok := returnedSet[1].(string)
	if !ok {
		return nil, 0, fmt.Errorf("expected debounceID as second item")
	}

	existingID, err := ulid.Parse(debounceIdStr)
	if err != nil {
		return nil, 0, fmt.Errorf("unknown debounce ID return value: %s", debounceIdStr)
	}

	var timeoutMillis int64
	if len(returnedSet) == 3 {
		timeoutMillis, ok = returnedSet[2].(int64)
		if !ok {
			return nil, 0, fmt.Errorf("expected timeout int")
		}
	}

	return &existingID, timeoutMillis, nil
}

// DebounceGetItem implements queue.DebounceOperations.
func (q *queue) DebounceGetItem(ctx context.Context, scope osqueue.Scope, debounceID ulid.ULID) ([]byte, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	cmd := client.B().Hget().Key(kg.Debounce(ctx)).Field(debounceID.String()).Build()
	byt, err := client.Do(ctx, cmd).AsBytes()
	if rueidis.IsRedisNil(err) {
		return nil, osqueue.ErrDebounceNotFound
	}
	if err != nil {
		return nil, err
	}
	return byt, nil
}

// DebounceDeleteItems implements queue.DebounceOperations.
func (q *queue) DebounceDeleteItems(ctx context.Context, scope osqueue.Scope, debounceIDs ...ulid.ULID) error {
	if len(debounceIDs) == 0 {
		return nil
	}
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	fields := make([]string, len(debounceIDs))
	for i, id := range debounceIDs {
		fields[i] = id.String()
	}
	cmd := client.B().Hdel().Key(kg.Debounce(ctx)).Field(fields...).Build()
	err := client.Do(ctx, cmd).Error()
	if rueidis.IsRedisNil(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error removing debounce: %w", err)
	}
	return nil
}

// DebounceDeleteMigratingFlag implements queue.DebounceOperations.
func (q *queue) DebounceDeleteMigratingFlag(ctx context.Context, scope osqueue.Scope, debounceID ulid.ULID) error {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	cmd := client.B().Hdel().Key(kg.DebounceMigrating(ctx)).Field(debounceID.String()).Build()
	err := client.Do(ctx, cmd).Error()
	if rueidis.IsRedisNil(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error removing debounce migrating flag: %w", err)
	}
	return nil
}

// DebounceGetPointer implements queue.DebounceOperations.
func (q *queue) DebounceGetPointer(ctx context.Context, scope osqueue.Scope, key string) (string, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	val, err := client.Do(ctx, client.B().Get().Key(kg.DebouncePointer(ctx, scope.FunctionID, key)).Build()).ToString()
	if rueidis.IsRedisNil(err) {
		return "", osqueue.ErrDebounceNotFound
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// DebounceDeletePointer implements queue.DebounceOperations.
func (q *queue) DebounceDeletePointer(ctx context.Context, scope osqueue.Scope, key string) error {
	client := q.RedisClient.Client()
	kg := q.RedisClient.DebounceKeyGenerator()

	err := client.Do(ctx, client.B().Del().Key(kg.DebouncePointer(ctx, scope.FunctionID, key)).Build()).Error()
	if err != nil && !rueidis.IsRedisNil(err) {
		return fmt.Errorf("error deleting debounce pointer: %w", err)
	}
	return nil
}
