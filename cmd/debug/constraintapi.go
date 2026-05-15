package debug

import (
	"github.com/inngest/inngest/cmd/debug/constraintapi"
	"github.com/urfave/cli/v3"
)

func constraintCommand() *cli.Command {
	return &cli.Command{
		Name:    "constraintapi",
		Aliases: []string{"c"},
		Usage:   "ConstraintAPI debugging commands",
		Commands: []*cli.Command{
			constraintapi.CheckCommand(),
		},
	}
}
