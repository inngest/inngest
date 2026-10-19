package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Client runs the docker binary found in PATH.
type Client struct {
	path   string
	stdout io.Writer
	stderr io.Writer
}

// New finds docker in PATH.  docker output goes to stdout and stderr.
func New(stdout, stderr io.Writer) (*Client, error) {
	// TODO: Look up podman as well as docker in the future.
	path, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker is required to build images but was not found in PATH: %w", err)
	}
	return &Client{
		path:   path,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// Build runs `docker build` for one platform.  a --platform in args that
// differs from platform is an error, because docker would build every
// platform listed.  --load puts the image in the local image store.  without
// it, a docker-container builder keeps the image only in its build cache and
// the local tag still points to the previous image.
func (c *Client) Build(ctx context.Context, platform string, args []string) error {
	for _, value := range flagValues(args, "--platform") {
		if value != platform {
			return fmt.Errorf("images must be built for %s; remove --platform %s", platform, value)
		}
	}
	args = append([]string{"build", "--platform", platform, "--load"}, args...)
	if err := c.run(ctx, args...); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}
	return nil
}

// flagValues returns the values of the named flags in args, written as
// "--name value" or "--name=value".
func flagValues(args []string, names ...string) []string {
	var values []string
	for i, arg := range args {
		for _, name := range names {
			switch {
			case arg == name && i+1 < len(args):
				values = append(values, args[i+1])
			case strings.HasPrefix(arg, name+"="):
				values = append(values, strings.TrimPrefix(arg, name+"="))
			}
		}
	}
	return values
}

// Save writes the image as a tar archive to path.
func (c *Client) Save(ctx context.Context, tag, path string) error {
	if err := c.run(ctx, "save", "-o", path, tag); err != nil {
		return fmt.Errorf("docker save failed: %w", err)
	}
	return nil
}

// Image is the part of `docker image inspect` output that the build checks.
type Image struct {
	OS           string `json:"Os"`
	Architecture string `json:"Architecture"`
	Variant      string `json:"Variant,omitempty"`
}

// Platform returns the image platform in the form --platform takes, such
// as linux/amd64.
func (i Image) Platform() string {
	if i.Variant != "" {
		return i.OS + "/" + i.Architecture + "/" + i.Variant
	}
	return i.OS + "/" + i.Architecture
}

// Inspect returns the image from `docker image inspect`.
func (c *Client) Inspect(ctx context.Context, tag string) (*Image, error) {
	out, err := exec.CommandContext(ctx, c.path, "image", "inspect", "--format", "{{json .}}", tag).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return nil, fmt.Errorf("docker image inspect failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
	}
	if err != nil {
		return nil, fmt.Errorf("docker image inspect failed: %w", err)
	}
	var image Image
	if err := json.Unmarshal(out, &image); err != nil {
		return nil, fmt.Errorf("docker image inspect returned invalid JSON: %w", err)
	}
	return &image, nil
}

func (c *Client) run(ctx context.Context, args ...string) error {
	ecmd := exec.CommandContext(ctx, c.path, args...)
	ecmd.Stdout = c.stdout
	ecmd.Stderr = c.stderr
	return ecmd.Run()
}
