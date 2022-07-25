package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/config"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/service"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/term"
)

type RunUIOpts struct {
	Events    []event.Event
	Seed      int64
	Function  function.Function
	LogBuffer *bytes.Buffer
	Verbose   bool
	DebugID   *uuid.UUID
}

func NewRunUI(ctx context.Context, opts RunUIOpts) (*RunUI, error) {
	r := &RunUI{
		ctx:     ctx,
		events:  opts.Events,
		seed:    opts.Seed,
		fn:      opts.Function,
		logBuf:  opts.LogBuffer,
		verbose: opts.Verbose,
		debugID: opts.DebugID,
	}
	return r, nil
}

// RunUI is used to render CLI output when running an action locally.
type RunUI struct {
	// ctx stores the parent context from creating the UI model.  This is
	// used when running the function to capture cnacellation signals.
	ctx context.Context

	// events stores the event data used as triggers for the function(s).
	events []event.Event

	// seed is the seed used to generate fake data
	seed int64

	// function is the function definition.
	fn function.Function

	// An error that has occurred while setting up or running the function(s).
	// The process does not exit on this error; it will be used to decide status
	// code returns.
	err error

	// Represents whether or not all functions and steps have finished running.
	// This is used to determine when to end the Bubbletea process.
	done bool

	// runs is the list of function executions that are occurring. It is used to
	// store the state of each particular execution to display in the UI.
	runs []RunUIExecution

	// sm is the state manager used for the execution.
	sm state.Manager

	q queue.Queue

	// logBuf stores the output of the logger, if we want to display this in
	// the UI (which we currently dont)
	logBuf *bytes.Buffer

	// Used to decide whether to print more information when running the command.
	verbose bool

	// debugID stores the debug ID used when running in debug mode.  Debug mode
	// pauses on each edge until we manually continue.
	debugID *uuid.UUID
	// debugPauses stores all available debug pauses.
	debugPauses []*state.Pause
}

type RunUIExecution struct {
	// id is the identifier for the execution, once started.
	id *state.Identifier

	// event is the event that was used to trigger the execution.
	event *event.Event

	// When true, the function run and all of its steps have either completed or
	// errored.
	done *bool

	// Stores the order in which executions outputs were seen, in order to display
	// the appropriate output in the UI.
	seenOutput *[]string
}

// A style for the label to show that a function is running.
var runsStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#fbbf24")).Padding(0, 1)

// A style for the label to show that all of a function's steps have
// successfully run.
var passStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#84cc16")).Padding(0, 1)

// A style for the label to show that at least one of a function's steps  has
// failed to run.
var failStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#b91c1c")).Padding(0, 1)

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
	var err error
	el := &inmemorydatastore.MemoryExecutionLoader{}
	if err := el.SetFunctions(ctx, []*function.Function{&r.fn}); err != nil {
		// This is a render loop, so store the error in our mutable state
		// for the View() function to render to the UI.
		r.err = err
		return
	}

	c, err := config.Dev(ctx)
	if err != nil {
		r.err = err
		return
	}

	// Create a singleton queue for initializing the fn.
	r.q, err = c.Queue.Service.Concrete.Queue()
	if err != nil {
		r.err = err
		return
	}
	// Return the in-memory state manager that was created from our
	// derived default config.
	//
	// NOTE: Each individual config struct returns a singleton in-memory
	// service, given the config struct has not been copied.
	r.sm, err = c.State.Service.Concrete.Manager(ctx)
	if err != nil {
		r.err = err
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

	// Loop over our given events and create a new run for each one.
	for _, evt := range r.events {
		wg.Add(1)

		go func(event event.Event) {
			defer wg.Done()

			var runId *state.Identifier

			runId, err = runner.Initialize(ctx, r.fn, event, r.sm, r.q)
			if err != nil {
				r.err = err
				return
			}
			if runId == nil {
				r.err = fmt.Errorf("no run id created")
				return
			}

			done := false
			seenOutput := []string{}

			execution := RunUIExecution{
				id:         runId,
				done:       &done,
				event:      &event,
				seenOutput: &seenOutput,
			}

			r.runs = append(r.runs, execution)

			// A clunky loop to watch for the completion of the run.
			for !*execution.done {
				var run state.State

				run, err = r.sm.Load(ctx, *runId)
				if err != nil {
					r.err = err
					return
				}

				output := run.Actions()

				// If we have output, we may be displaying this in stdout.
				// We receive output as a map[string], so here we store the order in
				// which we have originally seen any new keys to ensure the render loop
				// displays the output in a consistent order.
				for id := range output {
					haveSeenKey := false

					for _, seenKey := range seenOutput {
						if seenKey == id {
							haveSeenKey = true
							break
						}
					}

					if !haveSeenKey {
						seenOutput = append(seenOutput, id)
					}
				}

				hasErrors := len(run.Errors()) > 0
				if run.Metadata().Pending == 0 || hasErrors {
					// If we're here, either all of the function's steps have completed
					// successfully, or at least one of them has errored.
					//
					// Either way, we're counting this run as complete and returning an
					// error for it if it had errors so we can return a non-zero status
					// code.
					if hasErrors {
						r.err = fmt.Errorf("run failed")
					}

					*execution.done = true
					return
				}
				<-time.After(time.Millisecond * 5)
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
		case tea.KeyEnter:
			// TODO: Consume the selected pause.
			return r, nil
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
		ctx := context.Background()

		if r.done {
			return tea.Quit
		}

		if r.debugID != nil && len(r.runs) > 0 {
			// If the number of pauses equals the number of outstanding items in this
			// function, there's no need to query again;  it only messes with allocations.
			// Check how many pauses we have outstanding.
			runID := r.runs[0]
			s, err := r.sm.Load(ctx, *runID.id)
			if err != nil {
				r.err = err
				return nil
			}
			if s.Metadata().Pending == len(r.debugPauses) {
				return nil
			}

			// Update the pauses here.  It doesn't matter that this updates pointers;
			// we tick anyways.
			it, _ := r.sm.PausesByEvent(ctx, inngest.DebugEvent)
			pauses := []*state.Pause{}
			for it.Next(ctx) {
				pause := it.Val(ctx)
				if pause.Expression != nil && strings.Contains(*pause.Expression, r.debugID.String()) {
					pauses = append(pauses, pause)
				}
			}
			r.debugPauses = pauses
		}
		return nil
	}))

	return r, tea.Batch(cmds...)
}

