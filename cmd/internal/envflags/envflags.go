package envflags

import (
	"os"

	"github.com/urfave/cli/v3"
)

// GetEnvOrFlag returns the command line flag value, or falls back to the environment variable if the flag is empty
func GetEnvOrFlag(cmd *cli.Command, flagName, envName string) string {
	value := cmd.String(flagName)
	if value == "" {
		value = os.Getenv(envName)
	}
	return value
}

// GetEnvOrFlagWithDefault returns the environment variable first, then the command line flag, then the default
// This is useful for flags that have default values - environment variables take precedence over defaults
func GetEnvOrFlagWithDefault(cmd *cli.Command, flagName, envName, defaultValue string) string {
	// Check environment variable first
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	// Then check if flag was explicitly set (not just using default)
	if cmd.IsSet(flagName) {
		return cmd.String(flagName)
	}
	// Return default
	return defaultValue
}

// GetEnvOrStringSlice returns the command line flag value, or falls back to a single environment variable if the slice is empty
func GetEnvOrStringSlice(cmd *cli.Command, flagName, envName string) []string {
	values := cmd.StringSlice(flagName)
	if len(values) == 0 {
		if envValue := os.Getenv(envName); envValue != "" {
			return []string{envValue}
		}
	}
	return values
}
