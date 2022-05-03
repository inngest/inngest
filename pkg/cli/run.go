package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/inngest/inngestctl/pkg/runtime"
	"github.com/inngest/inngestctl/pkg/runtime/docker"
	"github.com/inngest/inngestctl/pkg/runtime/http"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/term"
)

type RunUIOpts struct {
	Action   inngest.ActionVersion
	Event    map[string]interface{}
	Seed     int64
	Function function.Function
}

func NewRunUI(ctx context.Context, opts RunUIOpts) (*RunUI, error) {
	var build *BuilderUI

	if opts.Action.Runtime.RuntimeType() == "docker" {
		var err error
		build, err = NewBuilder(ctx, BuilderUIOpts{
			BuildOpts: docker.BuildOpts{
				Path: ".",
				Tag:  opts.Action.DSN,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	r := &RunUI{
		ctx:    ctx,
		action: opts.Action,
		event:  opts.Event,
		seed:   opts.Seed,
		fn:     opts.Function,
		build:  build,
	}
	return r, nil
}

// RunUI is used to render CLI output when running an action locally.
type RunUI struct {
	// ctx stores the parent context from creating the UI model.  This is
	// used when running the function to capture cnacellation signals.
	ctx context.Context

	// action is the action we're running
	action inngest.ActionVersion
	// event stores the event data used as a trigger for the function.
	event map[string]interface{}
	// seed is the seed used to generate fake data
	seed int64
	// function is the function definition.
	fn function.Function

	// build stores a reference to the BuildUI component, rendering the
	// UI for building the function before running.
	build *BuilderUI

	err error

	// An atomic lock for starting the container.
	started int32

	// duration stores how long the function took to execute.
	duration time.Duration
	done     bool
	// response stores the response for the function
	response []byte
}

// Error returns the error from building or running the function, if part of the process failed.
func (r *RunUI) Error() error {
	return r.err
}

func (r *RunUI) Init() tea.Cmd {
	if r.build == nil {
		return nil
	}
	cmd := r.build.Init()
	return cmd
}

// run performs the running of the function.
func (r *RunUI) run(ctx context.Context) {
	var (
		exec runtime.Executor
		err  error
	)

	switch r.action.Runtime.RuntimeType() {
	case "docker":
		exec, err = docker.NewExecutor()
	case "http":
		exec = http.DefaultExecutor
	}

	if err != nil {
		r.err = err
		return
	}

	start := time.Now()
	resp, err := exec.Execute(ctx, r.action, map[string]interface{}{
		"event": r.event,
	})
	r.duration = time.Since(start)
	if err != nil {
		r.err = err
		return
	}
	r.done = true
	r.response, _ = json.Marshal(resp)
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

	// Send updates to Build so that the builder can update.  This is heirarchical;
	// Update is called via tea's manager, and we need to forward those to sub-UI
	// components.
	if r.build != nil {
		_, cmd := r.build.Update(msg)
		cmds = append(cmds, cmd)

		if r.build.Builder.Done() && r.build.Builder.Error() == nil && atomic.LoadInt32(&r.started) == 0 {
			// The build completed.  Run the function.
			atomic.StoreInt32(&r.started, 1)
			go func() {
				r.run(r.ctx)
			}()
		}

		if r.build.Builder.Done() && r.build.Builder.Error() != nil {
			// There was a build error.  Store the error so that the parent can os.Exit(1),
			// and quit the UI loop.
			r.err = r.build.Builder.Error()
			cmds = append(cmds, tea.Quit)
		}
	} else {
		go func() {
			r.run(r.ctx)
		}()
	}

	if r.done || r.duration != 0 || r.err != nil {
		// The fn has ran.
		cmds = append(cmds, tea.Quit)
	}

	cmds = append(cmds, tea.Tick(30*time.Millisecond, func(t time.Time) tea.Msg {
		if r.done || r.duration != 0 || r.err != nil {
			return tea.Quit
		}
		return nil
	}))

	return r, tea.Batch(cmds...)
}

func (r *RunUI) View() string {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))

	s := &strings.Builder{}

	if r.build != nil {
		s.WriteString(r.build.View())
		if !r.build.Builder.Done() {
			return s.String()
		}
	}

	if r.seed > 0 {
		s.WriteString(TextStyle.Copy().Padding(1, 0, 0, 0).Render("Running your function using seed "))
		s.WriteString(BoldStyle.Copy().Render(fmt.Sprintf("%d", r.seed)))
		s.WriteString("\n")
	} else {
		s.WriteString(TextStyle.Copy().Padding(1, 0, 0, 0).Render("Running your function..."))
		s.WriteString("\n")
	}

	if r.err != nil {
		s.WriteString(RenderError("There was an error running your function: "+r.err.Error()) + "\n")
		return s.String()
	}

	if r.duration == 0 {
		// We have't ran the action yet.
		return s.String()
	}

	s.WriteString(
		BoldStyle.Copy().Foreground(Green).Padding(0, 0, 1, 0).Render("Function complete"),
	)
	s.WriteString("\n")

	input, _ := json.Marshal(r.event)
	s.WriteString(TextStyle.Copy().Foreground(Feint).Render("Input:"))
	s.WriteString("\n")
	s.WriteString(TextStyle.Copy().Foreground(Feint).Render(wrap.String(string(input), width)))
	s.WriteString("\n")
	s.WriteString("\n")
	s.WriteString(TextStyle.Copy().Foreground(Feint).Render("Output:"))
	s.WriteString("\n")
	s.WriteString(TextStyle.Copy().Padding(0, 0, 1, 0).Render(string(r.response)))

	return s.String()
}
