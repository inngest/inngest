package constraintapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
)

// ConstraintDebugger provides debug-only methods for inspecting constraint state.
type ConstraintDebugger interface {
	// GetAccountConcurrency builds an account concurrency constraint and runs Check(),
	// returning the number of in-progress items for the account.
	GetAccountConcurrency(ctx context.Context, accountID uuid.UUID) (int, error)

	// GetFunctionConcurrency builds a function concurrency constraint and runs Check(),
	// returning the number of in-progress items for the function.
	GetFunctionConcurrency(ctx context.Context, accountID uuid.UUID, functionID uuid.UUID) (int, error)

	// CountAccountLeases returns the number of items in the account's leaseq zset.
	CountAccountLeases(ctx context.Context, accountID uuid.UUID) (int, error)

	// CountAccounts returns the number of accounts in the top-level scavenger zset.
	CountAccounts(ctx context.Context) (int, error)
}

func (r *redisCapacityManager) GetAccountConcurrency(ctx context.Context, accountID uuid.UUID) (int, error) {
	resp, _, internalErr := r.Check(ctx, &CapacityCheckRequest{
		AccountID: accountID,
		EnvID:     accountID, // Use accountID as envID for debug-only account-level check
		Configuration: ConstraintConfig{
			Concurrency: ConcurrencyConfig{
				AccountConcurrency: consts.DefaultConcurrencyLimit,
			},
		},
		Constraints: []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeAccount,
				},
			},
		},
	})
	if internalErr != nil {
		return 0, fmt.Errorf("check failed: %w", internalErr)
	}

	if len(resp.Usage) > 0 {
		return resp.Usage[0].Used, nil
	}

	return 0, nil
}

func (r *redisCapacityManager) GetFunctionConcurrency(ctx context.Context, accountID uuid.UUID, functionID uuid.UUID) (int, error) {
	resp, _, internalErr := r.Check(ctx, &CapacityCheckRequest{
		AccountID:  accountID,
		EnvID:      accountID, // Use accountID as envID for debug-only function-level check
		FunctionID: functionID,
		Configuration: ConstraintConfig{
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: consts.DefaultConcurrencyLimit,
			},
		},
		Constraints: []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
	})
	if internalErr != nil {
		return 0, fmt.Errorf("check failed: %w", internalErr)
	}

	if len(resp.Usage) > 0 {
		return resp.Usage[0].Used, nil
	}

	return 0, nil
}

func (r *redisCapacityManager) CountAccountLeases(ctx context.Context, accountID uuid.UUID) (int, error) {
	cmd := r.client.B().Zcard().Key(r.keyAccountLeases(accountID)).Build()
	count, err := r.client.Do(ctx, cmd).ToInt64()
	if err != nil {
		return 0, fmt.Errorf("zcard account leases: %w", err)
	}
	return int(count), nil
}

func (r *redisCapacityManager) CountAccounts(ctx context.Context) (int, error) {
	cmd := r.client.B().Zcard().Key(r.keyScavengerShard()).Build()
	count, err := r.client.Do(ctx, cmd).ToInt64()
	if err != nil {
		return 0, fmt.Errorf("zcard scavenger shard: %w", err)
	}
	return int(count), nil
}
