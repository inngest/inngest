package singleton

import (
	"context"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
)

func New(ctx context.Context, r *redis_state.QueueClient) Singleton {
	return &redisStore{r: r}
}

type redisStore struct {
	r *redis_state.QueueClient
}

func (r *redisStore) Singleton(ctx context.Context, key string, s inngest.Singleton) (bool, error) {
	return singleton(ctx, r, key, s)
}

func (r *redisStore) Exists(ctx context.Context, key string) (bool, error) {
	key = r.r.KeyGenerator().SingletonKey(&queue.Singleton{Key: key})

	client := r.r.Client()
	// result will be either 0 or 1
	result, err := r.r.Client().Do(ctx, client.B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}
