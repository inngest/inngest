package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a serverless function locally",
		Example: "inngestctl run",
		Run:     doRun,
	}
	return cmd
}

func doRun(cmd *cobra.Command, args []string) {
	fn, err := function.Load(".")
	if err != nil {
		fmt.Println("\n" + cli.RenderError("No inngest.json or inngest.cue file found in your current directory") + "\n")
		os.Exit(1)
		return
	}

	err = runFunction(cmd.Context(), *fn)
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
		return
	}
}

// runFunction builds the function's images and runs the function.
func runFunction(ctx context.Context, fn function.Function) error {
	evt, err := event()
	if err != nil {
		return err
	}

	actions, err := fn.Actions()
	if err != nil {
		return err
	}
	if len(actions) != 1 {
		return fmt.Errorf("running step-functions locally is not yet supported")
	}

	// Build the image.
	ui, err := cli.NewRunUI(ctx, actions[0], evt)
	if err != nil {
		return err
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		return err
	}
	return nil
}

// event retrieves the event for use within testing the function.  It first checks stdin
// to see if we're passed an event, or resorts to generating a fake event based off of
// the function's event type.
func event() (map[string]interface{}, error) {
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

	//. XXX: Generate a new event.
	return nil, nil
}
