package constraintapi

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util"
)

type SemaphoreReleaseMode int

const (
	// SemaphoreReleaseAuto decrements the semaphore counter when the constraint lease is released.
	// Used for worker concurrency where each step independently acquires and releases.
	SemaphoreReleaseAuto SemaphoreReleaseMode = 0

	// SemaphoreReleaseManual requires explicit release via the SemaphoreManager API.
	// Used for function concurrency where the hold persists across the entire run.
	SemaphoreReleaseManual SemaphoreReleaseMode = 1
)

// SemaphoreIDApp returns the semaphore ID for worker concurrency (per-app).
func SemaphoreIDApp(appID uuid.UUID) string {
	return fmt.Sprintf("app:%s", appID)
}

// SemaphoreIDFn returns the semaphore ID for function concurrency (no key).
func SemaphoreIDFn(functionID uuid.UUID) string {
	return fmt.Sprintf("fn:%s", functionID)
}

// SemaphoreIDFnKey returns the semaphore ID for function concurrency with a key expression.
// The ID is a hash of the function ID + the raw (unevaluated) expression.
func SemaphoreIDFnKey(functionID uuid.UUID, expression string) string {
	return fmt.Sprintf("fnkey:%s", util.XXHash(functionID.String()+expression))
}

// SemaphoreConstraint represents a semaphore-based capacity constraint.
// Semaphores track usage via simple counters (INCRBY/DECRBY) with separately
// managed capacity, providing O(1) capacity checks.
type SemaphoreConstraint struct {
	// ID is the unevaluated semaphore name, always prefixed:
	//   app:<uuid>    — worker concurrency
	//   fn:<uuid>     — function concurrency
	//   fnkey:<xxhash(fnID + expression)> — hash of function ID & unevaluated expression
	ID string

	// UsageValue is the xxhash of the *evaluated* expression, if the semaphore was created via
	// expressions.  This allows arbitrary expressions per fn for semaphores.
	UsageValue string

	// Weight is the number of units to acquire from the semaphore (default 1).
	Weight int64

	// Release controls when the semaphore counter is decremented.
	Release SemaphoreReleaseMode
}

func (s *SemaphoreConstraint) UsageKey(accountID uuid.UUID) string {
	return fmt.Sprintf("{cs}:%s:sem:%s:usage:%s", accountScope(accountID), s.ID, s.UsageValue)
}

func (s *SemaphoreConstraint) CapacityKey(accountID uuid.UUID) string {
	return fmt.Sprintf("{cs}:%s:sem:%s:cap", accountScope(accountID), s.ID)
}

func (s *SemaphoreConstraint) PrettyString() string {
	if s.UsageValue != "" {
		return fmt.Sprintf("id %s, usage_value %s, weight %d, release %d", s.ID, s.UsageValue, s.Weight, s.Release)
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
