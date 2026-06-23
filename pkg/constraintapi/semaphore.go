package constraintapi

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util"
)

type SemaphoreReleaseMode int

const (
	// SemaphoreReleaseAuto decrements the semaphore counter when the constraint lease is released.
	// Used for per-item worker and account concurrency.
	SemaphoreReleaseAuto SemaphoreReleaseMode = 0

	// SemaphoreReleaseManual requires explicit release via the SemaphoreManager API.
	// Used for function concurrency where the hold persists across the entire run.
	SemaphoreReleaseManual SemaphoreReleaseMode = 1
)

const (
	semaphorePrefixApp     = "app:"
	semaphorePrefixAccount = "acct:"
	semaphorePrefixFn      = "fn:"
	semaphorePrefixFnKey   = "fnkey:"
	semaphorePrefixHash    = "hash:"
)

// SemaphoreIDApp returns the semaphore ID for worker concurrency (per-app).
func SemaphoreIDApp(appID uuid.UUID) string {
	return fmt.Sprintf("%s%s", semaphorePrefixApp, appID)
}

// SemaphoreIDAccount returns the semaphore ID for account-scoped concurrency.
func SemaphoreIDAccount(accountID uuid.UUID) string {
	return fmt.Sprintf("%s%s", semaphorePrefixAccount, accountID)
}

// SemaphoreIDFn returns the semaphore ID for function concurrency (no key).
func SemaphoreIDFn(functionID uuid.UUID) string {
	return fmt.Sprintf("%s%s", semaphorePrefixFn, functionID)
}

// SemaphoreIDFnKey returns the semaphore ID for function concurrency with a key expression.
// The ID is a hash of the function ID + the raw (unevaluated) expression.
func SemaphoreIDFnKey(functionID uuid.UUID, expression string) string {
	return fmt.Sprintf("%s%s", semaphorePrefixFnKey, util.XXHash(functionID.String()+expression))
}

// SemaphoreConstraint represents a semaphore-based capacity constraint.
// Semaphores track usage via simple counters (INCRBY/DECRBY) with separately
// managed capacity, providing O(1) capacity checks.
type SemaphoreConstraint struct {
	// ID is the unevaluated semaphore name, always prefixed:
	//   app:<uuid>    — worker concurrency
	//   acct:<uuid>   — account concurrency
	//   fn:<uuid>     — function concurrency
	//   fnkey:<xxhash(fnID + expression)> — hash of function ID & unevaluated expression
	ID string

	// EvaluatedKeyHash is the xxhash of the *evaluated* expression, if the semaphore was created via
	// expressions.  This allows arbitrary expressions per fn for semaphores.
	EvaluatedKeyHash string

	// Weight is the number of units to acquire from the semaphore (default 1).
	Weight int64

	// Release controls when the semaphore counter is decremented.
	Release SemaphoreReleaseMode
}

func (s *SemaphoreConstraint) UsageKey(accountID uuid.UUID) string {
	return fmt.Sprintf("{cs}:%s:sem:%s:usage:%s", accountScope(accountID), s.ID, s.EvaluatedKeyHash)
}

func (s *SemaphoreConstraint) CapacityKey(accountID uuid.UUID) string {
	return fmt.Sprintf("{cs}:%s:sem:%s:cap", accountScope(accountID), s.ID)
}

func (s *SemaphoreConstraint) IsAccountConcurrency() bool {
	return s != nil && isAccountConcurrencySemaphore(s.ID, s.EvaluatedKeyHash)
}

func (s *SemaphoreConstraint) IsFunctionConcurrency() bool {
	return s != nil && isFunctionConcurrencySemaphore(s.ID, s.EvaluatedKeyHash)
}

func (s *SemaphoreConstraint) IsFunctionScoped() bool {
	return s != nil &&
		(strings.HasPrefix(s.ID, semaphorePrefixFn) ||
			strings.HasPrefix(s.ID, semaphorePrefixFnKey) ||
			strings.HasPrefix(s.ID, semaphorePrefixHash))
}

func (s Semaphore) IsAccountConcurrency() bool {
	return isAccountConcurrencySemaphore(s.ID, s.EvaluatedKeyHash)
}

func isAccountConcurrencySemaphore(id, evaluatedKeyHash string) bool {
	return evaluatedKeyHash == "" && strings.HasPrefix(id, semaphorePrefixAccount)
}

func isFunctionConcurrencySemaphore(id, evaluatedKeyHash string) bool {
	return evaluatedKeyHash == "" && strings.HasPrefix(id, semaphorePrefixFn)
}

func (s *SemaphoreConstraint) PrettyString() string {
	if s.EvaluatedKeyHash != "" {
		return fmt.Sprintf("id %s, evaluated_key_hash %s, weight %d, release %d", s.ID, s.EvaluatedKeyHash, s.Weight, s.Release)
	}
	return fmt.Sprintf("id %s, weight %d, release %d", s.ID, s.Weight, s.Release)
}

func (s *SemaphoreConstraint) PrettyStringConfig(config ConstraintConfig) string {
	for _, sc := range config.Semaphores {
		if sc.ID == s.ID {
			return fmt.Sprintf("weight %d, release %d", sc.Weight, sc.Release)
		}
	}
	return "unknown"
}
