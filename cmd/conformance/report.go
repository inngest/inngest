package conformance

import (
	"context"

	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/urfave/cli/v3"
)

func reportCommand() *cli.Command {
	return &cli.Command{
		Name:  "report",
		Usage: "Render an existing conformance report artifact.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "path",
				Usage:    "Path to a previously written conformance report JSON file.",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "report-format",
				Value: string(conf.ReportFormatPretty),
				Usage: "Output format for terminal rendering. One of: pretty, json, junit, markdown.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			report, err := conf.LoadReport(cmd.String("path"))
			if err != nil {
				return err
			}
			return renderReport(report, conf.ReportFormat(cmd.String("report-format")), cmd.Bool("json"))
		},
	}
}
