package ddb

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
)

func NewExecutionLoader(w cqrs.Manager) coredata.ExecutionLoader {
	return executionLoader{w}
}

type executionLoader struct {
	cqrs.Manager
}

// Functions returns all functions as inngest functions.
func (el executionLoader) Functions(ctx context.Context) ([]inngest.Function, error) {
	all, _ := el.GetFunctions(ctx)
	funcs := make([]inngest.Function, len(all))
	for n, i := range all {
		f := inngest.Function{}
		_ = json.Unmarshal([]byte(i.Config), &f)
		funcs[n] = f
	}
	return funcs, nil
}

// FunctionsScheduled returns all scheduled functions available.
func (el executionLoader) FunctionsScheduled(ctx context.Context) ([]inngest.Function, error) {
	// TODO: Make less naive by storing triggers and caching.
	fns, err := el.Functions(ctx)
	if err != nil {
		return nil, err
	}
	all := []inngest.Function{}
	for _, fn := range fns {
		for _, t := range fn.Triggers {
			if t.CronTrigger != nil {
				all = append(all, fn)
				break
			}
		}
	}
	return all, nil
}

// FunctionsByTrigger returns functions for the given trigger by event name.
func (el executionLoader) FunctionsByTrigger(ctx context.Context, eventName string) ([]inngest.Function, error) {
	// TODO: Make less naive by storing triggers and caching.
	fns, err := el.Functions(ctx)
	if err != nil {
		return nil, err
	}
	all := []inngest.Function{}
	for _, fn := range fns {
		for _, t := range fn.Triggers {
			if t.EventTrigger != nil && t.Event == eventName {
				all = append(all, fn)
				break
			}
		}
	}
	return all, nil
}
