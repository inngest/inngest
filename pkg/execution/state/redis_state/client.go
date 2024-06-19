package redis_state

import (
	"context"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

type ShardedClient struct {
	kg ShardedKeyGenerator
	r  rueidis.Client
}

func (s *ShardedClient) newScript(ctx context.Context, events, metadataByt, stepsByt []byte, input state.Input) (int64, error) {
	args, err := StrSlice([]any{
		events,
		metadataByt,
		stepsByt,
	})
	if err != nil {
		return 0, err
	}

	status, err := scripts["new"].Exec(
		ctx,
		s.r,
		[]string{
			s.kg.Idempotency(ctx, input.Identifier),
			s.kg.Events(ctx, input.Identifier),
			s.kg.RunMetadata(ctx, input.Identifier.RunID),
			s.kg.Actions(ctx, input.Identifier),
		},
		args,
	).AsInt64()

	return status, err
}

func (s *ShardedClient) updateMetadataScript(ctx context.Context, input []string, runID ulid.ULID) (int64, error) {
	status, err := scripts["updateMetadata"].Exec(
		ctx,
		s.r,
		[]string{
			s.kg.RunMetadata(ctx, runID),
		},
		input,
	).AsInt64()
	return status, err
}

func NewShardedClient(r rueidis.Client) *ShardedClient {
	return &ShardedClient{
		kg: newShardedKeyGenerator(),
		r:  r,
	}
}

func (s *ShardedClient) KeyGenerator() ShardedKeyGenerator {
	return s.kg
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
