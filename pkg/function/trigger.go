package function

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"github.com/gosimple/slug"
	"github.com/inngest/event-schemas/pkg/fakedata"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/expressions"
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

	return e.Definition.Validate(ctx)
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

// GenerateTriggerData generates deterministic random data from a single
// event trigger given the list of triggers.  The selected trigger must
// contain a cue schema definition.
func GenerateTriggerData(ctx context.Context, seed int64, triggers []Trigger) (event.Event, error) {
	evtTriggers := []Trigger{}
	for _, t := range triggers {
		if t.EventTrigger != nil {
			evtTriggers = append(evtTriggers, t)
		}
	}

	if len(evtTriggers) == 0 {
		return event.Event{}, nil
	}

	rng := rand.New(rand.NewSource(seed))
	i := rng.Intn(len(evtTriggers))
	if evtTriggers[i].EventTrigger.Definition == nil {
		return event.Event{}, nil
	}

	def, err := evtTriggers[i].EventTrigger.Definition.Cue(ctx)
	if err != nil {
		return event.Event{}, err
	}

	r := &cue.Runtime{}
	inst, err := r.Compile(".", def)
	if err != nil {
		return event.Event{}, err
	}

	fakedata.DefaultOptions.Rand = rng

	val, err := fakedata.Fake(ctx, inst.Value())
	if err != nil {
		return event.Event{}, err
	}

	mapped := map[string]interface{}{}
	err = val.Decode(&mapped)
	if err != nil {
		return event.Event{}, err
	}
	if _, ok := mapped["name"].(string); !ok {
		return event.Event{}, fmt.Errorf("no event name generated")
	}

	evt := event.Event{
		Name:      mapped["name"].(string),
		Timestamp: time.Now().UnixMilli(),
	}

	if mapped["data"] != nil {
		evt.Data = mapped["data"].(map[string]interface{})
	}
	if mapped["user"] != nil {
		evt.User = mapped["user"].(map[string]interface{})
	}
	if id, ok := mapped["id"].(string); ok {
		evt.ID = id
	}

	return evt, nil
}
