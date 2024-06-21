package redis_state

import (
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

type ShardedClient struct {
	kg        ShardedKeyGenerator
	shardedRc rueidis.Client
	u         *UnshardedClient
}

func NewShardedClient(u *UnshardedClient, r rueidis.Client) *ShardedClient {
	return &ShardedClient{
		u:         u,
		kg:        newShardedKeyGenerator(),
		shardedRc: r,
	}
}

func (s *ShardedClient) KeyGenerator() ShardedKeyGenerator {
	return s.kg
}

func (s *ShardedClient) Client(runID ulid.ULID) rueidis.Client {
	if s.KeyGenerator().IsSharded(runID) {
		return s.shardedRc
	}
	return s.u.Client()
}

type UnshardedClient struct {
	kg          UnshardedKeyGenerator
	unshardedRc rueidis.Client
}

func (u *UnshardedClient) KeyGenerator() UnshardedKeyGenerator {
	return u.kg
}

func (u *UnshardedClient) Client() rueidis.Client {
	return u.unshardedRc
}

func NewUnshardedClient(r rueidis.Client) *UnshardedClient {
	return &UnshardedClient{
		kg:          newUnshardedKeyGenerator(),
		unshardedRc: r,
	}
}
