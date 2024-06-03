package devconfig

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	EnableDiscovery bool
	EnablePoll      bool
	Host            string
	PollInterval    int
	Port            int
	RetryInterval   int
	Tick            int
	URLs            []string
}

func Read(cmd *cobra.Command) (Config, error) {
	c := Config{}
	var err error

	c, err = applyCLIConfig(c, cmd)
	if err != nil {
		return c, err
	}

	configPath, _ := cmd.Flags().GetString("config")
	c, err = applyFileConfig(c, configPath)
	if err != nil {
		return c, err
	}

	return c, nil
}

func applyCLIConfig(c Config, cmd *cobra.Command) (Config, error) {
	host, _ := cmd.Flags().GetString("host")
	c.Host = host

	noPoll, _ := cmd.Flags().GetBool("no-poll")
	c.EnablePoll = !noPoll

	noDiscovery, _ := cmd.Flags().GetBool("no-discovery")
	c.EnableDiscovery = !noDiscovery

	pollInterval, _ := cmd.Flags().GetInt("poll-interval")
	c.PollInterval = pollInterval

	if port, err := strconv.Atoi(cmd.Flag("port").Value.String()); err != nil {
		return c, err
	} else {
		c.Port = port
	}

	retryInterval, _ := cmd.Flags().GetInt("retry-interval")
	c.RetryInterval = retryInterval

	tick, _ := cmd.Flags().GetInt("tick")
	c.Tick = tick

	urls, _ := cmd.Flags().GetStringSlice("sdk-url")
	c.URLs = urls

	return c, nil
}

type fileConfigV1 struct {
	EnableDiscovery *bool     `yaml:"enableDiscovery"`
	EnablePoll      *bool     `yaml:"enablePoll"`
	Host            *string   `yaml:"host"`
	PollInterval    *int      `yaml:"pollInterval"`
	Port            *int      `yaml:"port"`
	RetryInterval   *int      `yaml:"retryInterval"`
	Tick            *int      `yaml:"tick"`
	URLs            *[]string `yaml:"urls"`
	Version         string    `yaml:"version"`
}

func applyFileConfig(c Config, path string) (Config, error) {
	byt, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		// The config wasn't found, so return the unmodified config
		return c, nil
	}

	// We've found a config file, so we'll load it and override fields
	var fileConfig fileConfigV1
	if err = yaml.Unmarshal(byt, &fileConfig); err != nil {
		return c, errors.Wrap(err, "error unmarshaling YAML file")
	}
	if fileConfig.Version == "" {
		return c, nil
	}

	if fileConfig.EnableDiscovery != nil {
		c.EnableDiscovery = *fileConfig.EnableDiscovery
	}

	if fileConfig.EnablePoll != nil {
		c.EnablePoll = *fileConfig.EnablePoll
	}

	if fileConfig.Host != nil {
		c.Host = *fileConfig.Host
	}

	if fileConfig.PollInterval != nil {
		c.PollInterval = *fileConfig.PollInterval
	}

	if fileConfig.Port != nil {
		c.Port = *fileConfig.Port
	}

	if fileConfig.RetryInterval != nil {
		c.RetryInterval = *fileConfig.RetryInterval
	}

	if fileConfig.Tick != nil {
		c.Tick = *fileConfig.Tick
	}

	if fileConfig.URLs != nil {
		c.URLs = *fileConfig.URLs
	}

	return c, nil
}
