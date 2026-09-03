package constraintapi

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

// TestResolveLimitsMatchesSerialized pins ResolveLimits to the limit, burst
// and period that ToSerializedConstraintItem embeds for the Lua scripts.  the
// memory backend reads limits through ResolveLimits, so a drift here means the
// two backends admit different amounts for the same request.
func TestResolveLimitsMatchesSerialized(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	config := ConstraintConfig{
		FunctionVersion: 3,
		RateLimit: []RateLimitConfig{
			{Scope: enums.RateLimitScopeFn, Limit: 120, Period: 60, KeyExpressionHash: "rl-fn"},
			{Scope: enums.RateLimitScopeAccount, Limit: 7, Period: 60, KeyExpressionHash: "rl-acct"},
			{Scope: enums.RateLimitScopeEnv, Limit: 15, Period: 1, KeyExpressionHash: ""},
		},
		Throttle: []ThrottleConfig{
			{Scope: enums.ThrottleScopeFn, Limit: 10, Burst: 3, Period: 60, KeyExpressionHash: "t-fn"},
			{Scope: enums.ThrottleScopeAccount, Limit: 5, Burst: 0, Period: 1, KeyExpressionHash: ""},
		},
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:     20,
			FunctionConcurrency:    5,
			AccountRunConcurrency:  50,
			FunctionRunConcurrency: 8,
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn, Limit: 2, KeyExpressionHash: "cc-fn"},
				{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount, Limit: 4, KeyExpressionHash: "cc-acct"},
			},
		},
		Semaphores: []Semaphore{{ID: "app:" + uuid.NewString(), Weight: 2}},
	}

	cases := []struct {
		name string
		item ConstraintItem
		want ConstraintLimits
	}{
		{
			name: "rate limit fn",
			item: ConstraintItem{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "rl-fn", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 120, Burst: 12, Period: 60_000_000_000},
		},
		{
			name: "rate limit account non divisible burst",
			item: ConstraintItem{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount, KeyExpressionHash: "rl-acct", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 7, Burst: 0, Period: 60_000_000_000},
		},
		{
			name: "rate limit env without key expression",
			item: ConstraintItem{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv, EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 15, Burst: 1, Period: 1_000_000_000},
		},
		{
			name: "rate limit unknown scope",
			item: ConstraintItem{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "missing", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{},
		},
		{
			name: "throttle fn",
			item: ConstraintItem{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t-fn", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 10, Burst: 3, Period: 60_000},
		},
		{
			name: "throttle account",
			item: ConstraintItem{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount, EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 5, Burst: 0, Period: 1_000},
		},
		{
			name: "throttle unknown",
			item: ConstraintItem{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv, EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{},
		},
		{
			name: "concurrency account step",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount}},
			want: ConstraintLimits{Limit: 20},
		},
		{
			name: "concurrency fn step",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn}},
			want: ConstraintLimits{Limit: 5},
		},
		{
			name: "concurrency account run",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeRun, Scope: enums.ConcurrencyScopeAccount}},
			want: ConstraintLimits{Limit: 50},
		},
		{
			name: "concurrency fn run",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeRun, Scope: enums.ConcurrencyScopeFn}},
			want: ConstraintLimits{Limit: 8},
		},
		{
			name: "concurrency env has no plain limit",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeEnv}},
			want: ConstraintLimits{},
		},
		{
			name: "concurrency custom fn",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn, KeyExpressionHash: "cc-fn", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 2},
		},
		{
			name: "concurrency custom account",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount, KeyExpressionHash: "cc-acct", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{Limit: 4},
		},
		{
			name: "concurrency custom unknown",
			item: ConstraintItem{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeEnv, KeyExpressionHash: "cc-missing", EvaluatedKeyHash: "v"}},
			want: ConstraintLimits{},
		},
		{
			name: "semaphore has no config limit",
			item: ConstraintItem{Kind: ConstraintKindSemaphore, Semaphore: &SemaphoreConstraint{ID: config.Semaphores[0].ID, Weight: 2}},
			want: ConstraintLimits{},
		},
		{
			name: "nil payload",
			item: ConstraintItem{Kind: ConstraintKindConcurrency},
			want: ConstraintLimits{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.item.ResolveLimits(config)
			require.Equal(t, tc.want, got)

			serialized := tc.item.ToSerializedConstraintItem(config, accountID, envID, fnID)
			switch tc.item.Kind {
			case ConstraintKindRateLimit:
				if tc.item.RateLimit == nil {
					return
				}
				require.Equal(t, got.Limit, serialized.RateLimit.Limit)
				require.Equal(t, got.Burst, serialized.RateLimit.Burst)
				require.Equal(t, got.Period, serialized.RateLimit.Period)
			case ConstraintKindThrottle:
				if tc.item.Throttle == nil {
					return
				}
				require.Equal(t, got.Limit, serialized.Throttle.Limit)
				require.Equal(t, got.Burst, serialized.Throttle.Burst)
				require.Equal(t, got.Period, serialized.Throttle.Period)
			case ConstraintKindConcurrency:
				if tc.item.Concurrency == nil {
					return
				}
				require.Equal(t, got.Limit, serialized.Concurrency.Limit)
			}
		})
	}
}

func TestSortConstraintsMatchesInternal(t *testing.T) {
	items := func() []ConstraintItem {
		return []ConstraintItem{
			{Kind: ConstraintKindSemaphore, Semaphore: &SemaphoreConstraint{ID: "fn:b"}},
			{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
			{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
			{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount}},
			{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
			{Kind: ConstraintKindSemaphore, Semaphore: &SemaphoreConstraint{ID: "app:a"}},
		}
	}

	exported, internal := items(), items()
	SortConstraints(exported)
	sortConstraints(internal)
	require.Equal(t, internal, exported)
	require.Equal(t, ConstraintKindRateLimit, exported[0].Kind)
	require.Equal(t, "app:a", exported[4].Semaphore.ID)
}
