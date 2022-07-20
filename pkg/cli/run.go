package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/runner"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/function"
	"github.com/inngest/inngest-cli/pkg/service"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/term"
)

type RunUIOpts struct {
	Events    []event.Event
	Seed      int64
	Function  function.Function
	LogBuffer *bytes.Buffer
}

func NewRunUI(ctx context.Context, opts RunUIOpts) (*RunUI, error) {
	r := &RunUI{
		ctx:    ctx,
		events: opts.Events,
		seed:   opts.Seed,
		fn:     opts.Function,
		logBuf: opts.LogBuffer,
	}
	return r, nil
}

// RunUI is used to render CLI output when running an action locally.
type RunUI struct {
	// ctx stores the parent context from creating the UI model.  This is
	// used when running the function to capture cnacellation signals.
	ctx context.Context

	// event stores the event data used as a trigger for the function.
	events []event.Event

	// seed is the seed used to generate fake data
	seed int64
	// function is the function definition.
	fn function.Function

	err  error
	done bool

	runs []RunUIExecution

	// sm is the state manager used for the execution.
	sm state.Manager

	// logBuf stores the output of the logger, if we want to display this in
	// the UI (which we currently dont)
	logBuf *bytes.Buffer
}

type RunUIExecution struct {
	// id is the identifier for the execution, once started.
	id    *state.Identifier
	event *event.Event
	// duration stores how long the function took to execute.
	// duration time.Duration
	done bool
	// err      error
}

// Error returns the error from building or running the function, if part of the process failed.
func (r *RunUI) Error() error {
	return r.err
}

func (r *RunUI) Init() tea.Cmd {
	go func() {
		r.run(r.ctx)
	}()
	return nil
}

// run performs the running of the function.
func (r *RunUI) run(ctx context.Context) {
	el := &coredata.MemoryExecutionLoader{}
	if err := el.SetFunctions(ctx, []*function.Function{&r.fn}); err != nil {
		// This is a render loop, so store the error in our mutable state
		// for the View() function to render to the UI.
		r.err = err
		return
	}

	c, _ := config.Default(ctx)
	// Create a singleton queue for initializing the fn.
	q, err := c.Queue.Service.Concrete.Producer()
	if err != nil {
		r.err = err
		return
	}
	// Return the in-memory state manager that was created from our
	// derived default config.
	//
	// NOTE: Each individual config struct returns a singleton in-memory
	// service, given the config struct has not been copied.
	r.sm, r.err = c.State.Service.Concrete.Manager(ctx)
	if r.err != nil {
		return
	}

	// In order to execute the function we need to create a new executor
	// service to execute the steps of our function.  We'll manually initialize
	// a new function run.
	exec := executor.NewService(*c, executor.WithExecutionLoader(el))
	go func() {
		if err := service.Start(ctx, exec); err != nil {
			r.err = err
			return
		}
	}()

	// XXX: We need to define a readiness check with each of our services,
	// then wait here for the readiness check to pass.

	var wg sync.WaitGroup

	for _, evt := range r.events {
		wg.Add(1)

		go func(event event.Event) {
			defer wg.Done()

			runId, err := runner.Initialize(ctx, r.fn, event, r.sm, q)
			if err != nil {
				r.err = err
				return
			}
			if runId == nil {
				r.err = fmt.Errorf("no run id created")
				return
			}

			done := false

			r.runs = append(r.runs, RunUIExecution{
				id:    runId,
				done:  false,
				event: &event,
			})

			for !done {
				var run state.State

				run, r.err = r.sm.Load(ctx, *runId)
				if r.err != nil {
					return
				}

				meta := run.Metadata()
				if meta.Pending == 0 {
					done = true
					return
				}
				<-time.After(time.Millisecond)
			}
		}(evt)
	}

	wg.Wait()
	r.done = true
}

