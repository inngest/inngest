package models

import (
	"github.com/inngest/inngest/pkg/cqrs"
)

func MakeFunction(f *cqrs.Function) (*Function, error) {
	fn, err := f.InngestFunction()
	if err != nil {
		return nil, err
	}

	triggers := make([]*FunctionTrigger, len(fn.Triggers))
	for n, t := range fn.Triggers {
		var (
			val string
			typ FunctionTriggerTypes
		)
		if t.CronTrigger != nil {
			typ = FunctionTriggerTypesCron
			val = t.Cron
		}
		if t.EventTrigger != nil {
			typ = FunctionTriggerTypesEvent
			val = t.Event
		}
		triggers[n] = &FunctionTrigger{
			Type:  typ,
			Value: val,
		}
	}

	return &Function{
		ID:          f.ID.String(),
		Name:        f.Name,
		Slug:        f.Slug,
		Config:      f.Config,
		Concurrency: fn.ConcurrencyLimit(),
		Triggers:    triggers,
		URL:         fn.Steps[0].URI,
	}, nil
}
