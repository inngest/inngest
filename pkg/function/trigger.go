package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/inngest/inngest-cli/pkg/expressions"
	cron "github.com/robfig/cron/v3"
)

// Trigger represents either an event trigger or a cron trigger.  Only one is valid;  when
// defining a function within Cue we enforce that only an event or cron field can be specified.
type Trigger struct {
	*EventTrigger
	*CronTrigger
}

func (t Trigger) Validate(ctx context.Context) error {
	if t.EventTrigger == nil && t.CronTrigger == nil {
		return fmt.Errorf("A trigger must supply an event name or a cron schedule")
	}
	if t.EventTrigger != nil && t.CronTrigger != nil {
		return fmt.Errorf("A trigger cannot have both an event and a cron trigger")
	}
	if t.EventTrigger != nil {
		return t.EventTrigger.Validate(ctx)
	}
	if t.CronTrigger != nil {
		return t.CronTrigger.Validate(ctx)
	}
	// heh.  this will (should) never happen.
	return fmt.Errorf("This trigger is neither an event trigger or cron trigger.  This should never happen :D")
}

// EventTrigger is a trigger which invokes the function each time a specific event is received.
type EventTrigger struct {
	// Event is the event name which triggers the function.
	Event string `json:"event"`

	// Expression is an optional expression which must evaluate to true for the function
	// to run.
	Expression *string `json:"expression,omitempty"`

	// Definition represents the schema or type definition for the event.
	Definition *EventDefinition `json:"definition,omitempty"`
}

func (e EventTrigger) TitleName() string {
	words := strings.ReplaceAll(slug.Make(e.Event), "-", " ")
	return strings.ReplaceAll(strings.Title(words), " ", "")
}

func (e EventTrigger) Validate(ctx context.Context) error {
	if e.Event == "" {
		return fmt.Errorf("An event trigger must specify an event name")
	}

	if e.Expression != nil {
		if _, err := expressions.NewExpressionEvaluator(ctx, *e.Expression); err != nil {
			return err
		}
	}
	if e.Definition == nil {
		// TODO: Warn that we have no event definition
		return nil
	}
	return e.Definition.Validate()
}

// CronTrigger is a trigger which invokes the function on a CRON schedule.
type CronTrigger struct {
	Cron string `json:"cron"`
}

func (c CronTrigger) Validate(ctx context.Context) error {
	_, err := cron.
		NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).
		Parse(c.Cron)
	if err != nil {
		return fmt.Errorf("'%s' isn't a valid cron schedule", c.Cron)
	}
	return nil
}
