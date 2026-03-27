package inngest

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/util"
)

// ConcurrencyLimits represents concurrency limits specified for a function.
//
// This is a separate struct, allowing us to handle unmarshalling single concurrency
// objects or an array of objects in one codepath for backwards compatibility.
type ConcurrencyLimits struct {
	Fn   []FnConcurrency   `json:"fn"`
	Step []StepConcurrency `json:"step"`

	// Deprecated: use Step instead.  This exists for backcompat where concurrency only worked
	// across steps.
	Limits []StepConcurrency
}

// PartitionConcurrency returns the partition concurrency limit for the overall function,
// where Concurrency is scoped to a function (by default) and has no key.
func (c ConcurrencyLimits) PartitionConcurrency() int {
	for _, item := range c.Limits {
		// This is a pure function limit.
		if item.IsPartitionLimit() {
			return item.Limit
		}
	}
	return 0
}

func (c ConcurrencyLimits) Validate(ctx context.Context) error {
	if len(c.Limits) > consts.MaxConcurrencyLimits {
		return syscode.Error{
			Code:    syscode.CodeConcurrencyLimitInvalid,
			Message: fmt.Sprintf("There are more concurrency limits specified than the allowed max of: %d", consts.MaxConcurrencyLimits),
		}
	}
	for _, l := range c.Limits {
		if err := l.Validate(ctx); err != nil {
			return err
		}
	}
	if len(c.Fn) > 1 {
		return syscode.Error{
			Code:    syscode.CodeConcurrencyLimitInvalid,
			Message: "Only one function concurrency limit is allowed",
		}
	}
	for _, f := range c.Fn {
		if err := f.Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConcurrencyLimits) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	*c = ConcurrencyLimits{}

	switch b[0] {
	case '{':
		// Try the new format first: {"fn": [...], "step": [...]}
		raw := struct {
			Fn   []FnConcurrency   `json:"fn"`
			Step []StepConcurrency `json:"step"`
		}{}
		if err := json.Unmarshal(b, &raw); err == nil && (len(raw.Fn) > 0 || len(raw.Step) > 0) {
			c.Fn = raw.Fn
			c.Step = raw.Step
			c.Limits = raw.Step
		} else {
			// Legacy: single step concurrency object, e.g. {"limit": 5, "key": "..."}
			item := &StepConcurrency{}
			if err := json.Unmarshal(b, item); err != nil {
				return err
			}
			c.Limits = []StepConcurrency{*item}
		}
	case '[':
		// XXX: This is legacy, ie. >= 1 step concurrnecy limits only.
		items := []StepConcurrency{}
		if err := json.Unmarshal(b, &items); err != nil {
			return err
		}
		c.Limits = items
	default:
		// Attempt to parse this as a number.
		int, err := strconv.Atoi(string(b))
		if err != nil {
			return nil
		}
		c.Limits = []StepConcurrency{
			{
				Scope: enums.ConcurrencyScopeFn,
				Limit: int,
			},
		}
		return nil
	}

	// Sort concurrency items by limit, increasing.
	sort.Slice(c.Limits, func(i, j int) bool {
		return c.Limits[i].Limit < c.Limits[j].Limit
	})

	// For each step concurrency limit, calculate the hash if not set.
	for n, item := range c.Limits {
		if item.Key != nil && item.Hash == "" {
			c.Limits[n].Hash = hashConcurrencyKey(*item.Key)
		}
	}

	// For each fn concurrency limit, compute the ID from the key if not already set.
	for n, item := range c.Fn {
		if item.Key != nil && item.ID == "" {
			c.Fn[n].ID = hashConcurrencyKey(*item.Key)
		}
	}

	// Backfill Step from Limits so new code can use .Step,
	if len(c.Step) == 0 && len(c.Limits) > 0 {
		c.Step = c.Limits
	}

	return nil
}

func hashConcurrencyKey(key string) string {
	return strconv.FormatUint(xxhash.Sum64String(key), 36)
}

func (c *ConcurrencyLimits) MarshalJSON() ([]byte, error) {
	if len(c.Fn) > 0 {
		// New format: {"fn": [...], "step": [...]}
		return json.Marshal(struct {
			Fn   []FnConcurrency   `json:"fn,omitempty"`
			Step []StepConcurrency `json:"step,omitempty"`
		}{
			Fn:   c.Fn,
			Step: c.Limits,
		})
	}
	// Legacy format: [...] (flat array of step concurrency)
	return json.Marshal(c.Limits)
}

