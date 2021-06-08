package commands

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "inngest",
	Short: "Serverless event-driven infrastructure platform",
}

func init() {
	rootCmd.PersistentFlags().String("log.type", "", "Log type (one of json, tty). Defaults to 'json' without a TTY")
	rootCmd.PersistentFlags().StringP("log.level", "l", "debug", "Log level")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("inngest")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

func Execute() {
	rootCmd.Execute()
}
