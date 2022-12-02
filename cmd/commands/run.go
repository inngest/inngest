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
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

var runSeed int64
var replayCount int64

type runFunctionOpts struct {
	// A function that loads the events to be run locally.
	eventFunc func() ([]event.Event, error)

	// If true, prints extra information about step input/output to stdout during
	// a function's run.
	verbose bool
}

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Hidden:  true,
		Short:   "Run a serverless function locally",
		Example: "inngest run",
		Run:     doRun,
	}

	cmd.Flags().StringP("trigger", "t", "", "Specifies the event trigger to use if there are multiple configured")
	cmd.Flags().Int64Var(&runSeed, "seed", 0, "Sets the seed for deterministically generating random events")
	cmd.Flags().BoolP("replay", "r", false, "Enables replay mode to replay real recent events")
	cmd.Flags().Int64VarP(&replayCount, "count", "c", 10, "Number of events to replay in replay mode")
	cmd.Flags().StringP("event-id", "e", "", "Specifies a specific event to replay in replay mode")
	cmd.Flags().BoolP("snapshot", "s", false, "Returns found or generated events as JSON instead of running them")

	return cmd
}

func doRun(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	path := "."
	if len(args) == 1 {
		path = args[0]
	}

	fn, err := function.Load(ctx, path)
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
		return
	}

	snapshotMode := cmd.Flag("snapshot").Value.String() == "true"

	if !snapshotMode {
		if err = buildImg(ctx, *fn); err != nil {
			// This should already have been printed to the terminal.
			fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
			os.Exit(1)
		}
	}

	triggerName := cmd.Flag("trigger").Value.String()
	hasVerboseFlag := cmd.Flag("verbose").Value.String() == "true"
	isReplayMode := cmd.Flag("replay").Value.String() == "true"
	rawEventId := cmd.Flag("event-id").Value.String()

	// In order to improve the dev UX, if there's no trigger provided we should
	// inspect the function and check to see if the fn only has one trigger.  If
	// so, use that trigger - it's the only option.
	if triggerName == "" && len(fn.Triggers) == 1 && fn.Triggers[0].EventTrigger != nil {
		triggerName = fn.Triggers[0].Event
	}

	eventFunc := generatedEventLoader(ctx, *fn, triggerName)
	if isReplayMode {
		// Replay N events.
		eventFunc = multiReplayEventLoader(ctx, triggerName, replayCount)
		if rawEventId != "" {
			// We're fetching a single ID.
			id, err := ulid.ParseStrict(rawEventId)
			if err != nil {
				fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
				os.Exit(1)
			}
			eventFunc = singleReplayEventLoader(ctx, &id)
		}
	}

	if cmd.Flag("snapshot").Value.String() == "true" {
		err := snapshotEvents(ctx, eventFunc)
		if err != nil {
			fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
			os.Exit(1)
		}

		os.Exit(0)
	}

	opts := runFunctionOpts{
		verbose:   hasVerboseFlag,
		eventFunc: eventFunc,
	}

	if err = runFunction(ctx, *fn, opts); err != nil {
		// Already printed.
		os.Exit(1)
	}
}

// Returns a loader that creates a single generated `triggerName` event, or a
// random triggering event from the function definition if `triggerName` is
// empty.
//
// Will also attempt to read an event from stdin.
func generatedEventLoader(ctx context.Context, fn function.Function, triggerName string) func() ([]event.Event, error) {
	return func() ([]event.Event, error) {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return []event.Event{}, err
		}
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			// Read stdin
			scanner := bufio.NewScanner(os.Stdin)

			var input []byte
			for scanner.Scan() {
				line := scanner.Bytes()
				if len(line) == 0 {
					break
				}
				input = append(input, line...)
			}

			multipleEvents := []event.Event{}
			err := json.Unmarshal(input, &multipleEvents)
			if err != nil {
				// If we couldn't parse multiple events, try a single event instead.
				singleEvent := event.Event{}
				err := json.Unmarshal(input, &singleEvent)

				return []event.Event{singleEvent}, err
			}

			return multipleEvents, nil
		}

		// If we're generating an event and haven't been given a random seed,
		// generate one now.
		if runSeed <= 0 {
			rand.Seed(time.Now().UnixNano())
			runSeed = rand.Int63n(1_000_000)
		}

		fakedEvent, err := fakeEvent(ctx, fn, triggerName)
		if err != nil {
			return nil, err
		}

		return []event.Event{fakedEvent}, nil
	}
}

// Returns a loader that fetches a particular event specified by `eventId`.
func singleReplayEventLoader(ctx context.Context, eventId *ulid.ULID) func() ([]event.Event, error) {
	return func() ([]event.Event, error) {
		s := clistate.RequireState(ctx)

		ws, err := clistate.Workspace(ctx)
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

// Returns a loader that fetches multiple recent `triggerName` events, up to a
// maximum of `count`.
func multiReplayEventLoader(ctx context.Context, triggerName string, count int64) func() ([]event.Event, error) {
	return func() ([]event.Event, error) {
		if triggerName == "" {
			return nil, fmt.Errorf("triggerName is required")
		}

		if count <= 0 {
			return nil, fmt.Errorf("count must be >0")
		}

		s := clistate.RequireState(ctx)

		ws, err := clistate.Workspace(ctx)
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
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")

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

func snapshotEvents(ctx context.Context, eventFunc func() ([]event.Event, error)) error {
	events, err := eventFunc()
	if err != nil {
		return err
	}

	json, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(json))

	return nil
}
