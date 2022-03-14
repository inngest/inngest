package build

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	builderDocker = "docker"
	builderPodman = "podman"
)

func Build(cmd *cobra.Command, args []string) {
	builder := strings.ToLower(viper.GetString("builder"))

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
	a := append([]string{"buildx", "build", "--load"}, args...)
	return a
}
