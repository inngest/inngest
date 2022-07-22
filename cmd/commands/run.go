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
	"github.com/inngest/inngest-cli/inngest/state"
	"github.com/inngest/inngest-cli/pkg/cli"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/function"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

var runSeed int64
var replayCount int64

type runFunctionOpts struct {
	eventFunc func() ([]event.Event, error)

	// If in replay mode, this is the number of recent events to fetch.
	// fetchRecentEvents int64

	// If true, prints extra information about step input/output to stdout during
	// a function's run.
	verbose bool

	// If non-empty, this is a specific event ID to fetch and replay.
	// fetchEventId *ulid.ULID

	// If non-empty, this is the name of the event trigger to use. If this is not
	// provided, we will try to infer the trigger from the function definition.
	// triggerName string
}

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a serverless function locally",
		Example: "inngest run",
		Run:     doRun,
	}

	cmd.Flags().StringP("trigger", "t", "", "Specifies the event trigger to use if there are multiple configured")
	cmd.Flags().Bool("event-only", false, "Prints the generated event to use without running the function")
	cmd.Flags().Int64Var(&runSeed, "seed", 0, "Sets the seed for deterministically generating random events")
	cmd.Flags().BoolP("replay", "r", false, "Enables replay mode to replay real recent events")
	cmd.Flags().Int64VarP(&replayCount, "count", "c", 10, "Number of events to replay in replay mode")
	cmd.Flags().StringP("event-id", "e", "", "Specifies a specific event to replay in replay mode")

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

	if cmd.Flag("event-only").Value.String() == "true" {
		evt, _ := fakeEvent(cmd.Context(), *fn, "")
		out, _ := json.Marshal(evt)
		fmt.Println(string(out))
		return
	}

	if err = buildImg(cmd.Context(), *fn); err != nil {
		// This should already have been printed to the terminal.
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}

	triggerName := cmd.Flag("trigger").Value.String()
	hasVerboseFlag := cmd.Flag("verbose").Value.String() == "true"
	isReplayMode := cmd.Flag("replay").Value.String() == "true"

	var fetchRecentEventCount int64 = 0

	if isReplayMode {
		fetchRecentEventCount = replayCount
	}

	var fetchEventId *ulid.ULID
	rawEventId := cmd.Flag("event-id").Value.String()

	if rawEventId != "" {
		eventId, err := ulid.ParseStrict(rawEventId)
		if err != nil {
			fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
			os.Exit(1)
		}

		fetchEventId = &eventId
	}

	ctx := cmd.Context()
	var eventFunc func() ([]event.Event, error)

	if isReplayMode {
		if fetchEventId != nil {
			eventFunc = singleReplayEventLoader(ctx, fetchEventId)
		} else {
			eventFunc = multiReplayEventLoader(ctx, triggerName, fetchRecentEventCount)
		}
	} else {
		eventFunc = generatedEventLoader()
	}

	opts := runFunctionOpts{
		verbose:   hasVerboseFlag,
		eventFunc: eventFunc,
	}

	if err = runFunction(cmd.Context(), *fn, opts); err != nil {
		os.Exit(1)
	}
}

func singleReplayEventLoader(ctx context.Context, eventId *ulid.ULID) func() ([]event.Event, error) {
	return func() ([]event.Event, error) {
		s := state.RequireState(ctx)

		ws, err := state.Workspace(ctx)
		if err != nil {
			return nil, err
		}

		archivedEvent, err := s.Client.RecentEvent(ctx, ws.ID, *eventId)
		if err != nil {
			return nil, err
		}

		if archivedEvent == nil {
			return nil, fmt.Errorf("no events found for event ID %s", eventId)
		}

		evt, err := archivedEvent.MarshalToEvent()
		if err != nil {
			return nil, err
		}

		return []event.Event{*evt}, nil
	}
}

