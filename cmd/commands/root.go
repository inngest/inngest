package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "inngest",
		Short: "A serverless event-driven infrastructure platform",
	}

	rootCmd.PersistentFlags().String("log.type", "", "Log type (one of json, tty). Defaults to 'json' without a TTY")
	rootCmd.PersistentFlags().StringP("log.level", "l", "debug", "Log level")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("inngest")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Register Top Level Commands
	rootCmd.AddCommand(NewCmdActions())
	rootCmd.AddCommand(NewCmdBuild())
	rootCmd.AddCommand(NewCmdLogin())
	rootCmd.AddCommand(NewCmdWorkflows())
	rootCmd.AddCommand(NewCmdWorkspaces())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
