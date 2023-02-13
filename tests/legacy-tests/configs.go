package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/inngest/inngest/pkg/config"
	_ "github.com/inngest/inngest/pkg/config/defaults"
)

// parseConfigs reads all functions from "./configs"
func parseConfigs(ctx context.Context) ([]*Config, error) {
	configs := []*Config{}
	abs, _ := filepath.Abs("./configs")
	entries, _ := os.ReadDir("./configs")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		c := &Config{dir: filepath.Join(abs, e.Name())}
		if err := c.Validate(ctx); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

// Config represents an Inngest setup using a specific
// configuration file for testing.
type Config struct {
	dir string

	config config.Config

	// inngest stores the Inngest command.
	out     *bytes.Buffer
	inngest *exec.Cmd
}

func (c *Config) Validate(ctx context.Context) error {
	reqs := []string{"config.cue", "start.sh"}
	for _, r := range reqs {
		if _, err := os.Stat(filepath.Join(c.dir, r)); os.IsNotExist(err) {
			return fmt.Errorf("%s has no %s", c.dir, r)
		}
	}

	byt, err := os.ReadFile(filepath.Join(c.dir, "config.cue"))
	if err != nil {
		return err
	}

	conf, err := config.Parse(byt)
	if err != nil {
		return err
	}

	c.config = *conf
	return err
}

func (c *Config) Wait() error {
	if c.inngest != nil {
		return c.inngest.Wait()
	}
	return nil
}

func isUp(buf *bytes.Buffer) bool {
	return bytes.Count(buf.Bytes(), []byte("service starting")) == 3
}