func multiReplayEventLoader(ctx context.Context, triggerName string, count int64) func() ([]event.Event, error) {
	return func() ([]event.Event, error) {
		if triggerName == "" {
			return nil, fmt.Errorf("triggerName is required")
		}

		if count <= 0 {
			return nil, fmt.Errorf("count must be >0")
		}

		s := state.RequireState(ctx)

		ws, err := state.Workspace(ctx)
		if err != nil {
			return nil, err
		}

		archivedEvents, err := s.Client.RecentEvents(ctx, ws.ID, triggerName, count)
		if err != nil {
			return nil, err
		}

		if len(archivedEvents) == 0 {
			return nil, fmt.Errorf("no events found for trigger %s", triggerName)
		}

		events := []event.Event{}

		for _, archivedEvent := range archivedEvents {
			evt, err := archivedEvent.MarshalToEvent()
			if err != nil {
				return nil, err
			}

			events = append(events, *evt)
		}

		return events, nil
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
func runFunction(ctx context.Context, fn function.Function, opts runFunctionOpts) error {
	evts, err := opts.eventFunc()
	if err != nil {
		return err
	}

	if opts.fetchEventId != nil {
		// We're replaying a specific event, so let's fetch it.
		evt, err := fetchEvent(ctx, *opts.fetchEventId)
		if err != nil {
			return err
		}

		if evt == nil {
			return fmt.Errorf("event not found: %s", *opts.fetchEventId)
		}

		evts = []event.Event{*evt}
	} else if opts.fetchRecentEvents > 0 {
		// Here we're replaying at least 1 recent event.
		triggerName := opts.triggerName

		// We need there to be one trigger name here; if we haven't been given one,
		// we might be able to find it in the function definition.
		if triggerName == "" {
			if len(fn.Triggers) == 0 {
				return fmt.Errorf("no trigger specified and no triggers found in function")
			}

			if len(fn.Triggers) > 1 {
				return fmt.Errorf("no trigger specified and multiple triggers found in function; provide a single trigger with the --trigger flag")
			}

			triggerName = fn.Triggers[0].Event

			if triggerName == "" {
				return fmt.Errorf("no trigger specified and could not find trigger in function definition")
			}
		}

		evts, err = fetchRecentEvents(ctx, triggerName, int64(opts.fetchRecentEvents))
		if err != nil {
			return err
		}

		if len(evts) == 0 {
			return fmt.Errorf("no events found for trigger %s", triggerName)
		}
	} else {
		// We're not replaying, so use a generated event instead.
		usingGeneratedEvent = true
		evts, err = generateEvents(ctx, fn, opts.triggerName)
		if err != nil {
			return err
		}

		if len(evts) == 0 {
			return fmt.Errorf("no events could be generated")
		}
	}

	// NOTE: The runner, executor, etc. uses logger from context.  Bubbletea
	// REALLY doesnt like it when you start logging to stderr/stdout;  it controls
	// the output.
	//
	// Here, we must create a new logger which writes to a buffer.
	buf := bytes.NewBuffer(nil)
	log := logger.Buffered(buf)
	ctx = logger.With(ctx, *log)

	if usingGeneratedEvent && runSeed <= 0 {
		rand.Seed(time.Now().UnixNano())
		runSeed = rand.Int63n(1_000_000)
	}

	// Run the function.
	ui, err := cli.NewRunUI(ctx, cli.RunUIOpts{
		Function:  fn,
		Events:    evts,
		Seed:      runSeed,
		LogBuffer: buf,
		Verbose:   opts.verbose || len(evts) == 1,
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
func generateEvents(ctx context.Context, fn function.Function, triggerName string) ([]event.Event, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return []event.Event{}, err
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// Read stdin
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		evt := scanner.Bytes()

		data := event.Event{}
		err := json.Unmarshal(evt, &data)
		return []event.Event{data}, err
	}

	fakedEvent, err := fakeEvent(ctx, fn, triggerName)
	if err != nil {
		return nil, err
	}

	return []event.Event{fakedEvent}, nil
}

// fakeEvent finds event triggers within the function definition, then chooses
// a random trigger from the definitions and generates fake data for the event.
func fakeEvent(ctx context.Context, fn function.Function, triggerName string) (event.Event, error) {
	triggers := []function.Trigger{}
	for _, t := range fn.Triggers {
		if t.EventTrigger != nil && (triggerName == "" || triggerName == t.EventTrigger.Event) {
			triggers = append(triggers, t)
		}
	}
	return function.GenerateTriggerData(ctx, runSeed, triggers)
}
