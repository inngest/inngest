package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/inngest/inngest-cli/pkg/config"
	_ "github.com/inngest/inngest-cli/pkg/config/defaults"
	"github.com/joho/godotenv"
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

func (c *Config) Up(ctx context.Context) error {
	// Run start.sh, to pre-up the config.
	fmt.Println("> Running start.sh")
	start := exec.Command("/bin/bash", "-c", fmt.Sprintf("cd %s && ./start.sh", c.dir))
	if out, err := start.CombinedOutput(); err != nil {
		return cmdError{err: fmt.Errorf("error running start.sh: %w", err), out: out}
	}
	fmt.Println("> Running serve")

	// Attempt to read the env file present.
	_ = godotenv.Load(filepath.Join(c.dir, "env"))

	// Run Inngest as an all-in-one server using this config.
	buf := &bytes.Buffer{}

	// This would allow us to stream output to stderr by changing inngest.Stderr to
	// the multiwriter.
	// w := io.MultiWriter(buf, os.Stderr)

	c.out = buf
	c.inngest = exec.CommandContext(ctx, "inngest", "serve", "-c", filepath.Join(c.dir, "config.cue"), "runner", "executor", "events-api")
	c.inngest.Env = os.Environ()
	c.inngest.Stderr = buf
	c.inngest.Stdout = buf
	if err := c.inngest.Start(); err != nil {
		return err
	}

	// TODO: Implement service hearbeating and statuses via an PI endpoint.
	// When running StartAll, report each individual service & an aggregate.
	// Have services report when they're running.
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			if isUp(c.out) {
				<-time.After(500 * time.Millisecond)
				return nil
			}
		case <-timeout:
			return cmdError{err: fmt.Errorf("inngest didn't start within timeout"), out: c.out.Bytes()}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Config) Wait() error {
	if c.inngest != nil {
		// TODO: Handle ctx cnacellation nicely
		return c.inngest.Wait()
	}
	return nil
}

func isUp(buf *bytes.Buffer) bool {
	return bytes.Count(buf.Bytes(), []byte("service starting")) == 3
}
