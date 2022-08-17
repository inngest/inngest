package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/joho/godotenv"
)

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
	c.inngest = exec.CommandContext(ctx, "go", "run", "../cmd/main.go", "serve", "-c", filepath.Join(c.dir, "config.cue"), "runner", "executor", "event-api")
	c.inngest.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
			c.Kill()
			return nil
		}
	}
}

func (c *Config) Kill() {
	if c.inngest == nil {
		return
	}
	pgid, err := syscall.Getpgid(c.inngest.Process.Pid)
	if err != nil {
		panic(err)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}
