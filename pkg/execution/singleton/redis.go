package singleton

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const getAndDeleteLua = `
local v = redis.call('get', KEYS[1])
if v == false then
  return nil
end
redis.call('del', KEYS[1])
return v
`

func New(ctx context.Context, r *redis_state.QueueClient) Singleton {
	return &redisStore{r: r,
		getAndDelScript: rueidis.NewLuaScript(getAndDeleteLua),
	}
}

type redisStore struct {
	r               *redis_state.QueueClient
	getAndDelScript *rueidis.Lua
}

func (r *redisStore) HandleSingleton(ctx context.Context, key string, s inngest.Singleton) (*ulid.ULID, error) {
	return singleton(ctx, r, key, s)
}

func (r *redisStore) ReleaseSingleton(ctx context.Context, key string) (*ulid.ULID, error) {
	redisKey := r.r.KeyGenerator().SingletonKey(&queue.Singleton{Key: key})
	client := r.r.Client()

	val, err := r.getAndDelScript.Exec(ctx, client, []string{redisKey}, nil).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, err
	}

	runID, err := ulid.Parse(val)
	if err != nil {
		return nil, err
	}

	return &runID, nil
}

func (r *redisStore) GetCurrentRunID(ctx context.Context, key string) (*ulid.ULID, error) {
	key = r.r.KeyGenerator().SingletonKey(&queue.Singleton{Key: key})

	client := r.r.Client()

	val, err := r.r.Client().Do(ctx, client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, err
	}

	runID, err := ulid.Parse(val)
	if err != nil {
		return nil, err
	}

	return &runID, nil
}
