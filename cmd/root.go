package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/inngest/inngest/cmd/devserver"
	"github.com/inngest/inngest/cmd/start"
	"github.com/inngest/inngest/cmd/version"
	"github.com/inngest/inngest/pkg/api/tel"
	inncli "github.com/inngest/inngest/pkg/cli"
	inngestversion "github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/update"
	isatty "github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
)

func init() {
	// Wrap urfave/cli's default help printer so help output is followed by an
	// update notice (when applicable). Covers `inngest help`, bare `inngest`,
	// and `inngest <subcmd> --help`.
	defaultHelp := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		defaultHelp(w, templ, data)
		update.Notify(os.Stderr, inngestversion.Version)
	}
}

// globalFlags are the flags that should be available on all commands
var globalFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "json",
		Usage: "Output logs as JSON.  Set to true if stdout is not a TTY.",
	},
	&cli.BoolFlag{
		Name:  "verbose",
		Usage: "Enable verbose logging.",
	},
	&cli.StringFlag{
		Name:    "log-level",
		Aliases: []string{"l"},
		Value:   "info",
		Usage:   "Set the log level.  One of: trace, debug, info, warn, error.",
	},
}

func execute() {
	app := &cli.Command{
		Name: "inngest",
		Usage: inncli.TextStyle.Render(fmt.Sprintf(
			"%s %s\n\n%s",
			"Inngest CLI",
			fmt.Sprintf("v%s", inngestversion.Print()),
			"The durable execution engine with built-in flow control.",
		)),
		Version: inngestversion.Print(),
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Set LOG_HANDLER environment variable based on --json flag
			// This ensures the logger respects the JSON output setting
			if cmd.Bool("json") {
				os.Setenv("LOG_HANDLER", "json")
			}

			if os.Getenv("LOG_LEVEL") == "" {
				// Set LOG_LEVEL environment variable so the logger picks it up
				if cmd.IsSet("log-level") {
					os.Setenv("LOG_LEVEL", cmd.String("log-level"))
				} else if cmd.Bool("verbose") {
					os.Setenv("LOG_LEVEL", "debug")
				} else {
					os.Setenv("LOG_LEVEL", "info")
				}
			}

			m := tel.NewMetadata(ctx)
			m.SetCliContext(cmd)
			tel.SendMetadata(ctx, m)

			// Best-effort background refresh of the cached "latest version"
			// record. Dedup'd by the cache TTL, so cheap on every invocation.
			// Check honors the same opt-out gates as Notify.
			go update.Check(context.Background(), inngestversion.Version)

			return ctx, nil
		},
		After: func(ctx context.Context, cmd *cli.Command) error {
			// Wait for any events to have been sent.
			tel.Wait()
			return nil
		},

		Flags: globalFlags,
		Commands: []*cli.Command{
			devserver.Command(),
			version.Command(),
			start.Command(),
			alpha(),
		},
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		// Always use JSON when not in a terminal
		os.Setenv("LOG_HANDLER", "json")
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
