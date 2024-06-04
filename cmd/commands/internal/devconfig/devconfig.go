package devconfig

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func InitConfig(cmd *cobra.Command) error {
	var err error
	err = errors.Join(err, viper.BindPFlag("host", cmd.Flags().Lookup("host")))
	err = errors.Join(err, viper.BindPFlag("no-discovery", cmd.Flags().Lookup("no-discovery")))
	err = errors.Join(err, viper.BindPFlag("no-poll", cmd.Flags().Lookup("no-poll")))
	err = errors.Join(err, viper.BindPFlag("poll-interval", cmd.Flags().Lookup("poll-interval")))
	err = errors.Join(err, viper.BindPFlag("port", cmd.Flags().Lookup("port")))
	err = errors.Join(err, viper.BindPFlag("retry-interval", cmd.Flags().Lookup("retry-interval")))
	err = errors.Join(err, viper.BindPFlag("tick", cmd.Flags().Lookup("tick")))
	err = errors.Join(err, viper.BindPFlag("urls", cmd.Flags().Lookup("sdk-url")))
	if err != nil {
		return err
	}

	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
		} else {
			viper.AddConfigPath(pwd)
			viper.SetConfigName("inngest.yml")
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					return nil
				}
				fmt.Println(err)
			}
			viper.Set("urls", viper.GetStringSlice("urls"))
		}
	}

	viper.AutomaticEnv()

	return nil
}
