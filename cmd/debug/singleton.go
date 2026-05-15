package debug

import (
	"github.com/inngest/inngest/cmd/debug/singleton"
	"github.com/urfave/cli/v3"
)

func singletonCommand() *cli.Command {
	return &cli.Command{
		Name:    "singleton",
		Aliases: []string{"s"},
		Usage:   "Singleton lock debugging commands",
		Commands: []*cli.Command{
			singleton.InfoCommand(),
			singleton.DeleteCommand(),
		},
	}
}
