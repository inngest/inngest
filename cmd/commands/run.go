package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/cli"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/function"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/spf13/cobra"
)

var runSeed int64

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a serverless function locally",
		Example: "inngest run",
		Run:     doRun,
	}

	cmd.Flags().String("event", "", "Specifies the event trigger to use if there are multiple configured")
	cmd.Flags().Bool("event-only", false, "Prints the generated event to use without running the function")
	cmd.Flags().Int64Var(&runSeed, "seed", 0, "Sets the seed for deterministically generating random events")
	return cmd
}

func doRun(cmd *cobra.Command, args []string) {
	path := "."
	if len(args) == 1 {
		path = args[0]
	}

	fn, err := function.Load(cmd.Context(), path)
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
		return
	}

	if err = buildImg(cmd.Context(), *fn); err != nil {
		// This should already have been printed to the terminal.
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}

	eventName := cmd.Flag("event").Value.String()
	if err = runFunction(cmd.Context(), *fn, eventName); err != nil {
		// This should already have been printed to the terminal.
		os.Exit(1)
	}
}

func buildImg(ctx context.Context, fn function.Function) error {
	a, _, _ := fn.Actions(ctx)
	if a[0].Runtime.RuntimeType() != inngest.RuntimeTypeDocker {
		return nil
	}

	opts, err := dockerdriver.FnBuildOpts(ctx, fn)
	if err != nil {
		return err
	}

	ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
		QuitOnComplete: true,
		BuildOpts:      opts,
	})
	if err != nil {
		return err
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		return err
	}
	return ui.Error()
}

// runFunction builds the function's images and runs the function.
func runFunction(ctx context.Context, fn function.Function, eventName string) error {
	if runSeed <= 0 {
		rand.Seed(time.Now().UnixNano())
		runSeed = rand.Int63n(1_000_000)
	}

	evt, err := generateEvent(ctx, fn, eventName)
	if err != nil {
		return err
	}

	// NOTE: The runner, executor, etc. uses logger from context.  Bubbletea
	// REALLY doesnt like it when you start logging to stderr/stdout;  it controls
	// the output.
	//
	// Here, we must create a new logger which writes to a buffer.
	buf := bytes.NewBuffer(nil)
	log := logger.Buffered(buf)
	ctx = logger.With(ctx, *log)

	// Run the function.
	ui, err := cli.NewRunUI(ctx, cli.RunUIOpts{
		Function:  fn,
		Event:     evt,
		Seed:      runSeed,
		LogBuffer: buf,
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

// generateEvent retrieves the event for use within testing the function.  It first checks stdin
// to see if we're passed an event, or resorts to generating a fake event based off of
// the function's event type.
func generateEvent(ctx context.Context, fn function.Function, eventName string) (event.Event, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return event.Event{}, err
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// Read stdin
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		evt := scanner.Bytes()

		data := event.Event{}
		err := json.Unmarshal(evt, &data)
		return data, err
	}

	return fakeEvent(ctx, fn, eventName)
}

// fakeEvent finds event triggers within the function definition, then chooses
// a random trigger from the definitions and generates fake data for the event.
func fakeEvent(ctx context.Context, fn function.Function, eventName string) (event.Event, error) {
	triggers := []function.Trigger{}
	for _, t := range fn.Triggers {
		if t.EventTrigger != nil && (eventName == "" || eventName == t.EventTrigger.Event) {
			triggers = append(triggers, t)
		}
	}
	return function.GenerateTriggerData(ctx, runSeed, triggers)
}
