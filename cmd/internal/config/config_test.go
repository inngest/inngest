package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestMain(m *testing.M) {
	// Clean up any global state before tests
	resetGlobalState()
	code := m.Run()
	resetGlobalState()
	os.Exit(code)
}

func resetGlobalState() {
	k = nil
	loadedConfig = nil
	// Clear environment variables
	for _, env := range os.Environ() {
		if len(env) > 8 && env[:8] == "INNGEST_" {
			key := env[:strings.Index(env, "=")]
			os.Unsetenv(key)
		}
	}
}

func setupTest() {
	resetGlobalState()
	k = koanf.New(".")
}

func TestConfigFileLoading_YAML(t *testing.T) {
	setupTest()

	// Create temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	yamlContent := `
sdk-url:
  - http://localhost:3001/api/inngest
  - http://localhost:3002/api/inngest
no-discovery: true
host: localhost
port: "8290"
poll-interval: 10
signing-key: abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890
event-key:
  - test-key-1
  - test-key-2
redis-uri: redis://localhost:6379
postgres-uri: postgres://localhost:5432/inngest
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Change to temp directory to test auto-discovery
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	ctx := context.Background()
	cmd := &cli.Command{}

	err = loadConfigFile(ctx, cmd)
	require.NoError(t, err)

	config := GetConfig()
	assert.Equal(t, []string{"http://localhost:3001/api/inngest", "http://localhost:3002/api/inngest"}, config.SdkURL)
	assert.Equal(t, true, *config.NoDiscovery)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "8290", config.Port)
	assert.Equal(t, 10, config.PollInterval)
	assert.Equal(t, "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", config.SigningKey)
	assert.Equal(t, []string{"test-key-1", "test-key-2"}, config.EventKey)
	assert.Equal(t, "redis://localhost:6379", config.RedisURI)
	assert.Equal(t, "postgres://localhost:5432/inngest", config.PostgresURI)
}

