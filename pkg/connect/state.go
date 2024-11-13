package connect

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/redis/rueidis"
)

type ConnectionStateManager interface {
	ConnectGatewayLifecycleListener

	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
}

type redisConnectionStateManager struct {
	client rueidis.Client
}

var ErrIdempotencyKeyExists = fmt.Errorf("idempotency key exists")

func NewRedisConnectionStateManager(client rueidis.Client) ConnectionStateManager {
	return &redisConnectionStateManager{
		client: client,
	}
}

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

//
// Lifecycle hooks
//

func (r *redisConnectionStateManager) OnConnected(ctx context.Context, data *connect.SDKConnectRequestData) {
}

func (r *redisConnectionStateManager) OnAuthenticated(ctx context.Context, auth *AuthResponse) {}

func (r *redisConnectionStateManager) OnSynced(ctx context.Context) {}

func (r *redisConnectionStateManager) OnDisconnected(ctx context.Context) {}
