package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
)

type ConstraintKind string

const (
	ConstraintKindRateLimit   ConstraintKind = "rate_limit"
	ConstraintKindConcurrency ConstraintKind = "concurrency"
	ConstraintKindThrottle    ConstraintKind = "throttle"
	ConstraintKindSemaphore   ConstraintKind = "semaphore"
)

func (k ConstraintKind) IsQueueConstraint() bool {
	return k == ConstraintKindConcurrency || k == ConstraintKindThrottle || k == ConstraintKindSemaphore
}

func (k ConstraintKind) PrettyString() string {
	switch k {
	case ConstraintKindRateLimit:
		return "rate limit"
	case ConstraintKindConcurrency:
		return "concurrency"
	case ConstraintKindThrottle:
		return "throttle"
	case ConstraintKindSemaphore:
		return "semaphore"
	default:
		return "unknown"
	}
}

type RateLimitConstraint struct {
	Scope enums.RateLimitScope

	KeyExpressionHash string

	EvaluatedKeyHash string
}

// StateKey returns the fully-qualified Redis key pointing to the rate limit GCRA state
func (r *RateLimitConstraint) StateKey(accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) string {
	var keyID string
	if r.KeyExpressionHash != "" {
		keyID = fmt.Sprintf("<%s:%s>", r.KeyExpressionHash, r.EvaluatedKeyHash)
	}

	switch r.Scope {
	case enums.RateLimitScopeAccount:
		return fmt.Sprintf("{cs}:%s:rl:a:%s%s", accountScope(accountID), accountID, keyID)
	case enums.RateLimitScopeEnv:
		return fmt.Sprintf("{cs}:%s:rl:e:%s%s", accountScope(accountID), envID, keyID)
	case enums.RateLimitScopeFn:
		return fmt.Sprintf("{cs}:%s:rl:f:%s%s", accountScope(accountID), fnID, keyID)
	default:
		return ""
	}
}

func (r *RateLimitConstraint) PrettyString() string {
	return fmt.Sprintf("scope %s, expression hash %s, key hash %s", r.Scope, r.KeyExpressionHash, r.EvaluatedKeyHash)
}

func (r *RateLimitConstraint) PrettyStringConfig(config ConstraintConfig) string {
	for _, rlc := range config.RateLimit {
		if rlc.Scope != r.Scope || rlc.KeyExpressionHash != r.KeyExpressionHash {
			continue
		}

		return fmt.Sprintf("period %d, limit %d", time.Duration(rlc.Period)*time.Second, rlc.Limit)
	}

	return "unknown"
}

type ConcurrencyConstraint struct {
	// Mode specifies whether concurrency is applied to step (default) or function run level
	Mode enums.ConcurrencyMode

	// Scope specifies the concurrency scope, defaults to function
	Scope enums.ConcurrencyScope

	// KeyExpressionHash is the hashed key expression. If this is set, this refers to a custom concurrency key.
	KeyExpressionHash string
	EvaluatedKeyHash  string
}

const ConcurrencyLimitRetryAfter = 2 * time.Second

func (c ConcurrencyConstraint) RetryAfter() time.Duration {
	return ConcurrencyLimitRetryAfter
}

func (c ConcurrencyConstraint) InProgressLeasesKey(accountID, envID, functionID uuid.UUID) string {
	switch c.Mode {
	case enums.ConcurrencyModeStep:
	case enums.ConcurrencyModeRun:
		// TODO: How are we going to enforce run level concurrency?
	}

	var scopeID string
	var entityID uuid.UUID
	switch c.Scope {
	case enums.ConcurrencyScopeAccount:
		scopeID = "a"
		entityID = accountID
	case enums.ConcurrencyScopeEnv:
		scopeID = "e"
		entityID = envID
	case enums.ConcurrencyScopeFn:
		scopeID = "f"
		entityID = functionID
	}

	var keyID string
	if c.IsCustomKey() {
		keyID = fmt.Sprintf("<%s:%s>", c.KeyExpressionHash, c.EvaluatedKeyHash)
	}

	return fmt.Sprintf("{cs}:%s:concurrency:%s:%s%s", accountScope(accountID), scopeID, entityID, keyID)
}