func (r *RunUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	// Enable quitting early.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			return r, tea.Quit
		}
		if msg.String() == "q" {
			return r, tea.Quit
		}
	}

	if r.done || r.err != nil {
		// The fn has ran.
		cmds = append(cmds, tea.Quit)
	}

	// Tick while the executor runs.
	cmds = append(cmds, tea.Tick(25*time.Millisecond, func(t time.Time) tea.Msg {
		if r.done || r.err != nil {
			return tea.Quit
		}
		return nil
	}))

	return r, tea.Batch(cmds...)
}

func (r *RunUI) View() string {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))

	s := &strings.Builder{}

	runCount := len(r.runs)
	hasMultipleRuns := runCount > 1

	if r.seed > 0 {
		s.WriteString(TextStyle.Copy().Padding(1, 0, 0, 0).Render("Running your function using seed "))
		s.WriteString(BoldStyle.Copy().Render(fmt.Sprintf("%d", r.seed)))
		s.WriteString("\n")
	} else if hasMultipleRuns {
		s.WriteString(TextStyle.Copy().Padding(1, 0, 0, 0).Render("Running your function..."))
		s.WriteString("\n\n")
	} else {
		s.WriteString(TextStyle.Copy().Padding(1, 0, 0, 0).Render(fmt.Sprintf("Running your function with %d recent events...", runCount)))
		s.WriteString("\n\n")
	}

	if hasMultipleRuns {
		for i, run := range r.runs {
			input, _ := json.Marshal(run.event)
			s.WriteString(BoldStyle.Render("Run #" + strconv.Itoa(i+1) + ": " + run.event.ID + "\n"))
			s.WriteString(FeintStyle.Render("Input:") + string(input) + "\n")
			s.WriteString(FeintStyle.Render("Output:") + r.RenderState(run) + "\n\n")
		}
	} else if runCount > 0 {
		run := r.runs[0]

		input, _ := json.Marshal(run.event)
		s.WriteString(FeintStyle.Render("Input:") + "\n")
		s.WriteString(TextStyle.Copy().Foreground(Feint).Render(wrap.String(string(input), width)))
		s.WriteString("\n\n")
		s.WriteString(FeintStyle.Render("Input:") + "\n")
		s.WriteString(r.RenderState(run))
	} else {
		// nothing has happened yet
		return s.String()
	}

	// s.WriteString(r.RenderState())

	if r.err != nil {
		s.WriteString(RenderError("There was an error running your function: "+r.err.Error()) + "\n")
		return s.String()
	}

	if !r.done {
		// We have't ran the action yet.
		return s.String()
	}

	// s.WriteString(
	// 	BoldStyle.Copy().Foreground(Green).Padding(0, 0, 1, 0).Render(
	// 		fmt.Sprintf("Function complete in %.2f seconds", r.duration.Seconds()),
	// 	),
	// )

	return s.String()
}

func (r *RunUI) RenderState(run RunUIExecution) string {
	// if r.id == nil {
	// 	return ""
	// }

	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	s := &strings.Builder{}

	state, err := r.sm.Load(context.Background(), *run.id)
	if err != nil {
		s.WriteString(RenderError("There was an error loading state: "+err.Error()) + "\n")
		return s.String()
	}

	output := state.Actions()
	errors := state.Errors()

	s.WriteString(BoldStyle.Render("Output") + "\n")
	if len(output) == 0 {
		s.WriteString(FeintStyle.Render("No output yet.") + "\n")
	}
	for id, data := range output {
		byt, _ := json.Marshal(data)
		s.WriteString(BoldStyle.Render(fmt.Sprintf("Step '%s'", id)))
		s.WriteString(": \n")
		s.WriteString(wrap.String(string(byt), width))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(BoldStyle.Render("Errors") + "\n")

	if len(errors) == 0 {
		s.WriteString(FeintStyle.Render("No errors 🥳") + "\n")
	}
	for id, err := range errors {
		s.WriteString(BoldStyle.Render(fmt.Sprintf("Step '%s'", id)))
		s.WriteString(": " + wrap.String(err.Error(), width))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	return s.String()
}
