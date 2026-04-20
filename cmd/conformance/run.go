package conformance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/urfave/cli/v3"
)

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Resolve and validate a conformance run plan.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to a conformance config file (YAML or JSON).",
			},
			&cli.StringFlag{
				Name:  "transport",
				Usage: "Filter cases by transport. One of: serve, connect.",
			},
			&cli.StringSliceFlag{
				Name:  "suite",
				Usage: "Run only the named suite(s).",
			},
			&cli.StringSliceFlag{
				Name:  "case",
				Usage: "Run only the named case(s).",
			},
			&cli.StringSliceFlag{
				Name:  "feature",
				Usage: "Run cases covering the named feature(s).",
			},
			&cli.StringFlag{
				Name:  "report-format",
				Value: string(conf.ReportFormatPretty),
				Usage: "Planned report format. One of: pretty, json, junit, markdown.",
			},
			&cli.StringFlag{
				Name:  "report-out",
				Usage: "Path for writing a future conformance report artifact.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx

			cfg, err := conf.LoadConfig(cmd.String("config"))
			if err != nil {
				return err
			}

			if cmd.IsSet("transport") {
				cfg.Transport = conf.Transport(cmd.String("transport"))
			}
			cfg.Suites = mergeStringSlices(cfg.Suites, cmd.StringSlice("suite"))
			cfg.Cases = mergeStringSlices(cfg.Cases, cmd.StringSlice("case"))
			cfg.Features = mergeStringSlices(cfg.Features, cmd.StringSlice("feature"))
			if cmd.IsSet("report-format") {
				cfg.Report.Format = conf.ReportFormat(cmd.String("report-format"))
			}
			if cmd.IsSet("report-out") {
				cfg.Report.Output = cmd.String("report-out")
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			registry := conf.DefaultRegistry()
			plan, err := cfg.Selection().Resolve(registry)
			if err != nil {
				return err
			}

			if cmd.Bool("json") || cfg.Report.Format == conf.ReportFormatJSON {
				byt, err := json.MarshalIndent(struct {
					SchemaVersion string               `json:"schema_version"`
					Config        conf.Config          `json:"config"`
					Plan          conf.RunPlan         `json:"plan"`
					Compatibility []conf.Compatibility `json:"compatibility_classes"`
					Note          string               `json:"note"`
				}{
					SchemaVersion: conf.ReportSchemaVersion,
					Config:        cfg,
					Plan:          plan,
					Compatibility: []conf.Compatibility{
						conf.CompatibilityFull,
						conf.CompatibilityPartial,
						conf.CompatibilityIncompatible,
						conf.CompatibilityUnknown,
					},
					Note: "Phase 1 validates configuration and selection only. Execution is not implemented yet.",
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(byt))
				return nil
			}

			return output.TextConformanceRunPlan(plan)
		},
	}
}

func mergeStringSlices(base []string, overrides []string) []string {
	if len(overrides) == 0 {
		return base
	}
	return overrides
}
