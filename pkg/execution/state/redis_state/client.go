package redis_state

import (
	"github.com/redis/rueidis"
)

type ShardedClient struct {
	kg ShardedKeyGenerator
	r  rueidis.Client
}

func NewShardedClient(r rueidis.Client) *ShardedClient {
	return &ShardedClient{
		kg: newShardedKeyGenerator(),
		r:  r,
	}
}

type UnshardedClient struct {
	kg UnshardedKeyGenerator
	r  rueidis.Client
}

func (u *UnshardedClient) KeyGenerator() UnshardedKeyGenerator {
	return u.kg
}

func NewUnshardedClient(r rueidis.Client) *UnshardedClient {
	return &UnshardedClient{
		kg: newUnshardedKeyGenerator(),
		r:  r,
	}
}
