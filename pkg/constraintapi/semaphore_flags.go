package constraintapi

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// UseAccountSemaphoreFn gates account-scoped semaphore concurrency for an account.
type UseAccountSemaphoreFn func(ctx context.Context, accountID uuid.UUID) (enable bool)

// AccountSemaphoreEnabled is the package-wide gate for account-scoped semaphore concurrency.
// Nil means account semaphores are disabled.
var AccountSemaphoreEnabled UseAccountSemaphoreFn

// AccountSemaphoreConcurrencyEnabled returns whether account-scoped semaphore concurrency is enabled.
func AccountSemaphoreConcurrencyEnabled(ctx context.Context, accountID uuid.UUID) bool {
	return AccountSemaphoreEnabled != nil && AccountSemaphoreEnabled(ctx, accountID)
}

func FilterDisabledAccountSemaphores(
	ctx context.Context,
	accountID uuid.UUID,
	semaphores []Semaphore,
) []Semaphore {
	// Account semaphores have their own rollout gate; all other semaphores stay active.
	if AccountSemaphoreConcurrencyEnabled(ctx, accountID) {
		return semaphores
	}

	return slices.DeleteFunc(slices.Clone(semaphores), Semaphore.IsAccountConcurrency)
}

func FilterDisabledAccountSemaphoreConstraints(
	ctx context.Context,
	accountID uuid.UUID,
	constraints []ConstraintItem,
) []ConstraintItem {
	// Existing queue items may still carry acct: semaphores after the flag is disabled.
	if AccountSemaphoreConcurrencyEnabled(ctx, accountID) {
		return constraints
	}

	return slices.DeleteFunc(slices.Clone(constraints), func(constraint ConstraintItem) bool {
		return constraint.Kind == ConstraintKindSemaphore && constraint.Semaphore.IsAccountConcurrency()
	})
}

func FilterDisabledAccountSemaphoreConfig(
	ctx context.Context,
	accountID uuid.UUID,
	config ConstraintConfig,
) ConstraintConfig {
	config.Semaphores = FilterDisabledAccountSemaphores(ctx, accountID, config.Semaphores)
	return config
}

func filterDisabledAccountSemaphoreRequest(
	ctx context.Context,
	accountID uuid.UUID,
	constraints []ConstraintItem,
	config ConstraintConfig,
) ([]ConstraintItem, ConstraintConfig, bool) {
	filteredConstraints := FilterDisabledAccountSemaphoreConstraints(ctx, accountID, constraints)
	filteredConfig := FilterDisabledAccountSemaphoreConfig(ctx, accountID, config)
	changed := len(filteredConstraints) != len(constraints) ||
		len(filteredConfig.Semaphores) != len(config.Semaphores)

	return filteredConstraints, filteredConfig, changed
}
