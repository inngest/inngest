package debug

import (
	"github.com/inngest/inngest/cmd/debug/pause"
	"github.com/urfave/cli/v3"
)

func pauseCommand() *cli.Command {
	return &cli.Command{
		Name:    "pause",
		Aliases: []string{"p"},
		Usage:   "Pause debugging commands",
		Commands: []*cli.Command{
			pause.PauseCommand(),
			pause.IndexCommand(),
		},
	}
}
