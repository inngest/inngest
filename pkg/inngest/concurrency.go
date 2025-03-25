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
	Limits []Concurrency
}

// PartitionConcurrencyLimits returns the partition concurrency limit for the overall function,
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
	return nil
}

func (c *ConcurrencyLimits) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	*c = ConcurrencyLimits{}

	switch b[0] {
	case '{':
		// Unmarshal a single concurrency itme.
		item := &Concurrency{}
		if err := json.Unmarshal(b, item); err != nil {
			return err
		}
		c.Limits = []Concurrency{*item}
	case '[':
		items := []Concurrency{}
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
		c.Limits = []Concurrency{
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

	// For each concurrency limit, calcluate the hash if not set.
	for n, item := range c.Limits {
		if item.Key != nil && item.Hash == "" {
			// Use xxhash for 64 bit hashing.  While this can have collisions, the
			// chance over the max of 2 keys is extremely low (almost impossible) and
			// it's much faster/shorter.
			c.Limits[n].Hash = hashConcurrencyKey(*item.Key)
		}
	}

	return nil
}

func hashConcurrencyKey(key string) string {
	return strconv.FormatUint(xxhash.Sum64String(key), 36)
}

func (c *ConcurrencyLimits) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Limits)
}

// Concurrency represents a single concurrency limit for a function.
type Concurrency struct {
	Limit int                    `json:"limit"`
	Key   *string                `json:"key,omitempty"`
	Scope enums.ConcurrencyScope `json:"scope"`
	Hash  string                 `json:"hash"`
}

func (c Concurrency) Validate(ctx context.Context) error {
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
func (c Concurrency) Evaluate(ctx context.Context, scopeID uuid.UUID, input map[string]any) string {
	evaluated := ""
	if c.Key != nil {
		// The input data is always wrapped in an event variable, for event.data.foo
		val, _, _ := expressions.Evaluate(ctx, *c.Key, map[string]any{"event": input})
		switch v := val.(type) {
		case string:
			evaluated = v
		default:
			evaluated = fmt.Sprintf("%v", v)
		}
	}
	return util.ConcurrencyKey(c.Scope, scopeID, evaluated)
}

func (c Concurrency) Prefix() string {
	return util.ConcurrencyScopePrefix(c.Scope)
}

func (c Concurrency) IsCustomLimit() bool {
	return !c.IsPartitionLimit()
}

// IsPartitionLimit returns whether this is the limit for the overall function,
// without a key.
func (c Concurrency) IsPartitionLimit() bool {
	return c.Scope == enums.ConcurrencyScopeFn && c.Key == nil
}
