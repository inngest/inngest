package expressions

import (
	"sort"
	"time"

	"github.com/google/cel-go/interpreter"
	"github.com/inngest/inngest/pkg/dateutil"
)

// timeDecorator returns a decorator for inspecting times used within an expression
//
// It returns a mutable timeRefs pointer which records every date and time seen within
// an expression once evaluated.
func timeDecorator(act interpreter.PartialActivation) (*timeRefs, interpreter.InterpretableDecorator) {
	// Create a new dispatcher with all functions added
	dispatcher := interpreter.NewDispatcher()
	overloads := celOverloads()
	_ = dispatcher.Add(overloads...)

	tr := &timeRefs{}

	return tr, func(i interpreter.Interpretable) (interpreter.Interpretable, error) {
		defer func() {
			// Prevent any unary call from panicing, but handle silently:  this is
			// best effort and isn't critical;  it shouldn't prevent the expression
			// from being evaluated.
			//
			// This is purely a precaution.
			_ = recover()
		}()

		// This is a straight up attribute.  See if it's a time, and if so add
		// it to the list.
		if attr, ok := i.(interpreter.InterpretableAttribute); ok {
			rv := attr.Eval(act).Value()
			if time, ok := rv.(time.Time); ok {
				tr.Add(time)
			}
			if time, err := dateutil.Parse(rv); err == nil {
				tr.Add(time)
			}
			return i, nil
		}

		// This may be a helper function we've added, such as "now_plus" or
		// date().  We only support unary helpers for timestamps, so if this
		// is a unary function which matches a helper name, evaluate it with
		// the single argument and see if it's a time.
		if call, ok := i.(interpreter.InterpretableCall); ok {
			fn, ok := dispatcher.FindOverload(call.Function())
			if !ok {
				return i, nil
			}

			switch call.Function() {
			case "now_minus", "now_plus", "now", "date":
				args := call.Args()
				if fn.Unary != nil {
					rv := fn.Unary(args[0].Eval(act)).Value()
					if time, ok := rv.(time.Time); ok {
						tr.Add(time)
					}
					if time, err := dateutil.Parse(rv); err == nil {
						tr.Add(time)
					}
				}
			}
		}

		// Continue as normal.
		return i, nil
	}
}

// timeRefs stores a list of times referenced within an expression
type timeRefs []time.Time

func (d timeRefs) Len() int      { return len(d) }
func (d timeRefs) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d timeRefs) Less(i, j int) bool {
	return d[i].Before(d[j])
}

func (d *timeRefs) Add(t time.Time) {
	copied := append(*d, t)
	*d = copied
}

func (d timeRefs) Next() *time.Time {
	now := time.Now().Add(time.Second)
	sort.Sort(d)
	for _, t := range d {
		if t.After(now) {
			return &t
		}
	}
	return nil
}