// FnConcurrencyScope determines what the semaphore is scoped to and
// controls release behavior.
type FnConcurrencyScope string

const (
	// FnConcurrencyScopeFn scopes the semaphore to the function. The semaphore is held
	// for the entire run (manual release on finalization). Only on start jobs.
	FnConcurrencyScopeFn FnConcurrencyScope = "fn"

	// FnConcurrencyScopeApp scopes the semaphore to the app. The semaphore is acquired
	// and released per step (auto-release), gating work behind worker capacity.
	// Added to ALL queue items.
	FnConcurrencyScopeApp FnConcurrencyScope = "app"
)

// FnConcurrency represents a concurrency limit enforced via semaphores.
// The Scope determines the semaphore ID, release mode, and which queue items
// carry the constraint.
type FnConcurrency struct {
	Limit int                `json:"limit"`
	Scope FnConcurrencyScope `json:"scope,omitempty"` // defaults to "fn"
	Key   *string            `json:"key,omitempty"`   // optional expression for the semaphore name

	// ID represents the pre-computed semaphore ID for this concurrency limit.
	// Set internally during registration (e.g., "app:<appID>" for connect apps).
	// For fn-scoped limits without a pre-set ID, evaluateFnConcurrency computes it.
	ID string `json:"id,omitempty"`
}

func (f FnConcurrency) Validate(ctx context.Context) error {
	switch f.EffectiveScope() {
	case FnConcurrencyScopeFn:
		if f.Limit <= 0 {
			return fmt.Errorf("function concurrency limit must be > 0")
		}
		if f.Key != nil {
			if _, err := expressions.NewExpressionEvaluator(ctx, *f.Key); err != nil {
				return fmt.Errorf("invalid function concurrency key '%s': %w", *f.Key, err)
			}
		}
	case FnConcurrencyScopeApp:
		// App scope is server-only — injected during connect registration.
		// Users cannot set this scope directly.
		return fmt.Errorf("app-scoped function concurrency cannot be set by users")
	default:
		return fmt.Errorf("invalid function concurrency scope: %s", f.Scope)
	}
	return nil
}

// EffectiveScope returns the scope, defaulting to FnConcurrencyScopeFn if unset.
func (f FnConcurrency) EffectiveScope() FnConcurrencyScope {
	if f.Scope == "" {
		return FnConcurrencyScopeFn
	}
	return f.Scope
}

// StepConcurrency represents a single step-level concurrency limit for a function.
type StepConcurrency struct {
	Limit int                    `json:"limit"`
	Key   *string                `json:"key,omitempty"`
	Scope enums.ConcurrencyScope `json:"scope"`
	Hash  string                 `json:"hash"`
}

func (c StepConcurrency) Validate(ctx context.Context) error {
	if c.Scope != enums.ConcurrencyScopeFn && c.Key == nil {
		return fmt.Errorf("A concurrency key must be specified for %s scoped limits", c.Scope)
	}
	if c.Key != nil {
		if _, err := expressions.NewExpressionEvaluator(ctx, *c.Key); err != nil {
			return fmt.Errorf("Invalid concurrency key '%s': %w", *c.Key, err)
		}
	}
	return nil
}

// Key returns the concurrency key
func (c StepConcurrency) EvaluatedKey(ctx context.Context, scopeID uuid.UUID, input map[string]any) string {
	evaluated := c.Evaluate(ctx, input)
	return util.ConcurrencyKey(c.Scope, scopeID, evaluated)
}

// Evaluate evaluates the custom concurrency key, without hashing.
func (c StepConcurrency) Evaluate(ctx context.Context, input map[string]any) string {
	evaluated := ""
	if c.Key != nil {
		// The input data is always wrapped in an event variable, for event.data.foo
		val, _ := expressions.Evaluate(ctx, *c.Key, map[string]any{"event": input})
		switch v := val.(type) {
		case string:
			evaluated = v
		default:
			evaluated = fmt.Sprintf("%v", v)
		}
	}

	return evaluated
}

func (c StepConcurrency) Prefix() string {
	return util.ConcurrencyScopePrefix(c.Scope)
}

func (c StepConcurrency) IsCustomLimit() bool {
	return !c.IsPartitionLimit()
}

// IsPartitionLimit returns whether this is the limit for the overall function,
// without a key.
func (c StepConcurrency) IsPartitionLimit() bool {
	return c.Scope == enums.ConcurrencyScopeFn && c.Key == nil
}
