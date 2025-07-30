package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/api/tel"
	inncli "github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/inngest/version"
	isatty "github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	"github.com/spf13/viper"
)

const (
	ViperLogLevelKey = "log.level"
)

// getGlobalFlags returns the global flags that should be available on all commands
func getGlobalFlags() []cli.Flag {
	return []cli.Flag{
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
}

// mergeFlags combines command-specific flags with global flags
func mergeFlags(commandFlags []cli.Flag) []cli.Flag {
	return append(commandFlags, getGlobalFlags()...)
}

func Execute() {
	app := &cli.App{
		Name: "inngest",
		Usage: inncli.TextStyle.Render(fmt.Sprintf(
			"%s %s\n\n%s",
			"Inngest CLI",
			fmt.Sprintf("v%s", version.Print()),
			"The durable execution engine with built-in flow control.",
		)),
		Version: version.Print(),
		UseShortOptionHandling: true,
		Before: func(c *cli.Context) error {
			// Bind global flags to viper
			if c.IsSet("log-level") {
				viper.Set(ViperLogLevelKey, c.String("log-level"))
			} else if c.Bool("verbose") {
				viper.Set(ViperLogLevelKey, "debug")
			} else {
				viper.Set(ViperLogLevelKey, "info")
			}
			
			// Also set the flag values in viper for other parts of the code
			viper.Set("json", c.Bool("json"))
			viper.Set("verbose", c.Bool("verbose"))
			viper.Set("log-level", c.String("log-level"))

			m := tel.NewMetadata(c.Context)
			m.SetCliContext(c)
			tel.SendMetadata(c.Context, m)
			return nil
		},
		After: func(c *cli.Context) error {
			// Wait for any events to have been sent.
			tel.Wait()
			return nil
		},

		// Add a note to the bottom of the help message
		CustomAppHelpTemplate: cli.AppHelpTemplate + fmt.Sprintf(
			"\n%s\n%s\n",
			"Request features, get help, and chat with us: ",
			"https://www.inngest.com/discord",
		),

		Flags: getGlobalFlags(),

	}

	// Add commands
	app.Commands = []*cli.Command{
		NewCmdDev(app),
		NewCmdVersion(),
		NewCmdStart(app),
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

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
