package debug

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:    "debug",
		Aliases: []string{"dbg"},
		Usage:   "Debug commands",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return fmt.Errorf("not implemented")
		},
	}
}
