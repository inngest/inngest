package connect

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/rueidis"
	"time"
)

type ConnectionStateManager interface {
	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
}

type redisConnectionStateManager struct {
	client rueidis.Client
}

var ErrIdempotencyKeyExists = fmt.Errorf("idempotency key exists")

func (r redisConnectionStateManager) SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error {
	idempotencyKey := fmt.Sprintf("{%s}:idempotency:%s", appId, requestId)
	res := r.client.Do(
		ctx,
		r.client.B().Set().Key(idempotencyKey).Value("1").Nx().Ex(time.Second*10).Build(),
	)
	set, err := res.AsBool()
	if (err == nil || rueidis.IsRedisNil(err)) && !set {
		return ErrIdempotencyKeyExists
	}
	if err != nil {
		return fmt.Errorf("could not set idempotency key: %w", err)
	}

	return nil
}

func NewRedisConnectionStateManager(client rueidis.Client) ConnectionStateManager {
	return &redisConnectionStateManager{
		client: client,
	}
}