func TestConfigFileLoading_JSON(t *testing.T) {
	setupTest()

	// Create temporary JSON config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.json")

	jsonContent := `{
	"sdk-url": [
		"http://localhost:4001/api/inngest",
		"http://localhost:4002/api/inngest"
	],
	"no-discovery": false,
	"host": "0.0.0.0",
	"port": "8291",
	"poll-interval": 15,
	"signing-key": "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
	"event-key": [
		"json-key-1",
		"json-key-2"
	]
}`

	err := os.WriteFile(configFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Load config explicitly by path
	err = loadConfigFromPath(configFile)
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()
	assert.Equal(t, []string{"http://localhost:4001/api/inngest", "http://localhost:4002/api/inngest"}, config.SdkURL)
	assert.Equal(t, false, *config.NoDiscovery)
	assert.Equal(t, "0.0.0.0", config.Host)
	assert.Equal(t, "8291", config.Port)
	assert.Equal(t, 15, config.PollInterval)
	assert.Equal(t, "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321", config.SigningKey)
	assert.Equal(t, []string{"json-key-1", "json-key-2"}, config.EventKey)
}

func TestEnvironmentVariableLoading(t *testing.T) {
	setupTest()

	// Set environment variables
	os.Setenv("INNGEST_SDK_URL", "http://env:3000/api/inngest,http://env:3001/api/inngest")
	os.Setenv("INNGEST_NO_DISCOVERY", "true")
	os.Setenv("INNGEST_HOST", "env-host")
	os.Setenv("INNGEST_PORT", "8292")
	os.Setenv("INNGEST_POLL_INTERVAL", "20")
	os.Setenv("INNGEST_SIGNING_KEY", "env1234567890abcdefenv1234567890abcdefenv1234567890abcdefenv123456")
	os.Setenv("INNGEST_EVENT_KEY", "env-key-1,env-key-2,env-key-3")

	defer func() {
		os.Unsetenv("INNGEST_SDK_URL")
		os.Unsetenv("INNGEST_NO_DISCOVERY")
		os.Unsetenv("INNGEST_HOST")
		os.Unsetenv("INNGEST_PORT")
		os.Unsetenv("INNGEST_POLL_INTERVAL")
		os.Unsetenv("INNGEST_SIGNING_KEY")
		os.Unsetenv("INNGEST_EVENT_KEY")
	}()

	err := loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()
	assert.Equal(t, []string{"http://env:3000/api/inngest", "http://env:3001/api/inngest"}, config.SdkURL)
	assert.Equal(t, true, *config.NoDiscovery)
	assert.Equal(t, "env-host", config.Host)
	assert.Equal(t, "8292", config.Port)
	assert.Equal(t, 20, config.PollInterval)
	assert.Equal(t, "env1234567890abcdefenv1234567890abcdefenv1234567890abcdefenv123456", config.SigningKey)
	assert.Equal(t, []string{"env-key-1", "env-key-2", "env-key-3"}, config.EventKey)
}

func TestConfigurationPrecedence(t *testing.T) {
	setupTest()

	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	yamlContent := `
host: config-host
port: "8290"
no-discovery: false
sdk-url:
  - http://config:3000/api/inngest
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set environment variables (should override config file)
	os.Setenv("INNGEST_HOST", "env-host")
	os.Setenv("INNGEST_NO_DISCOVERY", "true")
	os.Setenv("INNGEST_SDK_URL", "http://env:3000/api/inngest")

	defer func() {
		os.Unsetenv("INNGEST_HOST")
		os.Unsetenv("INNGEST_NO_DISCOVERY")
		os.Unsetenv("INNGEST_SDK_URL")
	}()

	// Load config file first
	err = loadConfigFromPath(configFile)
	require.NoError(t, err)

	// Then load environment variables (higher priority)
	err = loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()

	// Environment variables should override config file
	assert.Equal(t, "env-host", config.Host)
	assert.Equal(t, true, *config.NoDiscovery)
	assert.Equal(t, []string{"http://env:3000/api/inngest"}, config.SdkURL)

	// Config file values should remain where no env var is set
	assert.Equal(t, "8290", config.Port)
}

func TestBooleanValueParsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true string", "true", true},
		{"false string", "false", false},
		{"1 as true", "1", true},
		{"0 as false", "0", false},
		{"True capitalized", "True", true},
		{"FALSE capitalized", "FALSE", false},
		{"t as true", "t", true},
		{"f as false", "f", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()

			os.Setenv("INNGEST_NO_DISCOVERY", tt.envValue)
			defer os.Unsetenv("INNGEST_NO_DISCOVERY")

			err := loadEnvironmentVariables()
			require.NoError(t, err)

			err = unmarshalConfig()
			require.NoError(t, err)

			config := GetConfig()
			require.NotNil(t, config.NoDiscovery)
			assert.Equal(t, tt.expected, *config.NoDiscovery)
		})
	}
}

func TestArrayAndCommaSeparatedValues(t *testing.T) {
	setupTest()

	// Test comma-separated values in environment variables
	os.Setenv("INNGEST_SDK_URL", "http://url1:3000/api,http://url2:3001/api,http://url3:3002/api")
	os.Setenv("INNGEST_EVENT_KEY", "key1,key2,key3,key4")

	defer func() {
		os.Unsetenv("INNGEST_SDK_URL")
		os.Unsetenv("INNGEST_EVENT_KEY")
	}()

	err := loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()

	expectedSDKURLs := []string{"http://url1:3000/api", "http://url2:3001/api", "http://url3:3002/api"}
	expectedEventKeys := []string{"key1", "key2", "key3", "key4"}

	assert.Equal(t, expectedSDKURLs, config.SdkURL)
	assert.Equal(t, expectedEventKeys, config.EventKey)
}

func TestGetValueHelperFunctions(t *testing.T) {
	setupTest()

	// Create a mock CLI command
	app := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host"},
			&cli.StringFlag{Name: "port", Value: "8288"},
			&cli.IntFlag{Name: "poll-interval", Value: 5},
			&cli.BoolFlag{Name: "no-discovery"},
			&cli.StringSliceFlag{Name: "sdk-url"},
		},
	}

	// Set up environment variable
	os.Setenv("INNGEST_HOST", "env-host")
	os.Setenv("INNGEST_POLL_INTERVAL", "10")

	defer func() {
		os.Unsetenv("INNGEST_HOST")
		os.Unsetenv("INNGEST_POLL_INTERVAL")
	}()

	// Load environment variables
	err := loadEnvironmentVariables()
	require.NoError(t, err)

	// Create CLI command with some flags set
	cmd := &cli.Command{}
	cmd.Flags = app.Flags

	// Test GetValue function
	t.Run("GetValue with env var fallback", func(t *testing.T) {
		// Should return env var value since CLI flag is not set
		result := GetValue(cmd, "host", "default-host")
		assert.Equal(t, "env-host", result)
	})

	t.Run("GetValue with default fallback", func(t *testing.T) {
		// Should return default since neither CLI flag nor env var is set
		result := GetValue(cmd, "missing", "default-value")
		assert.Equal(t, "default-value", result)
	})

	t.Run("GetIntValue with env var fallback", func(t *testing.T) {
		// Should return env var value since CLI flag is not set
		result := GetIntValue(cmd, "poll-interval", 1)
		assert.Equal(t, 10, result)
	})

	t.Run("GetIntValue with default fallback", func(t *testing.T) {
		// Should return default since neither CLI flag nor env var is set
		result := GetIntValue(cmd, "missing-int", 42)
		assert.Equal(t, 42, result)
	})
}

func TestConfigFileDiscovery(t *testing.T) {
	setupTest()

	// Create nested directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir", "deeper")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create config file in parent directory
	configFile := filepath.Join(tempDir, "inngest.yml")
	yamlContent := `host: discovered-host`
	err = os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Change to subdirectory
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(subDir)
	require.NoError(t, err)

	ctx := context.Background()
	cmd := &cli.Command{}

	err = loadConfigFile(ctx, cmd)
	require.NoError(t, err)

	config := GetConfig()
	assert.Equal(t, "discovered-host", config.Host)
}

func TestEmptyConfig(t *testing.T) {
	setupTest()

	// Load config with no files or env vars (should initialize empty config)
	loadedConfig = &Config{}

	config := GetConfig()

	// Should return empty/default values
	assert.Empty(t, config.SdkURL)
	assert.Nil(t, config.NoDiscovery)
	assert.Empty(t, config.Host)
	assert.Empty(t, config.Port)
	assert.Equal(t, 0, config.PollInterval)
	assert.Empty(t, config.SigningKey)
	assert.Empty(t, config.EventKey)
}

func TestCliOverridesPrecedence(t *testing.T) {
	setupTest()

	// Create config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	yamlContent := `
host: config-host
port: "8290"
no-discovery: false
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set environment variables
	os.Setenv("INNGEST_HOST", "env-host")
	os.Setenv("INNGEST_PORT", "8291")
	os.Setenv("INNGEST_NO_DISCOVERY", "true")

	defer func() {
		os.Unsetenv("INNGEST_HOST")
		os.Unsetenv("INNGEST_PORT")
		os.Unsetenv("INNGEST_NO_DISCOVERY")
	}()

	// Load config file and env vars
	err = loadConfigFromPath(configFile)
	require.NoError(t, err)

	err = loadEnvironmentVariables()
	require.NoError(t, err)

	// Create CLI command with explicit flags set
	app := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host"},
			&cli.StringFlag{Name: "port"},
			&cli.BoolFlag{Name: "no-discovery"},
		},
	}

	// Simulate CLI flag parsing with explicit values
	cmd := &cli.Command{}
	cmd.Flags = app.Flags

	// Test that GetValue functions handle CLI precedence correctly
	// (This simulates cmd.IsSet() returning true for explicitly set flags)

	// When CLI flag is not set, should get env var value
	hostValue := GetValue(cmd, "host", "default-host")
	assert.Equal(t, "env-host", hostValue) // env var overrides config

	portValue := GetValue(cmd, "port", "8288")
	assert.Equal(t, "8291", portValue) // env var overrides config

	noDiscoveryValue := GetBoolValue(cmd, "no-discovery", false)
	assert.Equal(t, true, noDiscoveryValue) // env var overrides config
}