func (c ConcurrencyConstraint) IsCustomKey() bool {
	return c.KeyExpressionHash != ""
}

func (c *ConcurrencyConstraint) PrettyString() string {
	return fmt.Sprintf("mode %s, scope %s, expression hash %s, key hash %s", c.Mode, c.Scope, c.KeyExpressionHash, c.EvaluatedKeyHash)
}

func (c *ConcurrencyConstraint) PrettyStringConfig(config ConstraintConfig) string {
	var limit int
	switch {
	case c.Mode == enums.ConcurrencyModeStep && c.EvaluatedKeyHash == "" && c.Scope == enums.ConcurrencyScopeAccount:
		limit = config.Concurrency.AccountConcurrency
	case c.Mode == enums.ConcurrencyModeStep && c.EvaluatedKeyHash == "" && c.Scope == enums.ConcurrencyScopeFn:
		limit = config.Concurrency.FunctionConcurrency
	case c.EvaluatedKeyHash != "":
		for _, cc := range config.Concurrency.CustomConcurrencyKeys {
			if cc.Mode == c.Mode && cc.Scope == c.Scope && cc.KeyExpressionHash == c.KeyExpressionHash {
				limit = cc.Limit
				break
			}
		}

	default:
	}

	return fmt.Sprintf("limit %d", limit)
}

type ThrottleConstraint struct {
	Scope enums.ThrottleScope

	KeyExpressionHash string
	EvaluatedKeyHash  string
}

// StateKey returns the fully-qualified Redis key pointing to the throttle GCRA state
func (t *ThrottleConstraint) StateKey(accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) string {
	var keyID string
	if t.KeyExpressionHash != "" {
		keyID = fmt.Sprintf("<%s:%s>", t.KeyExpressionHash, t.EvaluatedKeyHash)
	}

	switch t.Scope {
	case enums.ThrottleScopeAccount:
		return fmt.Sprintf("{cs}:%s:throttle:a:%s%s", accountScope(accountID), accountID, keyID)
	case enums.ThrottleScopeEnv:
		return fmt.Sprintf("{cs}:%s:throttle:e:%s%s", accountScope(accountID), envID, keyID)
	case enums.ThrottleScopeFn:
		return fmt.Sprintf("{cs}:%s:throttle:f:%s%s", accountScope(accountID), fnID, keyID)
	default:
		return ""
	}
}

func (t *ThrottleConstraint) PrettyString() string {
	return fmt.Sprintf("scope %s, expression hash %s, key hash %s", t.Scope, t.KeyExpressionHash, t.EvaluatedKeyHash)
}

func (t *ThrottleConstraint) PrettyStringConfig(config ConstraintConfig) string {
	for _, tc := range config.Throttle {
		if tc.Scope != t.Scope || tc.KeyExpressionHash != t.KeyExpressionHash {
			continue
		}

		return fmt.Sprintf("period %d, limit %d, burst %d", time.Duration(tc.Period)*time.Second, tc.Limit, tc.Burst)
	}

	return "unknown"
}

const SemaphoreLimitRetryAfter = 2 * time.Second

type SemaphoreConstraint struct {
	// Name uniquely identifies the semaphore on the scope
	Name string

	// Scope specifies the semaphore scope
	Scope enums.SemaphoreScope

	// KeyExpressionHash is the hashed key expression, if set
	KeyExpressionHash string

	// EvaluatedKeyHash is the evaluated key hash
	EvaluatedKeyHash string

	// Amount is how much capacity to acquire per item
	Amount int
}

// StateKey returns the fully-qualified Redis key for the semaphore counter.
// Pattern: {cs}:a:<accountID>:sem:<scope>:<entityID>[:<exprHash>:<evalKeyHash>]
func (s *SemaphoreConstraint) StateKey(accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) string {
	var scopeID string
	var entityID uuid.UUID
	switch s.Scope {
	case enums.SemaphoreScopeAccount:
		scopeID = "a"
		entityID = accountID
	case enums.SemaphoreScopeEnv:
		scopeID = "e"
		entityID = envID
	case enums.SemaphoreScopeFn:
		scopeID = "f"
		entityID = fnID
	}

	key := fmt.Sprintf("{cs}:%s:sem:%s:%s", accountScope(accountID), scopeID, entityID)
	if s.KeyExpressionHash != "" {
		key = fmt.Sprintf("%s:%s:%s", key, s.KeyExpressionHash, s.EvaluatedKeyHash)
	}
	return key
}

