package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return command().Run(context.Background(), append([]string{"release-notes"}, args...))
}

func command() *cli.Command {
	return &cli.Command{
		Name:  "release-notes",
		Usage: "Collect and render release notes.",
		Commands: []*cli.Command{
			collectCommand(),
			buildCommand(),
			prBodyCommand(),
			prereleaseCommandCommand(),
			prereleaseVersionCommand(),
		},
		CommandNotFound: func(ctx context.Context, cmd *cli.Command, name string) {
			_, _ = fmt.Fprintf(cmd.ErrWriter, "unknown command %q\n", name)
		},
	}
}
