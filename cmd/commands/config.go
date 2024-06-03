package commands

import (
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

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

type serverConfig struct {
	EnableDiscovery bool
	EnablePoll      bool
	Host            string
	PollInterval    int
	Port            int
	RetryInterval   int
	Tick            int
	URLs            []string
}

func loadServerConfig(cmd *cobra.Command) (*serverConfig, error) {
	c := serverConfig{}

	host, _ := cmd.Flags().GetString("host")
	c.Host = host

	noPoll, _ := cmd.Flags().GetBool("no-poll")
	c.EnablePoll = !noPoll

	noDiscovery, _ := cmd.Flags().GetBool("no-discovery")
	c.EnableDiscovery = !noDiscovery

	pollInterval, _ := cmd.Flags().GetInt("poll-interval")
	c.PollInterval = pollInterval

	if port, err := strconv.Atoi(cmd.Flag("port").Value.String()); err != nil {
		return nil, err
	} else {
		c.Port = port
	}

	retryInterval, _ := cmd.Flags().GetInt("retry-interval")
	c.RetryInterval = retryInterval

	tick, _ := cmd.Flags().GetInt("tick")
	c.Tick = tick

	urls, _ := cmd.Flags().GetStringSlice("sdk-url")
	c.URLs = urls

	configPath, _ := cmd.Flags().GetString("config")
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return &c, nil
	}
	var fileConfig fileConfigV1
	if err = yaml.Unmarshal(yamlFile, &fileConfig); err != nil {
		log.Fatalf("error unmarshaling YAML file: %s", err)
	}
	if fileConfig.Version == "" {
		return &c, nil
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

	return &c, nil
}