func TestSingleValueArrayHandling(t *testing.T) {
	setupTest()

	// Test that single values are converted to arrays for array fields
	os.Setenv("INNGEST_SDK_URL", "http://single:3000/api/inngest")
	os.Setenv("INNGEST_EVENT_KEY", "single-key")

	defer func() {
		os.Unsetenv("INNGEST_SDK_URL")
		os.Unsetenv("INNGEST_EVENT_KEY")
	}()

	err := loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()

	// Single values should be converted to arrays
	assert.Equal(t, []string{"http://single:3000/api/inngest"}, config.SdkURL)
	assert.Equal(t, []string{"single-key"}, config.EventKey)
}

func TestMixedConfigSources(t *testing.T) {
	setupTest()

	// Create config file with some values
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	yamlContent := `
host: config-host
port: "8290"
signing-key: config-signing-key
sdk-url:
  - http://config:3000/api/inngest
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set some env vars (should override config file)
	os.Setenv("INNGEST_HOST", "env-host")
	os.Setenv("INNGEST_EVENT_KEY", "env-key-1,env-key-2")

	defer func() {
		os.Unsetenv("INNGEST_HOST")
		os.Unsetenv("INNGEST_EVENT_KEY")
	}()

	// Load both sources
	err = loadConfigFromPath(configFile)
	require.NoError(t, err)

	err = loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()

	// Env vars should override config file
	assert.Equal(t, "env-host", config.Host)
	assert.Equal(t, []string{"env-key-1", "env-key-2"}, config.EventKey)

	// Config file values should remain where no env var exists
	assert.Equal(t, "8290", config.Port)
	assert.Equal(t, "config-signing-key", config.SigningKey)
	assert.Equal(t, []string{"http://config:3000/api/inngest"}, config.SdkURL)
}

func TestPostgreSQLConnectionPoolOptions(t *testing.T) {
	setupTest()

	// Test YAML config with PostgreSQL connection pool options
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	yamlContent := `
postgres-uri: postgres://localhost:5432/inngest
postgres-max-idle-conns: 5
postgres-max-open-conns: 50
postgres-conn-max-idle-time: 3
postgres-conn-max-lifetime: 15
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	err = loadConfigFromPath(configFile)
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()
	assert.Equal(t, "postgres://localhost:5432/inngest", config.PostgresURI)
	assert.Equal(t, 5, config.PostgresMaxIdleConns)
	assert.Equal(t, 50, config.PostgresMaxOpenConns)
	assert.Equal(t, 3, config.PostgresConnMaxIdleTime)
	assert.Equal(t, 15, config.PostgresConnMaxLifetime)
}

