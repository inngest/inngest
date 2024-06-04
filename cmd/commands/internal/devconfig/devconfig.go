package devconfig

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func InitConfig(cmd *cobra.Command) {
	viper.BindPFlag("host", cmd.Flags().Lookup("host"))
	viper.BindPFlag("no-discovery", cmd.Flags().Lookup("no-discovery"))
	viper.BindPFlag("no-poll", cmd.Flags().Lookup("no-poll"))
	viper.BindPFlag("poll-interval", cmd.Flags().Lookup("poll-interval"))
	viper.BindPFlag("port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("retry-interval", cmd.Flags().Lookup("retry-interval"))
	viper.BindPFlag("tick", cmd.Flags().Lookup("tick"))
	viper.BindPFlag("urls", cmd.Flags().Lookup("sdk-url"))

	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
		}
		viper.AddConfigPath(pwd)
		viper.SetConfigName("inngest.yml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return
		}
		fmt.Println(err)
	}

	viper.Set("urls", viper.GetStringSlice("urls"))
}
