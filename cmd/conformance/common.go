package conformance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/inngest/inngest/pkg/conformance/runner/serve"
	"github.com/urfave/cli/v3"
)

// conformanceRunFlags centralizes the shared flag surface for `run` and
// `doctor`, keeping configuration and runtime resolution aligned.
func conformanceRunFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "config",
			Usage: "Path to a conformance config file (YAML or JSON).",
		},
		&cli.StringFlag{
			Name:  "transport",
			Usage: "Transport to execute. One of: serve, connect.",
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
			Name:  "timeout",
			Usage: "Overall case timeout, for example 60s or 2m.",
		},
		&cli.StringFlag{
			Name:  "sdk-url",
			Usage: "Serve endpoint exposed by the SDK fixture, for example http://127.0.0.1:3000/api/inngest.",
		},
		&cli.StringFlag{
			Name:  "introspect-path",
			Usage: "Override the introspection path relative to --sdk-url.",
		},
		&cli.StringFlag{
			Name:  "dev-url",
			Usage: "Base URL for the dev server when API and event endpoints share a host.",
		},
		&cli.StringFlag{
			Name:  "api-url",
			Usage: "Base URL for the dev server API used for function registration.",
		},
		&cli.StringFlag{
			Name:  "event-url",
			Usage: "Base URL for the event API used for publishing trigger events.",
		},
		&cli.StringFlag{
			Name:  "event-key",
			Usage: "Event key used to publish test events. Defaults to 'test'.",
		},
		&cli.StringFlag{
			Name:  "signing-key",
			Usage: "Signing key used to authorize function registration with the dev server.",
		},
		&cli.StringFlag{
			Name:  "report-format",
			Value: string(conf.ReportFormatPretty),
			Usage: "Terminal output format. One of: pretty, json, junit, markdown.",
		},
		&cli.StringFlag{
			Name:  "report-out",
			Usage: "Optional JSON artifact path written in addition to terminal output.",
		},
	}
}

// loadExecutionConfig merges file-based config with explicit CLI overrides.
func loadExecutionConfig(cmd *cli.Command) (conf.Config, error) {
	cfg, err := conf.LoadConfig(cmd.String("config"))
	if err != nil {
		return conf.Config{}, err
	}

	if cmd.IsSet("transport") {
		cfg.Transport = conf.Transport(cmd.String("transport"))
	}
	cfg.Suites = mergeStringSlices(cfg.Suites, cmd.StringSlice("suite"))
	cfg.Cases = mergeStringSlices(cfg.Cases, cmd.StringSlice("case"))
	cfg.Features = mergeStringSlices(cfg.Features, cmd.StringSlice("feature"))

	if cmd.IsSet("timeout") {
		cfg.Timeout = cmd.String("timeout")
	}
	if cmd.IsSet("sdk-url") {
		cfg.SDK.URL = cmd.String("sdk-url")
	}
	if cmd.IsSet("introspect-path") {
		cfg.SDK.IntrospectPath = cmd.String("introspect-path")
	}
	if cmd.IsSet("dev-url") {
		cfg.Dev.URL = cmd.String("dev-url")
	}
	if cmd.IsSet("api-url") {
		cfg.Dev.APIURL = cmd.String("api-url")
	}
	if cmd.IsSet("event-url") {
		cfg.Dev.EventURL = cmd.String("event-url")
	}
	if cmd.IsSet("event-key") {
		cfg.Dev.EventKey = cmd.String("event-key")
	}
	if cmd.IsSet("signing-key") {
		cfg.Dev.SigningKey = cmd.String("signing-key")
	}
	if cmd.IsSet("report-format") {
		cfg.Report.Format = conf.ReportFormat(cmd.String("report-format"))
	}
	if cmd.IsSet("report-out") {
		cfg.Report.Output = cmd.String("report-out")
	}

	if err := cfg.Validate(); err != nil {
		return conf.Config{}, err
	}

	return cfg, nil
}

// resolveExecutionPlan converts the merged config into a runtime plus a concrete
// case selection.
func resolveExecutionPlan(cfg conf.Config) (conf.RunPlan, conf.RuntimeConfig, error) {
	runtime, err := cfg.Runtime()
	if err != nil {
		return conf.RunPlan{}, conf.RuntimeConfig{}, err
	}

	selection := cfg.Selection()
	selection.Transport = runtime.Transport

	plan, err := selection.Resolve(conf.DefaultRegistry())
	if err != nil {
		return conf.RunPlan{}, conf.RuntimeConfig{}, err
	}

	return plan, runtime, nil
}

func newServeRunner() *serve.Runner {
	return serve.NewRunner(nil)
}

// renderReport prints a report using the Phase 2 terminal renderers.
func renderReport(report conf.Report, format conf.ReportFormat, forceJSON bool) error {
	if forceJSON || format == conf.ReportFormatJSON {
		byt, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(byt))
		return nil
	}

	switch format {
	case "", conf.ReportFormatPretty:
		return output.TextConformanceReport(report)
	case conf.ReportFormatJUnit, conf.ReportFormatMarkdown:
		return fmt.Errorf("report format %q is not implemented yet", format)
	default:
		return fmt.Errorf("unknown report format %q", format)
	}
}

func renderDoctor(checks []serve.Check, forceJSON bool) error {
	if forceJSON {
		byt, err := json.MarshalIndent(checks, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(byt))
		return nil
	}

	return output.TextConformanceDoctor(checks)
}

func runServe(ctx context.Context, plan conf.RunPlan, runtime conf.RuntimeConfig) (conf.Report, error) {
	return newServeRunner().Run(ctx, plan, runtime)
}