func (r *RunUI) View() string {
	s := &strings.Builder{}

	runCount := len(r.runs)
	hasMultipleRuns := runCount > 1

	if r.seed > 0 {
		s.WriteString("\nRunning your function using seed ")
		s.WriteString(BoldStyle.Copy().Render(fmt.Sprintf("%d", r.seed)))
		s.WriteString("\n\n")
	} else if hasMultipleRuns {
		s.WriteString(fmt.Sprintf("Running your function with %d recent events...", runCount))
		s.WriteString("\n\n")
	} else {
		s.WriteString("Running your function...")
		s.WriteString("\n\n")
	}

	if len(r.runs) > 0 {
		for _, run := range r.runs {
			s.WriteString(r.RenderState(run) + "\n")
		}
	}

	if r.err != nil {
		s.WriteString(RenderError(r.err.Error()) + "\n")
	}

	return s.String()
}

// Renders the UI for a particular given `run`.
func (r *RunUI) RenderState(run RunUIExecution) string {
	if run.id == nil {
		return ""
	}

	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	s := &strings.Builder{}

	state, err := r.sm.Load(context.Background(), *run.id)
	if err != nil {
		s.WriteString(RenderError("There was an error loading state: "+err.Error()) + "\n")
		return s.String()
	}

	errors := state.Errors()
	metadata := state.Metadata()

	done := *run.done
	passed := done && len(errors) == 0
	failed := done && len(errors) > 0

	status := runsStyle.Render("RUNNING")
	runId := run.event.ID
	info := FeintStyle.Render(fmt.Sprintf("%d step(s) running", metadata.Pending))

	if passed {
		status = passStyle.Render("SUCCESS")
		info = FeintStyle.Render("No errors")
		// TODO Log the duration of the run here too
	} else if failed {
		status = failStyle.Render("FAILURE")
		info = FeintStyle.Render(fmt.Sprintf("%d error(s)", len(errors)))
	}

	s.WriteString(strings.Join([]string{status, runId, info}, " "))

	// Now we've rendered a simple line for the run, decide whether we want to
	// log input/output data too.
	if failed || r.verbose {
		input, _ := json.Marshal(run.event)
		s.WriteString("\n\n")
		s.WriteString(TextStyle.Copy().Render("Input:") + "\n")
		s.WriteString(FeintStyle.Render(wrap.String(string(input), width)))
	}
	if failed {
		for _, err := range errors {
			s.WriteString("\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render(wrap.String(err.Error(), width)) + "\n")
		}
	}
	if failed || r.verbose {
		output := state.Actions()

		for _, key := range *run.seenOutput {
			data := output[key]

			if data != nil {
				byt, _ := json.Marshal(data)
				s.WriteString("\n\n" + BoldStyle.Render(fmt.Sprintf("Step '%s' output:", key)) + "\n")
				s.WriteString(FeintStyle.Render((wrap.String(string(byt), width))))
			}
		}

		s.WriteString("\n")
	}

	if r.debugID != nil {
		s.WriteString("\n")
		for _, pause := range r.debugPauses {
			s.WriteString(fmt.Sprintf("Paused on %s\n", pause.Incoming))
		}
	}

	return s.String()
}
