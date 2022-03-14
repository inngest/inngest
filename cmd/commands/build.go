package commands

import (
	"github.com/inngest/inngestctl/pkg/build"
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
$ inngestctl build . --builder docker -- --tag mycoolimage
`
)

func NewCmdBuild() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build PATH | URL",
		Short:   "Build an action",
		Long:    long,
		Example: example,
		Run:     build.Build,
		Args:    cobra.MinimumNArgs(1),
		Hidden:  true,
	}
	return cmd
}
