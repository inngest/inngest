package conformance

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func doctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Check conformance prerequisites.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			_ = cmd
			fmt.Println("Phase 1 stub: conformance doctor is not implemented yet.")
			return nil
		},
	}
}
