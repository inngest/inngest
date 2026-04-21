package conformance

import (
	"context"
	"fmt"

	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/urfave/cli/v3"
)

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run a conformance suite against the selected transport.",
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

			var report conf.Report
			switch runtime.Transport {
			case conf.TransportServe:
				report, err = runServe(ctx, plan, runtime)
			case conf.TransportConnect:
				return cli.Exit("connect conformance is not implemented yet", 2)
			default:
				return cli.Exit(fmt.Sprintf("unknown transport %q", runtime.Transport), 2)
			}
			if err != nil {
				return err
			}

			if cfg.Report.Output != "" {
				if err := conf.WriteReport(cfg.Report.Output, report); err != nil {
					return err
				}
			}

			return renderReport(report, cfg.Report.Format, cmd.Bool("json"))
		},
	}
}

func mergeStringSlices(base []string, overrides []string) []string {
	if len(overrides) == 0 {
		return base
	}
	return overrides
}
