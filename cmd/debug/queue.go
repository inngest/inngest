package debug

import (
	"github.com/inngest/inngest/cmd/debug/queue"
	"github.com/urfave/cli/v3"
)

func queueCommand() *cli.Command {
	return &cli.Command{
		Name:    "queue",
		Aliases: []string{"q"},
		Usage:   "Queue debugging commands",
		Commands: []*cli.Command{
			queue.PartitionCommand(),
			queue.ItemCommand(),
			queue.BacklogCommand(),
		},
	}
}
