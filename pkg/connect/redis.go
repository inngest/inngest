package connect

import "github.com/redis/rueidis"

type redisConnectionStateManager struct {
	client rueidis.Client
}

func NewRedisConnectionStateManager(client rueidis.Client) ConnectionStateManager {
	return &redisConnectionStateManager{
		client: client,
	}
}
