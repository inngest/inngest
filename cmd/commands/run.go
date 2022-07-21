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
	fetchRecentEvents int64
	verbose           bool
	fetchEventId      *ulid.ULID
}

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
	cmd.Flags().BoolP("replay", "r", false, "Enables replay mode to replay real recent events")
	cmd.Flags().Int64VarP(&replayCount, "count", "c", 10, "Number of events to replay in replay mode")
	cmd.Flags().StringP("event-id", "e", "", "Specifies a specific event to replay in replay mode")
	cmd.Flags().StringP("trigger", "t", "", "Specifies a the trigger you wish to retrieve events for in replay mode")

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

	eventName := cmd.Flag("event").Value.String()
	hasVerboseFlag := cmd.Flag("verbose").Value.String() == "true"
	isReplayMode := cmd.Flag("replay").Value.String() == "true"

	var fetchRecentEventCount int64 = 0

	if isReplayMode {
		fetchRecentEventCount = replayCount
	}

	opts := runFunctionOpts{
		fetchRecentEvents: fetchRecentEventCount,
		verbose:           hasVerboseFlag,
	}

	rawEventId := cmd.Flag("event-id").Value.String()

	if rawEventId != "" {
		eventId, err := ulid.ParseStrict(rawEventId)
		if err != nil {
			fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
			os.Exit(1)
		}

		opts.fetchEventId = &eventId
	}

	if err = runFunction(cmd.Context(), *fn, eventName, opts); err != nil {
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
func runFunction(ctx context.Context, fn function.Function, eventName string, opts runFunctionOpts) error {
	var evts []event.Event
	var err error

	if opts.fetchEventId != nil {
		evt, err := fetchEvent(ctx, *opts.fetchEventId)
		if err != nil {
			return err
		}
		evts = []event.Event{*evt}
	} else if opts.fetchRecentEvents > 0 {
		evts, err = fetchRecentEvents(ctx, eventName, int64(opts.fetchRecentEvents))
		if err != nil {
			return err
		}
	} else {
		evts, err = generateEvents(ctx, fn, eventName)
		if err != nil {
			return err
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

	if runSeed <= 0 {
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
func generateEvents(ctx context.Context, fn function.Function, eventName string) ([]event.Event, error) {
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

	fakedEvent, err := fakeEvent(ctx, fn, eventName)
	if err != nil {
		return nil, err
	}

	return []event.Event{fakedEvent}, nil
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

func fetchRecentEvents(ctx context.Context, eventName string, count int64) ([]event.Event, error) {
	s := state.RequireState(ctx)

	ws, err := state.Workspace(ctx)
	if err != nil {
		return nil, err
	}

	archivedEvents, err := s.Client.RecentEvents(ctx, ws.ID, eventName, count)
	if err != nil {
		fmt.Println("Oof error", err)
		return nil, err
	}

	events := []event.Event{}

	for _, archivedEvent := range archivedEvents {
		type evtData struct {
			Data map[string]interface{}
			Name string
			ts   int64
		}

		evt := &evtData{}

		if err := json.Unmarshal([]byte(archivedEvent.Event), &evt); err != nil {
			return nil, err
		}

		events = append(events, event.Event{
			ID:        archivedEvent.ID,
			Name:      archivedEvent.Name,
			Data:      evt.Data,
			Timestamp: evt.ts,
		})
	}

	return events, nil
}

func fetchEvent(ctx context.Context, eventId ulid.ULID) (*event.Event, error) {
	s := state.RequireState(ctx)

	ws, err := state.Workspace(ctx)
	if err != nil {
		return nil, err
	}

	archivedEvent, err := s.Client.RecentEvent(ctx, ws.ID, eventId)
	if err != nil {
		return nil, err
	}

	type evtData struct {
		Data map[string]interface{}
		Name string
		ts   int64
	}

	evt := &evtData{}

	if err := json.Unmarshal([]byte(archivedEvent.Event), &evt); err != nil {
		return nil, err
	}

	return &event.Event{
		ID:        archivedEvent.ID,
		Name:      archivedEvent.Name,
		Data:      evt.Data,
		Timestamp: evt.ts,
	}, nil
}
