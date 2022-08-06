package commands

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/cli/initialize"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Create a new serverless function",
		Example: "inngestctl init",
		Run:     runInit,
	}

	cmd.Flags().String("event", "", "An optional event name which triggers the function")
	cmd.Flags().String("cron", "", "An optional cron schedule to use as the function trigger")
	cmd.Flags().StringP("name", "n", "", "The function name")
	cmd.Flags().String("language", "", "The language to use within your project")
	cmd.Flags().String("url", "", "The URL to hit, if this function calls an external API")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) {
	if _, err := function.Load(cmd.Context(), "."); err == nil {
		// XXX: We can't both SilenceUsage and SilenceError, so we handle error checking inside
		// the init function here.
		fmt.Println("\n" + cli.RenderError("An inngest project already exists in this directory.  Remove the inngest file to continue.") + "\n")
		os.Exit(1)
		return
	}

	showWelcome := true
	if setting, ok := clistate.GetSetting(cmd.Context(), clistate.SettingRanInit).(bool); ok {
		// only show the welcome if we haven't ran init
		showWelcome = !setting
	}

	// Create a new TUI which walks through questions for creating a function.  Once
	// the walkthrough is complete, this blocks and returns.
	state, err := initialize.NewInitModel(initialize.InitOpts{
		ShowWelcome: showWelcome,
		Event:       cmd.Flag("event").Value.String(),
		Cron:        cmd.Flag("cron").Value.String(),
		Name:        cmd.Flag("name").Value.String(),
		Language:    cmd.Flag("language").Value.String(),
		URL:         cmd.Flag("url").Value.String(),
	})
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
	fn, err := state.Function(cmd.Context())
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("There was an error creating the function: %s", err)) + "\n")
		return
	}

	// Save a setting which indicates that we've ran init successfully.
	// This is used to prevent us from showing the welcome message on subsequent runs.
	_ = clistate.SaveSetting(cmd.Context(), clistate.SettingRanInit, true)

	// If we have a template, render that.
	tpl := state.Template()
	if tpl == nil {
		// Use a blank template with no fs.FS to render only the JSON into the directory.
		tpl = &scaffold.Template{}
	}

	var step function.Step
	for _, v := range fn.Steps {
		step = v
	}

	if err := tpl.Render(*fn, step); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("There was an error creating the function: %s", err)) + "\n")
		return
	}

	fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render(fmt.Sprintf("🎉 Done!  Your function has been created in ./%s", fn.Slug())))

	if tpl.PostSetup != "" {
		renderer, _ := glamour.NewTermRenderer(
			// detect background color and pick either the default dark or light theme
			glamour.WithAutoStyle(),
		)
		out, _ := renderer.Render(tpl.TemplatedPostSetup(*fn))
		fmt.Println(out)
	}

	tel.Send(cmd.Context(), state.TelEvent())

	fmt.Println(cli.TextStyle.Render("For more information, read our documentation at https://www.inngest.com/docs\n"))
}
