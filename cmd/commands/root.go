package commands

import (
	"fmt"
	"os"
	"strings"

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

	rootCmd.PersistentFlags().String("log.type", "", "Log type (one of json, tty). Defaults to 'json' without a TTY")
	rootCmd.PersistentFlags().StringP("log.level", "l", "debug", "Log level")
	rootCmd.PersistentFlags().StringP("builder", "b", "docker", "Specify the builder to use. Options: docker or podman")

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
	rootCmd.AddCommand(NewCmdInit())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
