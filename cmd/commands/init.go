package commands

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	cmd.Flags().StringP("template", "t", "", "The template to use for the function")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	if _, err := function.Load(ctx, "."); err == nil {
		// XXX: We can't both SilenceUsage and SilenceError, so we handle error checking inside
		// the init function here.
		fmt.Println("\n" + cli.RenderError("An inngest project already exists in this directory.  Remove the inngest file to continue.") + "\n")
		os.Exit(1)
		return
	}

	// If we've been given a template, skip questions and get straight to trying
	// to initialize the project.
	template := cmd.Flag("template").Value.String()
	var err error

	if template != "" {
		err = cloneTemplate(ctx, template)
	} else {
		err = createNewFunction(ctx, cmd)
	}

	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("%s", err)) + "\n")
		return
	}
}

func cloneTemplate(ctx context.Context, template string) error {
	// template = [repo]#[path]
	// ask for =  [fn-name]
	//
	// git clone https://[repo].git --depth 1 --no-checkout [fn-name]
	// cd [fn-name]
	// git sparse-checkout set [path] --cone
	// cp -r [path]/* .
	// rm -r examples/

	fnName := "foo"
	repo, examplePath, _ := strings.Cut(template, "#")
	tmpPath, err := os.MkdirTemp("", "inngest-template-*")
	if err != nil {
		return err
	}

	cloneCmd := exec.Command("git", "clone", "https://"+repo+".git", "--depth", "1", tmpPath)
	err = cloneCmd.Run()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(cwd, fnName)

	onlyOwnerWrite := 0755
	if err = os.MkdirAll(targetDir, fs.FileMode(onlyOwnerWrite)); err != nil {
		return err
	}

	err = os.Rename(filepath.Join(tmpPath, examplePath), targetDir)
	if err != nil {
		return fmt.Errorf("Error moving template from temp directory: %s", err)
	}

	err = os.RemoveAll(tmpPath)
	if err != nil {
		fmt.Println("\n" + cli.RenderWarning(fmt.Sprintf("Failed to remove temporary dir after copy: %s", err)) + "\n")
	}

	return nil
}

func createNewFunction(ctx context.Context, cmd *cobra.Command) error {
	showWelcome := true
	if setting, ok := clistate.GetSetting(ctx, clistate.SettingRanInit).(bool); ok {
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
		return fmt.Errorf("Error starting init command: %s", err)
	}
	if err := tea.NewProgram(state).Start(); err != nil {
		log.Fatal(err)
	}

	if state.DidQuitEarly() {
		return fmt.Errorf("Quitting without making your function. Take care!")
	}

	// Get the function from the state.
	fn, err := state.Function(ctx)
	if err != nil {
		return fmt.Errorf("There was an error creating the function: %s", err)
	}

	// Save a setting which indicates that we've ran init successfully.
	// This is used to prevent us from showing the welcome message on subsequent runs.
	_ = clistate.SaveSetting(ctx, clistate.SettingRanInit, true)

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
		return fmt.Errorf("There was an error creating the function: %s", err)
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

	return nil
}
