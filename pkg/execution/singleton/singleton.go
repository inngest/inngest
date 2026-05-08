package singleton

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

var (
	ErrEvaluatingSingletonExpression = fmt.Errorf("singleton expression evaluation failed")
	ErrNotASingleton                 = fmt.Errorf("singleton expression resolved to false")
)

type Singleton interface {
	HandleSingleton(ctx context.Context, key string, c inngest.Singleton, accountID uuid.UUID) (*ulid.ULID, error)
}

func New(ctx context.Context, shards queue.ShardRegistry) Singleton {
	return &store{shards: shards}
}

type store struct {
	shards queue.ShardRegistry
}

func (s *store) HandleSingleton(ctx context.Context, key string, cfg inngest.Singleton, accountID uuid.UUID) (*ulid.ULID, error) {
	return singleton(ctx, s.shards, key, cfg, accountID)
}

// SingletonKey returns the singleton key given a function ID, singleton config,
// and incoming event data.
func SingletonKey(ctx context.Context, id uuid.UUID, c inngest.Singleton, evt map[string]any) (string, error) {
	if c.Key == nil {
		return id.String(), nil
	}
	eval, err := expressions.NewExpressionEvaluator(ctx, *c.Key)
	if err != nil {
		return "", ErrEvaluatingSingletonExpression
	}
	res, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": evt}))
	if err != nil {
		return "", ErrEvaluatingSingletonExpression
	}
	if v, ok := res.(bool); ok && !v {
		return "", ErrNotASingleton
	}

	return hash(res, id), nil
}

func hash(res any, id uuid.UUID) string {
	sum := util.XXHash(res)
	return fmt.Sprintf("%s-%s", id, sum)
}

//	singleton retrieves or releases a singleton lock based on the given mode.
//
// - If the mode is SingletonModeSkip, it returns the currently held run ID without modifying the lock.
//
// - If the mode is SingletonModeCancel, it attempts to release the lock and returns the run ID that was released.
func singleton(ctx context.Context, shards queue.ShardRegistry, key string, s inngest.Singleton, accountID uuid.UUID) (*ulid.ULID, error) {
	shard, err := shards.Resolve(ctx, accountID, nil)
	if err != nil {
		return nil, err
	}
	switch s.Mode {
	case enums.SingletonModeSkip:
		return shard.SingletonGetRunID(ctx, key)
	case enums.SingletonModeCancel:
		return shard.SingletonReleaseRunID(ctx, key)
	default:
		return nil, fmt.Errorf("singleton mode %d not implemented", s.Mode)
	}
}
