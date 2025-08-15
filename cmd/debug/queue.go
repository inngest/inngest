package debug

import (
	"context"
	"fmt"

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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return fmt.Errorf("queue commands not yet implemented - use subcommands")
		},
	}
}