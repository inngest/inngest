package conformance

import (
	"context"

	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/urfave/cli/v3"
)

func doctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Check conformance prerequisites.",
		Flags: conformanceRunFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := loadExecutionConfig(cmd)
			if err != nil {
				return err
			}

			plan, runtime, err := resolveExecutionPlan(cfg)
			if err != nil {
				return err
			}

			switch runtime.Transport {
			case conf.TransportServe:
				checks, err := newServeRunner().Doctor(ctx, plan, runtime)
				if err != nil {
					return err
				}
				return renderDoctor(checks, cmd.Bool("json"))
			case conf.TransportConnect:
				return cli.Exit("connect conformance doctor is not implemented yet", 2)
			default:
				return cli.Exit("unknown conformance transport", 2)
			}
		},
	}
}
