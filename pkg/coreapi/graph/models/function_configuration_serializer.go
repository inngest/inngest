package models

import (
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
)

// used for dev server since we don't have user's plan
const UnknownPlanConcurrencyLimit = -1

func ToFunctionConfiguration(fn *inngest.Function, planConcurrencyLimit int) *FunctionConfiguration {
	var concurrencyConfig []*ConcurrencyConfiguration
	if fn.Concurrency != nil {
		concurrencyConfig = make([]*ConcurrencyConfiguration, len(fn.Concurrency.Limits))
		for i, conc := range fn.Concurrency.Limits {
			concLimit := conc.Limit
			if planConcurrencyLimit > 0 {
				concLimit = min(conc.Limit, planConcurrencyLimit)
			}

			concurrencyConfig[i] = &ConcurrencyConfiguration{
				Scope: mapScope(conc.Scope),
				Limit: &ConcurrencyLimitConfiguration{
					Value:       concLimit,
					IsPlanLimit: boolPtr(concLimit == planConcurrencyLimit),
				},
				Key: conc.Key,
			}
		}
	} else {
		concurrencyConfig = []*ConcurrencyConfiguration{
			{
				Scope: ConcurrencyScopeAccount,
				Limit: &ConcurrencyLimitConfiguration{
					Value:       planConcurrencyLimit, // TODO: render -1 as infinite on frontend?
					IsPlanLimit: boolPtr(true),
				},
			},
		}
	}

	var priority *string
	if fn.Priority != nil && fn.Priority.Run != nil {
		priority = fn.Priority.Run
	}

	var throttle *ThrottleConfiguration
	if fn.Throttle != nil {
		throttle = &ThrottleConfiguration{
			Burst: int(fn.Throttle.Burst),
			Key:   fn.Throttle.Key,
			Limit: int(fn.Throttle.Limit),

			// TODO: We need a custom "time.Duration to string" function that
			// formats it more closely to what we want. For example,
			// time.Duration.String formats `48 * time.Hour` as "48h0m0s", but
			// we really want it to be "2d"
			Period: fn.Throttle.Period.String(),
		}
	}

	var singleton *SingletonConfiguration
	if fn.Singleton != nil {
		singleton = &SingletonConfiguration{
			Key:  fn.Singleton.Key,
			Mode: mapSingletonMode(fn.Singleton.Mode),
		}
	}

	return &FunctionConfiguration{
		Cancellations: mapCancellations(fn.Cancel),
		Retries: &RetryConfiguration{
			Value:     fn.Steps[0].RetryCount(),
			IsDefault: boolPtr(fn.Steps[0].RetryCount() == consts.DefaultRetryCount),
		},
		Priority:    priority,
		EventsBatch: mapEventsBatch(fn.EventBatch),
		Concurrency: concurrencyConfig,
		RateLimit:   mapRateLimit(fn.RateLimit),
		Debounce:    mapDebounce(fn.Debounce),
		Throttle:    throttle,
		Singleton:   singleton,
	}
}

func mapCancellations(cancels []inngest.Cancel) []*CancellationConfiguration {
	cancellations := make([]*CancellationConfiguration, len(cancels))
	for i, cancel := range cancels {
		cancellations[i] = &CancellationConfiguration{
			Event:     cancel.Event,
			Timeout:   cancel.Timeout,
			Condition: cancel.If,
		}
	}
	return cancellations
}

func mapEventsBatch(batch *inngest.EventBatchConfig) *EventsBatchConfiguration {
	if batch == nil {
		return nil
	}
	return &EventsBatchConfiguration{
		MaxSize: batch.MaxSize,
		Timeout: batch.Timeout,
		Key:     batch.Key,
	}
}

func mapRateLimit(limit *inngest.RateLimit) *RateLimitConfiguration {
	if limit == nil {
		return nil
	}
	return &RateLimitConfiguration{
		Limit:  int(limit.Limit),
		Period: limit.Period,
		Key:    limit.Key,
	}
}

func mapDebounce(debounce *inngest.Debounce) *DebounceConfiguration {
	if debounce == nil {
		return nil
	}
	return &DebounceConfiguration{
		Period: debounce.Period,
		Key:    debounce.Key,
	}
}

func mapScope(internalEnum enums.ConcurrencyScope) ConcurrencyScope {
	var enumMapping = map[enums.ConcurrencyScope]ConcurrencyScope{
		enums.ConcurrencyScopeFn:      ConcurrencyScopeFunction,
		enums.ConcurrencyScopeEnv:     ConcurrencyScopeEnvironment,
		enums.ConcurrencyScopeAccount: ConcurrencyScopeAccount,
	}

	if gqlEnum, ok := enumMapping[internalEnum]; ok {
		return gqlEnum
	}
	return ConcurrencyScopeFunction
}

func mapSingletonMode(internalEnum enums.SingletonMode) SingletonMode {
	var enumMapping = map[enums.SingletonMode]SingletonMode{
		enums.SingletonModeSkip:   SingletonModeSkip,
		enums.SingletonModeCancel: SingletonModeCancel,
	}

	if gqlEnum, ok := enumMapping[internalEnum]; ok {
		return gqlEnum
	}

	return SingletonModeSkip
}

func boolPtr(b bool) *bool {
	return &b
}
