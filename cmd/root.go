package main

import (
	"context"
	"fmt"
	"os"

	"github.com/inngest/inngest/cmd/devserver"
	"github.com/inngest/inngest/cmd/start"
	"github.com/inngest/inngest/cmd/version"
	"github.com/inngest/inngest/pkg/api/tel"
	inncli "github.com/inngest/inngest/pkg/cli"
	inngestversion "github.com/inngest/inngest/pkg/inngest/version"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v3"
)

const (
	ViperLogLevelKey = "log.level"
)

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
			// Bind global flags to viper
			if cmd.IsSet("log-level") {
				viper.Set(ViperLogLevelKey, cmd.String("log-level"))
			} else if cmd.Bool("verbose") {
				viper.Set(ViperLogLevelKey, "debug")
			} else {
				viper.Set(ViperLogLevelKey, "info")
			}

			// Also set the flag values in viper for other parts of the code
			viper.Set("json", cmd.Bool("json"))
			viper.Set("verbose", cmd.Bool("verbose"))
			viper.Set("log-level", cmd.String("log-level"))

			// Set LOG_HANDLER environment variable based on --json flag
			// This ensures the logger respects the JSON output setting
			if cmd.Bool("json") {
				os.Setenv("LOG_HANDLER", "json")
			}

			// Set LOG_LEVEL environment variable so the logger picks it up
			if cmd.IsSet("log-level") {
				os.Setenv("LOG_LEVEL", cmd.String("log-level"))
			} else if cmd.Bool("verbose") {
				os.Setenv("LOG_LEVEL", "debug")
			} else {
				os.Setenv("LOG_LEVEL", "info")
			}

			m := tel.NewMetadata(ctx)
			m.SetCliContext(cmd)
			tel.SendMetadata(ctx, m)
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
		},
	}

	// Set up flag binding with viper
	for _, flag := range app.Flags {
		if f, ok := flag.(*cli.BoolFlag); ok {
			viper.SetDefault(f.Name, false)
		} else if f, ok := flag.(*cli.StringFlag); ok {
			viper.SetDefault(f.Name, f.Value)
		}
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		// Always use JSON when not in a terminal
		viper.Set("json", true)
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
