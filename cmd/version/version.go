package version

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name: "version",
		Usage: fmt.Sprintf(
			"Shows the inngest CLI version (saving time, it's: %s)",
			version.Print(),
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(version.Print())
			return nil
		},
	}
}
