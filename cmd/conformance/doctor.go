package conformance

import (
	"context"

	"github.com/inngest/inngest/pkg/cli/output"
	"github.com/urfave/cli/v3"
)

func doctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Check conformance prerequisites.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			_ = cmd
			return output.TextConformanceStub("doctor", "Phase 1 stub: conformance doctor is not implemented yet.")
		},
	}
}
