package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/inngest/version"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ViperLogLevelKey = "log.level"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use: "inngest",
		Short: cli.TextStyle.Render(fmt.Sprintf(
			"%s %s\n\n%s",
			"Inngest CLI",
			fmt.Sprintf("v%s", version.Print()),
			"The durable execution engine with built-in flow control.",
		)),
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if viper.IsSet("log-level") {
				viper.Set(ViperLogLevelKey, viper.GetString("log-level"))
			} else if viper.GetBool("verbose") {
				viper.Set(ViperLogLevelKey, "debug")
			} else {
				viper.Set(ViperLogLevelKey, "info")
			}

			m := tel.NewMetadata(cmd.Context())
			m.SetCobraCmd(cmd)
			tel.SendMetadata(cmd.Context(), m)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Wait for any events to have been sent.
			tel.Wait()
		},
	}

	// Add a note to the bottom of the help message
	tmpl := rootCmd.HelpTemplate() + fmt.Sprintf(
		"\n%s\n%s\n",
		"Request features, get help, and chat with us: ",
		"https://www.inngest.com/discord",
	)
	rootCmd.SetHelpTemplate(tmpl)

	rootCmd.PersistentFlags().Bool("json", false, "Output logs as JSON.  Set to true if stdout is not a TTY.")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging.")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Set the log level.  One of: trace, debug, info, warn, error.")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		// Alwyas use JSON when not in a terminal
		viper.Set("json", true)
	}

	// Register Top Level Commands
	rootCmd.AddCommand(NewCmdDev(rootCmd))
	rootCmd.AddCommand(NewCmdVersion())
	rootCmd.AddCommand(NewCmdStart(rootCmd))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
