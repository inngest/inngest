package models

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jinzhu/copier"
)

func TestToFunctionConfiguration(t *testing.T) {
	tests := []struct {
		name                 string
		fn                   *inngest.Function
		planConcurrencyLimit int
		expected             *FunctionConfiguration
	}{
		{
			name: "cancellations",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Cancel: []inngest.Cancel{
					{
						Event:   "test/cancel",
						Timeout: util.StrPtr("30m"),
						If:      util.StrPtr("event.data.id == 2"),
					},
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Cancellations: []*CancellationConfiguration{
					{
						Event:     "test/cancel",
						Timeout:   util.StrPtr("30m"),
						Condition: util.StrPtr("event.data.id == 2"),
					},
				},
			}),
		},
		{
			// default retries are already covered by every other test case
			name: "retries, non-default",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Steps: []inngest.Step{
					{
						ID:      "test-step",
						Name:    "test-step",
						URI:     "https://example.com/api/inngest?step=foo",
						Retries: intPtr(10),
					},
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Retries: &RetryConfiguration{
					Value:     10,
					IsDefault: boolPtr(false),
				},
			}),
		},
		{
			name: "priority",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Priority: &inngest.Priority{
					Run: util.StrPtr("event.data.plan == 'enterprise' ? 180 : 0"),
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Priority: util.StrPtr("event.data.plan == 'enterprise' ? 180 : 0"),
			}),
		},
		{
			name: "batch events",
			fn: mergeWithDefaultFunction(&inngest.Function{
				EventBatch: &inngest.EventBatchConfig{
					MaxSize: 50,
					Timeout: "30s",
					Key:     util.StrPtr("event.data.customer_id"),
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				EventsBatch: &EventsBatchConfiguration{
					MaxSize: 50,
					Timeout: "30s",
					Key:     util.StrPtr("event.data.customer_id"),
				},
			}),
		},
		{
			name: "concurrency, no plan limit (i.e. dev-server)",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Concurrency: &inngest.ConcurrencyLimits{
					Limits: []inngest.Concurrency{
						{
							Scope: enums.ConcurrencyScopeAccount,
							Limit: 20,
							Key:   util.StrPtr("event.data.id == 2"),
						},
						{
							Scope: enums.ConcurrencyScopeFn,
							Limit: 5,
							Key:   util.StrPtr("event.data.id == 2"),
						},
					},
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Concurrency: []*ConcurrencyConfiguration{
					{
						Scope: ConcurrencyScopeAccount,
						Limit: &ConcurrencyLimitConfiguration{
							Value:       20,
							IsPlanLimit: boolPtr(false),
						},
						Key: util.StrPtr("event.data.id == 2"),
					},
					{
						Scope: ConcurrencyScopeFunction,
						Limit: &ConcurrencyLimitConfiguration{
							Value:       5,
							IsPlanLimit: boolPtr(false),
						},
						Key: util.StrPtr("event.data.id == 2"),
					},
				},
			}),
		},
		{
			name: "concurrency, with plan limit (i.e. cloud)",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Concurrency: &inngest.ConcurrencyLimits{
					Limits: []inngest.Concurrency{
						{
							Scope: enums.ConcurrencyScopeAccount,
							Limit: 20,
							Key:   util.StrPtr("event.data.id == 2"),
						},
						{
							Scope: enums.ConcurrencyScopeFn,
							Limit: 5,
							Key:   util.StrPtr("event.data.id == 2"),
						},
					},
				},
			}),
			planConcurrencyLimit: 10,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Concurrency: []*ConcurrencyConfiguration{
					{
						Scope: ConcurrencyScopeAccount,
						Limit: &ConcurrencyLimitConfiguration{
							Value:       10,
							IsPlanLimit: boolPtr(true),
						},
						Key: util.StrPtr("event.data.id == 2"),
					},
					{
						Scope: ConcurrencyScopeFunction,
						Limit: &ConcurrencyLimitConfiguration{
							Value:       5,
							IsPlanLimit: boolPtr(false),
						},
						Key: util.StrPtr("event.data.id == 2"),
					},
				},
			}),
		},
		{
			name: "rate limit",
			fn: mergeWithDefaultFunction(&inngest.Function{
				RateLimit: &inngest.RateLimit{
					Limit:  10,
					Period: "30s",
					Key:    util.StrPtr("event.data.customer_id"),
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				RateLimit: &RateLimitConfiguration{
					Limit:  10,
					Period: "30s",
					Key:    util.StrPtr("event.data.customer_id"),
				},
			}),
		},
		{
			name: "debounce",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Debounce: &inngest.Debounce{
					Key:    nil,
					Period: "10s",
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Debounce: &DebounceConfiguration{
					Key:    nil,
					Period: "10s",
				},
			}),
		},
		{
			name: "throttle",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Throttle: &inngest.Throttle{
					Limit:  10,
					Period: 30 * time.Minute,
					Burst:  3,
					Key:    util.StrPtr("event.data.customer_id"),
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Throttle: &ThrottleConfiguration{
					Limit:  10,
					Period: "30m0s",
					Burst:  3,
					Key:    util.StrPtr("event.data.customer_id"),
				},
			}),
		},
		{
			name: "singleton",
			fn: mergeWithDefaultFunction(&inngest.Function{
				Singleton: &inngest.Singleton{
					Mode: enums.SingletonModeSkip,
					Key:  util.StrPtr("event.data.id == 2"),
				},
			}),
			planConcurrencyLimit: UnknownPlanConcurrencyLimit,
			expected: mergeWithDefaultFunctionConfiguration(&FunctionConfiguration{
				Singleton: &SingletonConfiguration{
					Mode: SingletonModeSkip,
					Key:  util.StrPtr("event.data.id == 2"),
				},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn.Validate(context.Background())
			if err != nil {
				t.Errorf("Invalid function configuration for test %v: %v", tt.name, err)
			}
			if got := ToFunctionConfiguration(tt.fn, tt.planConcurrencyLimit); !reflect.DeepEqual(got, tt.expected) {
				gotJson, _ := json.MarshalIndent(got, "", "  ")
				expectedJson, _ := json.MarshalIndent(tt.expected, "", "  ")
				t.Errorf("ToFunctionConfiguration() = \n%s, expected \n%s", gotJson, expectedJson)
			}
		})
	}
}

func mergeWithDefaultFunction(overlay *inngest.Function) *inngest.Function {
	base := &inngest.Function{
		Name: "test/function",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event: "test/trigger",
				},
			},
		},
		Steps: []inngest.Step{
			{
				ID:   "test-step",
				Name: "test-step",
				URI:  "https://example.com/api/inngest?step=foo",
			},
		},
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
	}
	err := copier.CopyWithOption(base, overlay, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return nil
	}
	return base

}

func mergeWithDefaultFunctionConfiguration(overlay *FunctionConfiguration) *FunctionConfiguration {
	base := &FunctionConfiguration{
		Cancellations: []*CancellationConfiguration{},
		Retries: &RetryConfiguration{
			Value:     4,
			IsDefault: boolPtr(true),
		},
		Concurrency: []*ConcurrencyConfiguration{
			{
				Scope: ConcurrencyScopeAccount,
				Limit: &ConcurrencyLimitConfiguration{
					Value:       UnknownPlanConcurrencyLimit,
					IsPlanLimit: boolPtr(true),
				},
			},
		},
	}
	err := copier.CopyWithOption(base, overlay, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return nil
	}
	return base
}

func intPtr(b int) *int {
	return &b
}
