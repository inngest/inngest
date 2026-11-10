package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/inngest/inngest/cmd/apiv2cli"
	authcmd "github.com/inngest/inngest/cmd/auth"
	computecmd "github.com/inngest/inngest/cmd/compute"
	"github.com/inngest/inngest/cmd/devserver"
	"github.com/inngest/inngest/cmd/duckdb"
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
		sendHelpTelemetry(data)
	}
}

// sendHelpEvent is swapped out in tests.
var sendHelpEvent = func(ctx context.Context, command string, eventContext map[string]any) {
	tel.SendCommandExecutedEvent(ctx, command, eventContext)
}

// sendHelpTelemetry reports `inngest dev` help views. Help short-circuits
// before any Before/Action runs for `dev --help`, `dev -h`, and `dev help`, so
// this is the only path that sees all of them. Events are flushed by the
// tel.Wait() after app.Run in execute().
func sendHelpTelemetry(data any) {
	cmd, ok := data.(*cli.Command)
	if !ok || cmd.FullName() != "inngest dev" {
		return
	}
	sendHelpEvent(context.Background(), "inngest dev", map[string]any{
		"flags": map[string]bool{"help": true},
	})
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

			// `api` and `dev` send their own events w/ flag context (`dev`
			// help views are sent from the HelpPrinter wrapper).
			if cmd.Args().Len() == 0 || (cmd.Args().Get(0) != "api" && cmd.Args().Get(0) != "dev") {
				tel.SendCmdExecutedEvent(ctx, cmd)
			}

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
			authcmd.LoginCommand(),
			authcmd.LogoutCommand(),
			authcmd.AuthCommand(),
			computecmd.Command(),
			computecmd.DeployCommand(),
			apiv2cli.Command(),
			devserver.Command(),
			duckdb.Command(),
			version.Command(),
			start.Command(),
			alpha(),
		},
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		// Always use JSON when not in a terminal
		os.Setenv("LOG_HANDLER", "json")
	}

	err := app.Run(context.Background(), os.Args)
	// Flush telemetry sent from paths that skip the root After hook (IE help).
	tel.Wait()
	if err != nil {
		var reported *authcmd.ReportedError
		if errors.As(err, &reported) {
			os.Exit(1)
		}
		fmt.Println(err)
		os.Exit(1)
	}
}
