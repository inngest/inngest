package version

import (
	"context"
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/update"
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
			// Notice on stderr so stdout (the version string) stays clean
			// for `inngest version | xargs ...` style usage.
			update.Notify(os.Stderr, version.Version)
			return nil
		},
	}
}