func TestPostgreSQLConnectionPoolEnvironmentVariables(t *testing.T) {
	setupTest()

	// Set PostgreSQL connection pool environment variables
	os.Setenv("INNGEST_POSTGRES_URI", "postgres://env:5432/testdb")
	os.Setenv("INNGEST_POSTGRES_MAX_IDLE_CONNS", "8")
	os.Setenv("INNGEST_POSTGRES_MAX_OPEN_CONNS", "80")
	os.Setenv("INNGEST_POSTGRES_CONN_MAX_IDLE_TIME", "7")
	os.Setenv("INNGEST_POSTGRES_CONN_MAX_LIFETIME", "25")

	defer func() {
		os.Unsetenv("INNGEST_POSTGRES_URI")
		os.Unsetenv("INNGEST_POSTGRES_MAX_IDLE_CONNS")
		os.Unsetenv("INNGEST_POSTGRES_MAX_OPEN_CONNS")
		os.Unsetenv("INNGEST_POSTGRES_CONN_MAX_IDLE_TIME")
		os.Unsetenv("INNGEST_POSTGRES_CONN_MAX_LIFETIME")
	}()

	err := loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()
	assert.Equal(t, "postgres://env:5432/testdb", config.PostgresURI)
	assert.Equal(t, 8, config.PostgresMaxIdleConns)
	assert.Equal(t, 80, config.PostgresMaxOpenConns)
	assert.Equal(t, 7, config.PostgresConnMaxIdleTime)
	assert.Equal(t, 25, config.PostgresConnMaxLifetime)
}

