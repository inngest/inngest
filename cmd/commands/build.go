package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const (
	builderDocker = "docker"
	builderPodman = "podman"
	long          = `
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
		Run:     build,
		Args:    cobra.MinimumNArgs(1),
	}
	cmd.Flags().StringP("builder", "b", "docker", "Specify the builder to use. Options: docker or podman")
	return cmd
}

func build(cmd *cobra.Command, args []string) {
	builder, err := cmd.Flags().GetString("builder")
	if err != nil {
		fmt.Println(err)
		return
	}

	builder = strings.ToLower(builder)
	if builder != builderDocker && builder != builderPodman {
		fmt.Printf("Invalid builder specified:%v\nValid values are [docker] or [podman]\n\n", builder)
		fmt.Println(cmd.Help())
		return
	}

	path, err := exec.LookPath(builder)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Printf("The builder %v was not found in your PATH, please add it\n", builder)
			return
		}
		fmt.Println(err)
		return
	}

	builderArgs := createBuildCommand(args)

	dockerCmd := exec.Command(path, builderArgs...)
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdout = os.Stdout
	err = dockerCmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

func createBuildCommand(args []string) []string {
	var a []string
	a = append([]string{"buildx", "build", "--load"}, args...)
	return a
}
