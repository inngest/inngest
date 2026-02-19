package debug

import (
	"github.com/inngest/inngest/cmd/debug/debounce"
	"github.com/urfave/cli/v3"
)

func debounceCommand() *cli.Command {
	return &cli.Command{
		Name:    "debounce",
		Aliases: []string{"db"},
		Usage:   "Debounce debugging commands",
		Commands: []*cli.Command{
			debounce.InfoCommand(),
			debounce.DeleteCommand(),
			debounce.DeleteByIDCommand(),
			debounce.RunCommand(),
		},
	}
}
