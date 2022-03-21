package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/docker"
	"github.com/spf13/cobra"
)

const (
	long = `
Build an action for testing or pushing to Inngest.

Supports passing any argument to the builder after a '--' separator 
from the commands args and flags.
`
	example = `
$ inngestctl build .
$ inngestctl build /path/to/Dockerfile
$ inngestctl build . -- --tag mycoolimage
`
)

func NewCmdBuild() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build PATH | URL",
		Short:   "Build an action",
		Long:    long,
		Example: example,
		Run:     build,
		Args:    cobra.MinimumNArgs(1),
		Hidden:  true,
	}
	return cmd
}

func build(cmd *cobra.Command, args []string) {
	ui, err := cli.NewBuilder(cmd.Context(), docker.BuildOpts{
		Args: args,
	})
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
}
