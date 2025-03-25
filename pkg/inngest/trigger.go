package inngest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/expressions"
	cron "github.com/robfig/cron/v3"
)

// Triggerable represents a single or multiple triggers for a function.
type Triggerable interface {
	Triggers() []Trigger
}

type MultipleTriggers []Trigger

func (m MultipleTriggers) Triggers() []Trigger {
	return m
}

func (m MultipleTriggers) Validate(ctx context.Context) error {
	var err error

	if len(m) < 1 {
		err = multierror.Append(err, fmt.Errorf("At least one trigger is required"))
	} else if len(m) > consts.MaxTriggers {
		err = multierror.Append(err, fmt.Errorf("This function exceeds the max number of triggers: %d", consts.MaxTriggers))
	}

	seen := make(map[string]struct{})

	for _, t := range m {
		key, herr := t.Hash()
		if herr != nil {
			return fmt.Errorf("failed to hash trigger: %w", herr)
		}

		if _, ok := seen[key]; ok {
			err = multierror.Append(err, fmt.Errorf("duplicate trigger %s", t.Name()))
		}
		seen[key] = struct{}{}

		if terr := t.Validate(ctx); terr != nil {
			err = multierror.Append(err, terr)
		}
	}

	return err
}

// Trigger represents either an event trigger or a cron trigger.  Only one is valid;  when
// defining a function within Cue we enforce that only an event or cron field can be specified.
type Trigger struct {
	*EventTrigger
	*CronTrigger
}

func (t Trigger) Triggers() []Trigger {
	return []Trigger{t}
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

// Hash returns a string hashed key for the trigger based on its type and
// arguments.
func (t Trigger) Hash() (string, error) {
	byt, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	hash := xxhash.Sum64(byt)

	return fmt.Sprintf("%x", hash), nil
}

// Name returns a human-readable name for the trigger.
func (t Trigger) Name() string {
	if t.EventTrigger != nil {
		return fmt.Sprintf("event: %s", t.EventTrigger.Event)
	}
	if t.CronTrigger != nil {
		return fmt.Sprintf("cron: %s", t.CronTrigger.Cron)
	}
	return "Unknown"
}

// EventTrigger is a trigger which invokes the function each time a specific event is received.
type EventTrigger struct {
	// Event is the event name which triggers the function.
	Event string `json:"event"`

	// Expression is an optional expression which must evaluate to true for the function
	// to run.
	Expression *string `json:"expression,omitempty"`
}

func (e EventTrigger) TitleName() string {
	joiner := "_"
	replacements := []string{".", "/", "-"}

	rep := e.Event
	for k, v := range replacements {
		rep = strings.ReplaceAll(rep, v, strings.Repeat(joiner, k+1))
	}

	words := strings.Split(rep, joiner)
	for i, w := range words {
		words[i] = strings.Title(w)
	}

	return strings.Join(words, joiner)
}

func (e EventTrigger) Validate(ctx context.Context) error {
	if e.Event == "" {
		return fmt.Errorf("An event trigger must specify an event name")
	}

	if e.Expression != nil {
		if err := expressions.Validate(ctx, *e.Expression); err != nil {
			return fmt.Errorf("invalid trigger expression on '%s': %w", e.Event, err)
		}
	}
	return nil
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