func (s *SemaphoreConstraint) RetryAfter() time.Duration {
	return SemaphoreLimitRetryAfter
}

func (s *SemaphoreConstraint) PrettyString() string {
	return fmt.Sprintf("name %s, scope %s, expression hash %s, key hash %s, amount %d", s.Name, s.Scope, s.KeyExpressionHash, s.EvaluatedKeyHash, s.Amount)
}

func (s *SemaphoreConstraint) PrettyStringConfig(config ConstraintConfig) string {
	for _, sc := range config.Semaphore {
		if sc.Name == s.Name && sc.Scope == s.Scope && sc.KeyExpressionHash == s.KeyExpressionHash {
			return fmt.Sprintf("capacity %d", sc.Capacity)
		}
	}
	return "unknown"
}

type ConstraintItem struct {
	Kind ConstraintKind

	Concurrency *ConcurrencyConstraint
	Throttle    *ThrottleConstraint
	RateLimit   *RateLimitConstraint
	Semaphore   *SemaphoreConstraint
}

// IsFunctionLevelConstraint returns whether the constraint is on the function level
func (ci ConstraintItem) IsFunctionLevelConstraint() bool {
	switch ci.Kind {
	case ConstraintKindRateLimit:
		return ci.RateLimit != nil && ci.RateLimit.Scope == enums.RateLimitScopeFn
	case ConstraintKindThrottle:
		return ci.Throttle != nil && ci.Throttle.Scope == enums.ThrottleScopeFn
	case ConstraintKindConcurrency:
		return ci.Concurrency != nil && ci.Concurrency.Scope == enums.ConcurrencyScopeFn
	case ConstraintKindSemaphore:
		return ci.Semaphore != nil && ci.Semaphore.Scope == enums.SemaphoreScopeFn
	default:
		return false
	}
}

func (ci ConstraintItem) PrettyString() string {
	switch ci.Kind {
	case ConstraintKindConcurrency:
		return ci.Concurrency.PrettyString()
	case ConstraintKindRateLimit:
		return ci.RateLimit.PrettyString()
	case ConstraintKindThrottle:
		return ci.Throttle.PrettyString()
	case ConstraintKindSemaphore:
		return ci.Semaphore.PrettyString()
	default:
		return "unknown"
	}
}

func (ci ConstraintItem) PrettyStringConfig(config ConstraintConfig) string {
	switch ci.Kind {
	case ConstraintKindConcurrency:
		return ci.Concurrency.PrettyStringConfig(config)
	case ConstraintKindRateLimit:
		return ci.RateLimit.PrettyStringConfig(config)
	case ConstraintKindThrottle:
		return ci.Throttle.PrettyStringConfig(config)
	case ConstraintKindSemaphore:
		return ci.Semaphore.PrettyStringConfig(config)
	default:
		return "unknown"
	}
}

// MetricsIdentifier returns an identifier for the constraint item.
// For throttle and rate limit, it returns the kind.
// For concurrency, it returns the scope.
func (ci ConstraintItem) MetricsIdentifier() string {
	switch ci.Kind {
	case ConstraintKindConcurrency:
		if ci.Concurrency != nil {
			return ci.Concurrency.Scope.String()
		}
		return ci.Kind.PrettyString()
	default:
		return ci.Kind.PrettyString()
	}
}