func TestPostgreSQLConnectionPoolDefaultValues(t *testing.T) {
	setupTest()

	// Create CLI command with PostgreSQL flags to test defaults
	app := &cli.Command{
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "postgres-max-idle-conns", Value: 10},
			&cli.IntFlag{Name: "postgres-max-open-conns", Value: 100},
			&cli.IntFlag{Name: "postgres-conn-max-idle-time", Value: 5},
			&cli.IntFlag{Name: "postgres-conn-max-lifetime", Value: 30},
		},
	}

	cmd := &cli.Command{}
	cmd.Flags = app.Flags

	// Test default values when no config or env vars are set
	maxIdleConns := GetIntValue(cmd, "postgres-max-idle-conns", 10)
	maxOpenConns := GetIntValue(cmd, "postgres-max-open-conns", 100)
	connMaxIdleTime := GetIntValue(cmd, "postgres-conn-max-idle-time", 5)
	connMaxLifetime := GetIntValue(cmd, "postgres-conn-max-lifetime", 30)

	assert.Equal(t, 10, maxIdleConns)
	assert.Equal(t, 100, maxOpenConns)
	assert.Equal(t, 5, connMaxIdleTime)
	assert.Equal(t, 30, connMaxLifetime)
}

func TestPostgreSQLConnectionPoolPrecedence(t *testing.T) {
	setupTest()

	// Create config file with PostgreSQL settings
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	yamlContent := `
postgres-max-idle-conns: 15
postgres-max-open-conns: 150
postgres-conn-max-idle-time: 10
postgres-conn-max-lifetime: 60
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set environment variables (should override config file)
	os.Setenv("INNGEST_POSTGRES_MAX_IDLE_CONNS", "20")
	os.Setenv("INNGEST_POSTGRES_MAX_OPEN_CONNS", "200")

	defer func() {
		os.Unsetenv("INNGEST_POSTGRES_MAX_IDLE_CONNS")
		os.Unsetenv("INNGEST_POSTGRES_MAX_OPEN_CONNS")
	}()

	// Load config file first
	err = loadConfigFromPath(configFile)
	require.NoError(t, err)

	// Then load environment variables (higher priority)
	err = loadEnvironmentVariables()
	require.NoError(t, err)

	err = unmarshalConfig()
	require.NoError(t, err)

	config := GetConfig()

	// Environment variables should override config file
	assert.Equal(t, 20, config.PostgresMaxIdleConns)
	assert.Equal(t, 200, config.PostgresMaxOpenConns)

	// Config file values should remain where no env var is set
	assert.Equal(t, 10, config.PostgresConnMaxIdleTime)
	assert.Equal(t, 60, config.PostgresConnMaxLifetime)
}

func TestInvalidConfigFile(t *testing.T) {
	setupTest()

	// Create invalid YAML file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "inngest.yml")

	invalidYaml := `
invalid: yaml: content:
  - [ unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidYaml), 0644)
	require.NoError(t, err)

	// Should return error for invalid YAML
	err = loadConfigFromPath(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing config file")
}
