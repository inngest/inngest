package base_cqrs

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/inngest/inngest/pkg/inngest"
)

// Functions returns all functions as inngest functions.
func (w wrapper) Functions(ctx context.Context) ([]inngest.Function, error) {
	all, _ := w.GetFunctions(ctx)
	funcs := make([]inngest.Function, len(all))
	for n, i := range all {
		f := inngest.Function{}
		_ = json.Unmarshal([]byte(i.Config), &f)
		funcs[n] = f
	}
	return funcs, nil
}

// FunctionsScheduled returns all scheduled functions available.
func (w wrapper) FunctionsScheduled(ctx context.Context) ([]inngest.Function, error) {
	// TODO: Make less naive by storing triggers and caching.
	fns, err := w.Functions(ctx)
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
func (w wrapper) FunctionsByTrigger(ctx context.Context, eventName string) ([]inngest.Function, error) {

	matchingTriggers := matchingTriggerNames(eventName)

	// TODO: Make less naive by storing triggers and caching.
	fns, err := w.Functions(ctx)
	if err != nil {
		return nil, err
	}
	all := []inngest.Function{}
	for _, fn := range fns {
		for _, t := range fn.Triggers {
			if t.EventTrigger != nil {
				for _, trigger := range matchingTriggers {
					if t.Event == trigger {
						all = append(all, fn)
						break
					}
				}
			}
		}
	}
	return all, nil
}

// matchingTriggerNames returns all matching trigger names for the given event name
// including wildcards.
func matchingTriggerNames(e string) []string {
	prefixes := []string{e}

	parts := strings.Split(e, "/")
	if len(parts) > 1 {
		for n := range parts[0 : len(parts)-1] {
			prefix := strings.Join(parts[0:n+1], "/")
			prefixes = append(prefixes, prefix+"/*")
		}
	}

	parts = strings.Split(e, ".")
	if len(parts) > 1 {
		for n := range parts[0 : len(parts)-1] {
			prefix := strings.Join(parts[0:n+1], ".")
			prefixes = append(prefixes, prefix+".*")
		}
	}

	return prefixes
}