func (ci ConstraintItem) CacheKey(accountID, envID, functionID uuid.UUID) string {
	switch ci.Kind {
	case ConstraintKindConcurrency:
		if ci.Concurrency == nil {
			return ""
		}
		c := ci.Concurrency

		var scopeLetter string
		var entityID uuid.UUID
		switch c.Scope {
		case enums.ConcurrencyScopeAccount:
			scopeLetter = "a"
		case enums.ConcurrencyScopeEnv:
			scopeLetter = "e"
			entityID = envID
		case enums.ConcurrencyScopeFn:
			scopeLetter = "f"
			entityID = functionID
		}

		// Custom key uses expression and evaluated hashes
		if c.KeyExpressionHash != "" {
			return fmt.Sprintf("%s:c:%s:%s:%s", accountID, scopeLetter, c.KeyExpressionHash, c.EvaluatedKeyHash)
		}

		// Non-custom function/env scoped constraints include entity ID
		if entityID != uuid.Nil {
			return fmt.Sprintf("%s:c:%s:%s", accountID, scopeLetter, entityID)
		}

		// Account scope with no custom key
		return fmt.Sprintf("%s:c:%s", accountID, scopeLetter)

	case ConstraintKindThrottle:
		if ci.Throttle == nil {
			return ""
		}
		t := ci.Throttle

		var scopeLetter string
		var entityID uuid.UUID
		switch t.Scope {
		case enums.ThrottleScopeAccount:
			scopeLetter = "a"
		case enums.ThrottleScopeEnv:
			scopeLetter = "e"
			entityID = envID
		case enums.ThrottleScopeFn:
			scopeLetter = "f"
			entityID = functionID
		}

		// Custom key uses expression and evaluated hashes
		if t.KeyExpressionHash != "" {
			return fmt.Sprintf("%s:t:%s:%s:%s", accountID, scopeLetter, t.KeyExpressionHash, t.EvaluatedKeyHash)
		}

		// Non-custom function/env scoped constraints include entity ID
		if entityID != uuid.Nil {
			return fmt.Sprintf("%s:t:%s:%s", accountID, scopeLetter, entityID)
		}

		// Account scope with no custom key
		return fmt.Sprintf("%s:t:%s", accountID, scopeLetter)

	case ConstraintKindRateLimit:
		if ci.RateLimit == nil {
			return ""
		}
		r := ci.RateLimit

		var scopeLetter string
		var entityID uuid.UUID
		switch r.Scope {
		case enums.RateLimitScopeAccount:
			scopeLetter = "a"
		case enums.RateLimitScopeEnv:
			scopeLetter = "e"
			entityID = envID
		case enums.RateLimitScopeFn:
			scopeLetter = "f"
			entityID = functionID
		}

		// Custom key uses expression and evaluated hashes
		if r.KeyExpressionHash != "" {
			return fmt.Sprintf("%s:r:%s:%s:%s", accountID, scopeLetter, r.KeyExpressionHash, r.EvaluatedKeyHash)
		}

		// Non-custom function/env scoped constraints include entity ID
		if entityID != uuid.Nil {
			return fmt.Sprintf("%s:r:%s:%s", accountID, scopeLetter, entityID)
		}

		// Account scope with no custom key
		return fmt.Sprintf("%s:r:%s", accountID, scopeLetter)

	case ConstraintKindSemaphore:
		if ci.Semaphore == nil {
			return ""
		}
		s := ci.Semaphore

		var scopeLetter string
		var entityID uuid.UUID
		switch s.Scope {
		case enums.SemaphoreScopeAccount:
			scopeLetter = "a"
		case enums.SemaphoreScopeEnv:
			scopeLetter = "e"
			entityID = envID
		case enums.SemaphoreScopeFn:
			scopeLetter = "f"
			entityID = functionID
		}

		// Custom key uses expression and evaluated hashes
		if s.KeyExpressionHash != "" {
			return fmt.Sprintf("%s:sem:%s:%s:%s:%s", accountID, scopeLetter, s.Name, s.KeyExpressionHash, s.EvaluatedKeyHash)
		}

		// Non-custom function/env scoped constraints include entity ID
		if entityID != uuid.Nil {
			return fmt.Sprintf("%s:sem:%s:%s:%s", accountID, scopeLetter, s.Name, entityID)
		}

		// Account scope with no custom key
		return fmt.Sprintf("%s:sem:%s:%s", accountID, scopeLetter, s.Name)

	default:
		return ""
	}
}

type ConstraintUsage struct {
	Constraint ConstraintItem

	Used  int
	Limit int
}
