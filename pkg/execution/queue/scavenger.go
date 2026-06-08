package queue

import (
	"context"
	"math/rand"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

func NewScavengerRole(opts ...QueueRoleOpt) QueueRole {
	return newQueueRole(QueueRoleScavenger, RoleLeaseDuration, 30*time.Second, func(ctx context.Context, shard QueueShard) error {
		count, err := shard.Scavenge(ctx, ScavengePeekSize)
		if err != nil {
			return err
		}
		if count > 0 {
			logger.StdlibLogger(ctx).Info("scavenged lost jobs", "len", count)
		}
		return nil
	}, nil, opts...)
}

func RandomScavengeOffset(seed int64, count int64, limit int) int64 {
	// only apply random offset if there are more total items to scavenge than the limit
	if count > int64(limit) {
		r := rand.New(rand.NewSource(seed))

		// the result of count-limit must be greater than 0 as we have already checked count > limit
		// we increase the argument by 1 to make the highest possible index accessible
		// example: for count = 9, limit = 3, we want to access indices 0 through 6, not 0 through 5
		return r.Int63n(count - int64(limit) + 1)
	}

	return 0
}
