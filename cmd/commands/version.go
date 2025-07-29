package commands

import (
	"fmt"

	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/urfave/cli/v2"
)

func NewCmdVersion() *cli.Command {
	return &cli.Command{
		Name: "version",
		Usage: fmt.Sprintf(
			"Shows the inngest CLI version (saving time, it's: %s)",
			version.Print(),
		),
		Action: func(c *cli.Context) error {
			fmt.Println(version.Print())
			return nil
		},
	}
}
