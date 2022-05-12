package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"cuelang.org/go/cue"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/event-schemas/pkg/fakedata"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/execution/actionloader"
	"github.com/inngest/inngestctl/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngestctl/pkg/execution/executor"
	"github.com/inngest/inngestctl/pkg/execution/runner"
	"github.com/inngest/inngestctl/pkg/execution/state/inmemory"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/spf13/cobra"
)

var runSeed int64

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a serverless function locally",
		Example: "inngestctl run",
		Run:     doRun,
	}

	cmd.Flags().Int64Var(&runSeed, "seed", 0, "Sets the seed for deterministically generating random events")
	return cmd
}

func doRun(cmd *cobra.Command, args []string) {
	path := "."
	if len(args) == 1 {
		path = args[0]
	}

	fn, err := function.Load(path)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("\n" + cli.RenderError("No inngest.json or inngest.cue file found in your current directory") + "\n")
		os.Exit(1)
		return
	}

	err = buildImg(cmd.Context(), *fn)
	if err != nil {
		// This should already have been printed to the terminal.
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = runExecutor(cmd.Context(), *fn)
	if err != nil {
		// This should already have been printed to the terminal.
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func buildImg(ctx context.Context, fn function.Function) error {
	a, _, _ := fn.Actions(ctx)

	ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
		QuitOnComplete: true,
		BuildOpts: dockerdriver.BuildOpts{
			Path: ".",
			Tag:  a[0].DSN,
		},
	})
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	return nil
}

func runExecutor(ctx context.Context, fn function.Function) error {
	al := actionloader.NewMemoryLoader()

	// XXX: HAVE ONE SINGLE FN/FLOW/STEP DEFINITION PLEASE FFS.
	flow, err := fn.Workflow(ctx)
	if err != nil {
		return err
	}

	avs, _, _ := fn.Actions(ctx)
	for _, a := range avs {
		al.Add(a)
	}

	// Create a new state manager.
	sm := inmemory.NewStateManager()

	// Create our drivers.
	dd, err := dockerdriver.New()
	if err != nil {
		return fmt.Errorf("error creating action loader: %w", err)
	}

	// Create an executor with the state manager and drivers.
	exec, err := executor.NewExecutor(
		executor.WithStateManager(sm),
		executor.WithActionLoader(al),
		executor.WithRuntimeDrivers(
			dd,
		),
	)
	if err != nil {
		return fmt.Errorf("error creating executor: %w", err)
	}

	// Create a high-level runner, which executes our functions.
	r := runner.NewInMemoryRunner(sm, exec)
	id, err := r.NewRun(ctx, *flow)
	if err != nil {
		log.Fatalf("error creating new run: %s", err)
	}

	if err := r.Execute(ctx, *id); err != nil {
		log.Fatalf("error creating executor: %s", err)
	}

	return nil
}

// runFunction builds the function's images and runs the function.
func runFunction(ctx context.Context, fn function.Function) error {
	if runSeed <= 0 {
		rand.Seed(time.Now().UnixNano())
		runSeed = rand.Int63n(1_000_000)
	}

	evt, err := event(ctx, fn)
	if err != nil {
		return err
	}

	actions, _, err := fn.Actions(ctx)
	if err != nil {
		return err
	}
	if len(actions) != 1 {
		return fmt.Errorf("running step-functions locally is not yet supported")
	}

	// Build the image.
	ui, err := cli.NewRunUI(ctx, cli.RunUIOpts{
		Function: fn,
		Action:   actions[0],
		Event:    evt,
		Seed:     runSeed,
	})
	if err != nil {
		return err
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		return err
	}
	// So we can exit with a non-zero code.
	return ui.Error()
}

// event retrieves the event for use within testing the function.  It first checks stdin
// to see if we're passed an event, or resorts to generating a fake event based off of
// the function's event type.
func event(ctx context.Context, fn function.Function) (map[string]interface{}, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// Read stdin
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		evt := scanner.Bytes()

		data := map[string]interface{}{}
		err := json.Unmarshal(evt, &data)
		return data, err
	}

	return fakeEvent(ctx, fn)
}

// fakeEvent finds event triggers within the function definition, then chooses
// a random trigger from the definitions and generates fake data for the event.
func fakeEvent(ctx context.Context, fn function.Function) (map[string]interface{}, error) {
	evtTriggers := []function.Trigger{}
	for _, t := range fn.Triggers {
		if t.EventTrigger != nil {
			evtTriggers = append(evtTriggers, t)
		}
	}

	if len(evtTriggers) == 0 {
		return map[string]interface{}{}, nil
	}

	rng := rand.New(rand.NewSource(runSeed))

	i := rng.Intn(len(evtTriggers))
	if evtTriggers[i].EventTrigger.Definition == nil {
		return map[string]interface{}{}, nil
	}

	def, err := evtTriggers[i].EventTrigger.Definition.Cue()
	if err != nil {
		return nil, err
	}

	r := &cue.Runtime{}
	inst, err := r.Compile(".", def)
	if err != nil {
		return nil, err
	}

	fakedata.DefaultOptions.Rand = rng

	val, err := fakedata.Fake(ctx, inst.Value())
	if err != nil {
		return nil, err
	}

	mapped := map[string]interface{}{}
	err = val.Decode(&mapped)

	return mapped, err
}
