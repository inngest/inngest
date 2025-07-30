package localconfig

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v3"
)

func InitDevConfig(ctx context.Context, cmd *cli.Command) error {
	if err := mapDevFlags(cmd); err != nil {
		return err
	}

	loadConfigFile(ctx, cmd)

	return nil
}

func InitStartConfig(ctx context.Context, cmd *cli.Command) error {
	if err := mapStartFlags(cmd); err != nil {
		return err
	}

	loadConfigFile(ctx, cmd)

	return nil
}

func loadConfigFile(ctx context.Context, cmd *cli.Command) {
	l := logger.StdlibLogger(ctx)

	// Automatially bind environment variables
	viper.SetEnvPrefix("INNGEST")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	configPath := cmd.String("config")
	if configPath != "" {
		// User specified the config file so we'll use that
		viper.SetConfigFile(configPath)
	} else {
		// Don't need to specify the extension since Viper will try to load
		// various extensions (inngest.json, inngest.yaml, etc.)
		viper.SetConfigName("inngest")

		if cwd, err := os.Getwd(); err != nil {
			l.Warn("error getting current directory", "error", err)
		} else {
			// Walk up the directory tree looking for a config file
			dir := cwd
			for {
				viper.AddConfigPath(dir)

				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}

				dir = parent
			}
		}

		if homeDir, err := os.UserHomeDir(); err != nil {
			l.Warn("error getting home directory", "error", err)
		} else {
			// Fallback to ~/.config/inngest
			viper.AddConfigPath(filepath.Join(homeDir, ".config", "inngest"))
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if configPath != "" {
			// User explicitly specified a config file but we couldn't read it
			log.Fatalf("Error reading config file: %v", err)
		}
	} else {
		l.Info("using config", "file", viper.ConfigFileUsed())
	}
}

// mapDevFlags binds the command line flags to the viper configuration
func mapDevFlags(cmd *cli.Command) error {
	// With urfave/cli, we no longer need to bind flags to viper
	// since we can access them directly from the context
	// Keep this function for compatibility but make it a no-op
	return nil
}

// mapStartFlags binds the command line flags to the viper configuration
func mapStartFlags(cmd *cli.Command) error {
	// With urfave/cli, we no longer need to bind flags to viper
	// since we can access them directly from the context
	// Keep this function for compatibility but make it a no-op
	return nil
}
