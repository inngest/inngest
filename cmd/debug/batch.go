package debug

import (
	"github.com/inngest/inngest/cmd/debug/batch"
	"github.com/urfave/cli/v3"
)

func batchCommand() *cli.Command {
	return &cli.Command{
		Name:    "batch",
		Aliases: []string{"b"},
		Usage:   "Batch debugging commands",
		Commands: []*cli.Command{
			batch.InfoCommand(),
			batch.DeleteCommand(),
			batch.RunCommand(),
		},
	}
}
