package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"
)

// Config represents the complete configuration structure for Inngest CLI
type Config struct {
	// Dev command configuration
	SdkURL      []string `koanf:"sdk-url"`
	NoDiscovery *bool    `koanf:"no-discovery"`
	NoPoll      *bool    `koanf:"no-poll"`
	Host        string   `koanf:"host"`
	Port        string   `koanf:"port"`

	// Advanced dev command configuration
	PollInterval       int   `koanf:"poll-interval"`
	RetryInterval      int   `koanf:"retry-interval"`
	QueueWorkers       int   `koanf:"queue-workers"`
	Tick               int   `koanf:"tick"`
	ConnectGatewayPort int   `koanf:"connect-gateway-port"`
	InMemory           *bool `koanf:"in-memory"`

	// Start command configuration
	SigningKey string   `koanf:"signing-key"`
	EventKey   []string `koanf:"event-key"`

	// Database configuration
	RedisURI                    string `koanf:"redis-uri"`
	PostgresURI                 string `koanf:"postgres-uri"`
	PostgresMaxIdleConns        int    `koanf:"postgres-max-idle-conns"`
	PostgresMaxOpenConns        int    `koanf:"postgres-max-open-conns"`
	PostgresConnMaxIdleTime     int    `koanf:"postgres-conn-max-idle-time"`
	PostgresConnMaxLifetime     int    `koanf:"postgres-conn-max-lifetime"`
	SqliteDir                   string `koanf:"sqlite-dir"`
}

// Global variables to store koanf instance and loaded configuration
var (
	k            = koanf.New(".")
	loadedConfig *Config
)

// InitDevConfig initializes configuration for the dev command
func InitDevConfig(ctx context.Context, cmd *cli.Command) error {
	return loadConfigFile(ctx, cmd)
}

// InitStartConfig initializes configuration for the start command
func InitStartConfig(ctx context.Context, cmd *cli.Command) error {
	return loadConfigFile(ctx, cmd)
}

// GetConfig returns the loaded configuration struct
func GetConfig() *Config {
	if loadedConfig == nil {
		return &Config{} // Return empty config if none loaded
	}
	return loadedConfig
}

// loadConfigFile loads configuration from multiple sources in priority order:
// 1. Config files (lowest priority)
// 2. Environment variables with INNGEST_ prefix
// 3. CLI flags (handled separately, highest priority)
func loadConfigFile(ctx context.Context, cmd *cli.Command) error {
	l := logger.StdlibLogger(ctx)

	// Step 1: Load config file (lowest priority)
	configLoaded := false
	configPath := cmd.String("config")
	if configPath == "" {
		configPath = os.Getenv("INNGEST_CONFIG")
	}
	if configPath != "" {
		if err := loadConfigFromPath(configPath); err != nil {
			return fmt.Errorf("error reading config file %s: %w", configPath, err)
		}
		l.Info("using config", "file", configPath)
	} else {
		// Search for config files in standard locations
		searchPaths := getConfigSearchPaths(l)
		configNames := []string{"inngest.json", "inngest.yaml", "inngest.yml"}

		for _, searchPath := range searchPaths {
			for _, configName := range configNames {
				fullPath := filepath.Join(searchPath, configName)
				if _, err := os.Stat(fullPath); err == nil {
					if err := loadConfigFromPath(fullPath); err != nil {
						l.Warn("error reading config file", "file", fullPath, "error", err)
						continue
					}
					l.Info("using config", "file", fullPath)
					configLoaded = true
					break
				}
			}
			if configLoaded {
				break
			}
		}
	}

	// Step 2: Load environment variables (higher priority than config files)
	if err := loadEnvironmentVariables(); err != nil {
		return fmt.Errorf("error loading environment variables: %w", err)
	}

	// Step 3: Unmarshal the final configuration
	return unmarshalConfig()
}

