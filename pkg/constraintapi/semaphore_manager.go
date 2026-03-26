package constraintapi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

const (
	semaphoreIdempotencyTTL = 60 * time.Second
)

// SemaphoreManager provides underlying internal APIs for managing semaphores.  these are required because,
// unlike other constraints, semaphores can be manually adjusted:  the capacity must be adjusted when new
// workers come online, and for fn concurrency Release is called manually.
type SemaphoreManager interface {
	// SetCapacity sets the total capacity for a named semaphore.
	SetCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) error

	// AdjustCapacity atomically adjusts capacity by delta (e.g., +N on worker connect, -N on disconnect).
	AdjustCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) error

	// GetCapacity returns current capacity and usage for a named semaphore.
	GetCapacity(ctx context.Context, accountID uuid.UUID, name, usageValue string) (capacity int64, usage int64, err error)

	// ReleaseSemaphore decrements the usage counter for a manual-release semaphore.
	// Called on run finalization for function concurrency. Must be idempotent.
	ReleaseSemaphore(ctx context.Context, accountID uuid.UUID, name, usageValue, idempotencyKey string, weight int64) error
}

type redisSemaphoreManager struct {
	client rueidis.Client
}

func NewRedisSemaphoreManager(client rueidis.Client) SemaphoreManager {
	return &redisSemaphoreManager{client: client}
}

func semaphoreCapacityKey(accountID uuid.UUID, name string) string {
	return fmt.Sprintf("{cs}:%s:sem:%s:cap", accountScope(accountID), name)
}

func semaphoreUsageKey(accountID uuid.UUID, name, usageValue string) string {
	return fmt.Sprintf("{cs}:%s:sem:%s:usage:%s", accountScope(accountID), name, usageValue)
}

func semaphoreIdempotencyKey(accountID uuid.UUID, op, idempotencyKey string) string {
	return fmt.Sprintf("{cs}:%s:sem:ik:%s:%s", accountScope(accountID), op, idempotencyKey)
}

func (m *redisSemaphoreManager) SetCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) error {
	keys := []string{
		semaphoreCapacityKey(accountID, name),
		semaphoreIdempotencyKey(accountID, "setcap", idempotencyKey),
	}
	args := []string{
		fmt.Sprintf("%d", capacity),
		fmt.Sprintf("%d", int(semaphoreIdempotencyTTL.Seconds())),
	}

	return scripts["semaphore_set_capacity"].Exec(ctx, m.client, keys, args).Error()
}

func (m *redisSemaphoreManager) AdjustCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) error {
	keys := []string{
		semaphoreCapacityKey(accountID, name),
		semaphoreIdempotencyKey(accountID, "adjcap", idempotencyKey),
	}
	args := []string{
		fmt.Sprintf("%d", delta),
		fmt.Sprintf("%d", int(semaphoreIdempotencyTTL.Seconds())),
	}

	return scripts["semaphore_adjust_capacity"].Exec(ctx, m.client, keys, args).Error()
}

func (m *redisSemaphoreManager) GetCapacity(ctx context.Context, accountID uuid.UUID, name, usageValue string) (int64, int64, error) {
	capKey := semaphoreCapacityKey(accountID, name)
	usageKey := semaphoreUsageKey(accountID, name, usageValue)

	results := m.client.DoMulti(ctx,
		m.client.B().Get().Key(capKey).Build(),
		m.client.B().Get().Key(usageKey).Build(),
	)

	capacity, err := results[0].AsInt64()
	if rueidis.IsRedisNil(err) {
		capacity = 0
	} else if err != nil {
		return 0, 0, fmt.Errorf("could not get semaphore capacity: %w", err)
	}

	usage, err := results[1].AsInt64()
	if rueidis.IsRedisNil(err) {
		usage = 0
	} else if err != nil {
		return 0, 0, fmt.Errorf("could not get semaphore usage: %w", err)
	}

	return capacity, usage, nil
}

func (m *redisSemaphoreManager) ReleaseSemaphore(ctx context.Context, accountID uuid.UUID, name, usageValue, idempotencyKey string, weight int64) error {
	keys := []string{
		semaphoreUsageKey(accountID, name, usageValue),
		semaphoreIdempotencyKey(accountID, "rel", idempotencyKey),
	}
	args := []string{
		fmt.Sprintf("%d", weight),
		fmt.Sprintf("%d", int(semaphoreIdempotencyTTL.Seconds())),
	}

	return scripts["semaphore_release"].Exec(ctx, m.client, keys, args).Error()
}
