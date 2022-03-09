package commands

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/spf13/cobra"
)

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Create a new serverless function",
		Example: "inngestctl init",
		Run:     runInit,
	}
	cmd.Flags().StringP("builder", "b", "docker", "Specify the builder to use. Options: docker or podman")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) {
	if _, err := function.Load("."); err == nil {
		// XXX: We can't both SilenceUsage and SilenceErroo we handle error checking inside
		// the init function here.
		fmt.Println("\n" + cli.RenderError("An inngest project already exists in this directory.  Remove the inngest file to continue.") + "\n")
		os.Exit(1)
		return
	}

	// Create a new TUI which walks through questions for creating a function.  Once
	// the walkthrough is complete, this blocks and returns.
	state, err := cli.NewInitModel()
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Error starting init command: %s", err)) + "\n")
		return
	}
	if err := tea.NewProgram(state).Start(); err != nil {
		log.Fatal(err)
	}

	if state.DidQuitEarly() {
		fmt.Println("\n" + cli.RenderWarning("Quitting without making your function.  Take care!") + "\n")
		return
	}

	// Get the function from the state.
	fn, err := state.Function()
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("There was an error creating the function: %s", err)) + "\n")
		return
	}

	// Once complete, state should contain everything we need to create our
	// function file.
	byt, err := function.MarshalJSON(*fn)
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Error creating JSON: %s", err)) + "\n")
		return
	}

	if err := os.WriteFile("./inngest.json", byt, 0600); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Error writing inngest.json: %s", err)) + "\n")
	}

	fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render("🎉 Done!  ./inngest.json has been created and you're ready to go."))
	fmt.Println(cli.TextStyle.Render("For more information, read our documentation at https://www.inngest.com/docs\n"))
}
