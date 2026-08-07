package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/inngest/inngest/cmd/apiv2cli"
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
		Usage: "Output logs as JSON.  Defaults to true when stdout is not a TTY; set --json=false to force human-readable output.",
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

// resolveLogHandler determines the LOG_HANDLER value to apply from the --json
// flag, whether stdout is a TTY, and the current LOG_HANDLER env value. It
// returns "json" or "dev", or "" to leave the current LOG_HANDLER untouched.
//
// Priority:
//  1. --json=true: always "json".
//  2. --json=false: force the human-readable "dev" handler, but only when the
//     current handler would otherwise produce JSON. An explicit non-JSON
//     handler (e.g. LOG_HANDLER=text) is already what the user asked for, so it
//     is preserved. This is what lets --json=false override the non-TTY JSON
//     default (#4379).
//  3. --json not set, non-TTY: default to "json" (machine-readable logs for
//     pipes, Docker, turbo, etc.). Matches the previous default behavior, which
//     forced JSON in non-TTY contexts regardless of any prior LOG_HANDLER.
//  4. --json not set, TTY: leave unchanged so a configured LOG_HANDLER env var
//     (or the default dev handler) applies.
func resolveLogHandler(jsonSet, jsonValue, isTTY bool, current string) string {
	if jsonSet {
		if jsonValue {
			return "json"
		}
		if isJSONHandler(current) {
			return "dev"
		}
		// Already a human-readable handler (dev/text); honor the user's choice.
		return ""
	}
	if !isTTY {
		return "json"
	}
	return ""
}

// isJSONHandler reports whether a LOG_HANDLER value resolves to JSON output.
// An empty/unset value resolves to the human-readable dev handler, so it is not
// considered JSON here.
func isJSONHandler(handler string) bool {
	return strings.ToLower(strings.TrimSpace(handler)) == "json"
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
			// Resolve the log handler from the --json flag and TTY state. This
			// runs after flags are parsed, so an explicit --json=false can
			// override the non-TTY default of JSON output (see #4379).
			if handler := resolveLogHandler(cmd.IsSet("json"), cmd.Bool("json"), isatty.IsTerminal(os.Stdout.Fd()), os.Getenv("LOG_HANDLER")); handler != "" {
				os.Setenv("LOG_HANDLER", handler)
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

			if cmd.Args().Len() == 0 || cmd.Args().Get(0) != "api" {
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
			apiv2cli.Command(),
			devserver.Command(),
			version.Command(),
			start.Command(),
			alpha(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
