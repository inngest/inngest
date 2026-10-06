package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

type dockerRunner func(context.Context, io.Writer, io.Writer, ...string) error

func runDocker(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

type dockerInspection struct {
	ID           string `json:"Id"`
	OS           string `json:"Os"`
	Architecture string
	Config       struct {
		Entrypoint, Cmd, Env         []string
		WorkingDir, User, StopSignal string
		Labels                       map[string]string
		ExposedPorts, Volumes        map[string]json.RawMessage
		Healthcheck                  *struct{ Test []string }
	}
}

func (d dockerInspection) config() (*apiv2.ImageConfig, error) {
	if d.OS != "linux" || d.Architecture != "amd64" {
		return nil, errors.New("custom images must target linux/amd64")
	}
	if h := d.Config.Healthcheck; h != nil && len(h.Test) > 0 && h.Test[0] != "NONE" {
		return nil, errors.New("Docker HEALTHCHECK is unsupported; disable it with HEALTHCHECK NONE")
	}
	c := &apiv2.ImageConfig{Entrypoint: d.Config.Entrypoint, Cmd: d.Config.Cmd, Env: map[string]string{}, WorkingDir: d.Config.WorkingDir, User: d.Config.User, StopSignal: d.Config.StopSignal, Labels: d.Config.Labels}
	for _, entry := range d.Config.Env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, errors.New("image contains invalid environment metadata")
		}
		c.Env[key] = value
	}
	for port := range d.Config.ExposedPorts {
		c.ExposedPorts = append(c.ExposedPorts, port)
	}
	for volume := range d.Config.Volumes {
		c.Volumes = append(c.Volumes, volume)
	}
	sort.Strings(c.ExposedPorts)
	sort.Strings(c.Volumes)
	return c, nil
}

func exportDocker(ctx context.Context, run dockerRunner, directory, file, target string, args []string, temp string, output, logs io.Writer) (*apiv2.ImageConfig, error) {
	iidfile := filepath.Join(temp, "image-id")
	build := []string{"buildx", "build", "--platform", "linux/amd64", "--load", "--iidfile", iidfile, "--file", filepath.Join(directory, file)}
	if target != "" {
		build = append(build, "--target", target)
	}
	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			return nil, errors.New("build arguments must use NAME=value")
		}
		build = append(build, "--build-arg", arg)
	}
	build = append(build, directory)
	if err := run(ctx, logs, logs, build...); err != nil {
		return nil, fmt.Errorf("docker build: %w", err)
	}
	id, err := os.ReadFile(iidfile)
	if err != nil {
		return nil, err
	}
	imageID := strings.TrimSpace(string(id))
	if !strings.HasPrefix(imageID, "sha256:") || len(imageID) != 71 {
		return nil, errors.New("Docker did not produce an immutable image ID")
	}
	var inspection bytes.Buffer
	if err := run(ctx, &inspection, logs, "image", "inspect", imageID); err != nil {
		return nil, fmt.Errorf("docker inspect: %w", err)
	}
	var images []dockerInspection
	if err := json.Unmarshal(inspection.Bytes(), &images); err != nil || len(images) != 1 {
		return nil, errors.New("Docker returned invalid image metadata")
	}
	config, err := images[0].config()
	if err != nil {
		return nil, err
	}
	var created bytes.Buffer
	// Create never executes the image. Override the entrypoint so even a
	// scratch image with no CMD can be exported without needing /bin/sh.
	if err := run(ctx, &created, logs, "create", "--platform", "linux/amd64", "--entrypoint", "/inngest-export-unused", imageID); err != nil {
		return nil, fmt.Errorf("docker create: %w", err)
	}
	container := strings.TrimSpace(created.String())
	if !containerIDPattern.MatchString(container) {
		return nil, errors.New("Docker returned invalid container ID")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		_ = run(cleanup, io.Discard, logs, "rm", "--force", "--volumes", container)
	}()
	if err := run(ctx, output, logs, "export", container); err != nil {
		return nil, fmt.Errorf("docker export: %w", err)
	}
	return config, nil
}
