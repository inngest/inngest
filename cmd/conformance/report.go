package conformance

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func reportCommand() *cli.Command {
	return &cli.Command{
		Name:  "report",
		Usage: "Render an existing conformance report artifact.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			_ = cmd
			fmt.Println("Phase 1 stub: conformance report rendering is not implemented yet.")
			return nil
		},
	}
}
