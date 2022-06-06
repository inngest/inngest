package commands

import (
	"fmt"
	"os"

	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	longDescription = `
    ____                            __
   /  _/___  ____  ____ ____  _____/ /_
   / // __ \/ __ \/ __ '/ _ \/ ___/ __/
 _/ // / / / / / / /_/ /  __(__  ) /_
/___/_/ /_/_/ /_/\__, /\___/____/\__/
                /____/
`
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "inngest",
		Short: "A serverless event-driven infrastructure platform",
		Long:  longDescription,
	}

	rootCmd.PersistentFlags().Bool("prod", false, "Use the production environment for the current command.")
	rootCmd.PersistentFlags().Bool("json", false, "Output logs as JSON.  Set to true if stdout is not a TTY.")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		// Alwyas use JSON when not in a terminal
		viper.Set("json", true)
	}

	// Register Top Level Commands
	rootCmd.AddCommand(NewCmdBuild())
	rootCmd.AddCommand(NewCmdLogin())
	rootCmd.AddCommand(NewCmdWorkflows())
	rootCmd.AddCommand(NewCmdWorkspaces())
	rootCmd.AddCommand(NewCmdInit())
	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdDeploy())
	rootCmd.AddCommand(NewCmdActions())
	rootCmd.AddCommand(NewCmdDev())
	rootCmd.AddCommand(NewCmdVersion())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
