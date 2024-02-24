package models

import (
	"fmt"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
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

	concurrency := 0
	if fn.Concurrency != nil {
		concurrency = fn.Concurrency.PartitionConcurrency()
	}

	return &Function{
		AppID:       f.AppID.String(),
		ID:          f.ID.String(),
		Name:        f.Name,
		Slug:        f.Slug,
		Config:      string(f.Config),
		Concurrency: concurrency,
		Triggers:    triggers,
		URL:         fn.Steps[0].URI,
	}, nil
}

func MakeFunctionRun(f *cqrs.FunctionRun) *FunctionRun {
	// TODO: Map GQL types to CQRS types and remove this.
	r := &FunctionRun{
		ID:         f.RunID.String(),
		FunctionID: f.FunctionID.String(),
		FinishedAt: f.EndedAt,
		StartedAt:  &f.RunStartedAt,
		EventID:    f.EventID.String(),
		BatchID:    f.BatchID,
	}
	if len(f.Output) > 0 {
		str := string(f.Output)
		r.Output = &str
	}
	return r
}

func ToFunctionRunStatus(s enums.RunStatus) (FunctionRunStatus, error) {
	switch s {
	case enums.RunStatusRunning:
		return FunctionRunStatusRunning, nil
	case enums.RunStatusCompleted:
		return FunctionRunStatusCompleted, nil
	case enums.RunStatusFailed:
		return FunctionRunStatusFailed, nil
	case enums.RunStatusCancelled:
		return FunctionRunStatusCancelled, nil
	case enums.RunStatusScheduled:
		return FunctionRunStatusQueued, nil
	default:
		return FunctionRunStatusRunning, fmt.Errorf("unknown run status: %d", s)
	}
}
