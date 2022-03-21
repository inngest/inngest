package commands

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/inngest/inngestctl/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Create a new serverless function",
		Example: "inngestctl init",
		Run:     runInit,
	}
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

	// If we have a template, render that.
	tpl := state.Template()
	if tpl == nil {
		// Use a blank template with no fs.FS to render only the JSON into the directory.
		tpl = &scaffold.Template{}
	}
	if err := tpl.Render(*fn); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("There was an error creating the function: %s", err)) + "\n")
		return
	}

	fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render(fmt.Sprintf("🎉 Done!  Your project has been created in ./%s", fn.Slug())))

	if tpl.PostSetup != "" {
		renderer, _ := glamour.NewTermRenderer(
			// detect background color and pick either the default dark or light theme
			glamour.WithAutoStyle(),
		)
		out, _ := renderer.Render(tpl.PostSetup)
		fmt.Println(out)
	}

	fmt.Println(cli.TextStyle.Render("For more information, read our documentation at https://www.inngest.com/docs\n"))
}