// getConfigSearchPaths returns directories to search for config files
func getConfigSearchPaths(l logger.Logger) []string {
	var paths []string

	// Start with current directory and walk up
	if cwd, err := os.Getwd(); err != nil {
		l.Warn("error getting current directory", "error", err)
	} else {
		dir := cwd
		for {
			paths = append(paths, dir)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Add home directory config path
	if homeDir, err := os.UserHomeDir(); err != nil {
		l.Warn("error getting home directory", "error", err)
	} else {
		paths = append(paths, filepath.Join(homeDir, ".config", "inngest"))
	}

	return paths
}

// loadConfigFromPath loads configuration from a specific file path using koanf
func loadConfigFromPath(path string) error {
	ext := filepath.Ext(path)

	var parser koanf.Parser
	switch ext {
	case ".json":
		parser = json.Parser()
	case ".yaml", ".yml":
		parser = yaml.Parser()
	default:
		// Try YAML first, then JSON for files without extensions
		parser = yaml.Parser()
	}

	// Load the config file
	if err := k.Load(file.Provider(path), parser); err != nil {
		// If YAML parsing failed and no extension was provided, try JSON
		if ext == "" {
			if err := k.Load(file.Provider(path), json.Parser()); err != nil {
				return fmt.Errorf("config file must be JSON or YAML: %w", err)
			}
		} else {
			return fmt.Errorf("error parsing config file: %w", err)
		}
	}

	return nil
}

// loadEnvironmentVariables loads environment variables with INNGEST_ prefix
func loadEnvironmentVariables() error {
	// Load environment variables with INNGEST_ prefix
	// The callback function receives the full env var name, so we need to strip the prefix
	return k.Load(env.ProviderWithValue("INNGEST_", "", func(key, value string) (string, interface{}) {
		// Convert environment variable names to config keys
		// INNGEST_SDK_URL -> sdk-url
		// INNGEST_NO_DISCOVERY -> no-discovery
		// INNGEST_POLL_INTERVAL -> poll-interval
		var configKey string
		if strings.HasPrefix(key, "INNGEST_") {
			configKey = strings.ToLower(strings.ReplaceAll(key[8:], "_", "-"))
		} else {
			configKey = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
		}

		// Handle array fields - convert single values and comma-separated values to arrays
		if configKey == "sdk-url" || configKey == "event-key" {
			if strings.Contains(value, ",") {
				return configKey, strings.Split(value, ",")
			}
			// Single value should also be converted to array for consistency
			return configKey, []string{value}
		}

		return configKey, value
	}), nil)
}

// unmarshalConfig unmarshals the loaded configuration into the Config struct
func unmarshalConfig() error {
	loadedConfig = &Config{}
	if err := k.Unmarshal("", loadedConfig); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}
	return nil
}

// GetValue gets a configuration value with proper priority: CLI flag > env var > config file > default
// This replaces the need for envflags helpers since koanf handles the priority automatically
func GetValue(cmd *cli.Command, key, defaultValue string) string {
	// First check if CLI flag is explicitly set (highest priority)
	if cmd.IsSet(key) {
		return cmd.String(key)
	}
	
	// Then check koanf (which includes env vars and config file in correct priority)
	if value := k.String(key); value != "" {
		return value
	}
	
	// Finally return default
	return defaultValue
}

// GetIntValue gets an integer configuration value with proper priority
func GetIntValue(cmd *cli.Command, key string, defaultValue int) int {
	// First check if CLI flag is explicitly set (highest priority)
	if cmd.IsSet(key) {
		return cmd.Int(key)
	}
	
	// Then check koanf (which includes env vars and config file in correct priority)
	if k.Exists(key) {
		return k.Int(key)
	}
	
	// Finally return default
	return defaultValue
}

// GetStringSlice gets a string slice configuration value with proper priority
func GetStringSlice(cmd *cli.Command, key string) []string {
	// First check if CLI flag is explicitly set (highest priority)
	if cmd.IsSet(key) {
		return cmd.StringSlice(key)
	}
	
	// Then check koanf (which includes env vars and config file in correct priority)
	return k.Strings(key)
}

// GetBoolValue gets a boolean configuration value with proper priority
func GetBoolValue(cmd *cli.Command, key string, defaultValue bool) bool {
	// First check if CLI flag is explicitly set (highest priority)
	if cmd.IsSet(key) {
		return cmd.Bool(key)
	}
	
	// Then check koanf (which includes env vars and config file in correct priority)
	if k.Exists(key) {
		return k.Bool(key)
	}
	
	// Finally return default
	return defaultValue
}

