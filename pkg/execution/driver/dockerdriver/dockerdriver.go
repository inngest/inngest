package dockerdriver

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gosimple/slug"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function/env"
)

// New returns a basic docker implementation for running containers within a workflow.  This
// executes containers running on a local docker instance, then monitors the containers until
// it finishes running to scrape the output.  It does this in a blocking, synchronous manner.
//
// NOTE: This does not persist information about running containers, so if the executor
// terminates the container's output will not be returned.  A more reliable way of executing
// containers for the docker runtime would be to use a scheduler which maintains state in a
// fault tolerant manner.
func New() (driver.Driver, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &dockerExec{
		client: c,
	}, nil
}

type dockerExec struct {
	client *docker.Client
	config *Config

	envreader env.EnvReader
}

type handle struct {
	c *docker.Container
}

func (dockerExec) RuntimeType() string {
	return "docker"
}

// SetEnvReader fulfils the driver.EnvManager interface, allowing the docker driver
// to read env variables for each function ran.  This is used within the dev server
// for local management of secrets during development.
func (d *dockerExec) SetEnvReader(r env.EnvReader) {
	d.envreader = r
}

func (d *dockerExec) Execute(ctx context.Context, s state.State, action inngest.ActionVersion, edge inngest.Edge, wf inngest.Step, idx int) (*state.DriverResponse, error) {
	var (
		h   *handle
		err error
	)

	// always clean the container up.
	defer func() {
		if h == nil {
			return
		}
		_ = d.client.StopContainer(h.c.ID, 0)
		_ = d.client.RemoveContainer(docker.RemoveContainerOptions{
			ID: h.c.ID,
		})
	}()

	h, err = d.start(ctx, s, wf, idx)
	if err != nil {
		return nil, err
	}

	stdout, stderr, err := d.watch(ctx, h)
	if err != nil {
		return nil, err
	}

	exit, err := d.client.WaitContainer(h.c.ID)
	if err != nil {
		return nil, err
	}

	byt, err := io.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("error reading docker output: %w", err)
	}

	resp := &state.DriverResponse{
		Output:        map[string]any{},
		ActionVersion: action.Version,
	}

	if len(byt) == 0 {
		byt, _ = io.ReadAll(stderr)
		resp.Output.(map[string]any)["stderr"] = string(byt)
	}
	if exit != 0 {
		resp.Err = fmt.Errorf("Non-zero status code: %d\nOutput: %s", exit, string(byt))
	}

	// note: right now we're ignoring timestamps.
	split := bytes.SplitN(byt, []byte(" "), 2)
	_, content := split[0], split[1]

	// XXX: we could support ndjson here, treating the last line as output and any previous lines
	// as stderr logs.

	// Return the output as JSON
	if err := json.Unmarshal(content, &resp.Output); err != nil {
		resp.Output.(map[string]any)["body"] = string(content)
	}

	return resp, nil
}

// start creates and runs the container.
func (d *dockerExec) start(ctx context.Context, state state.State, wa inngest.Step, idx int) (*handle, error) {
	opts, err := d.startOpts(ctx, state, wa, idx)
	if err != nil {
		return nil, err
	}

	container, err := d.client.CreateContainer(opts)
	if err != nil {
		return nil, fmt.Errorf("error creating container: %w", err)
	}

	for i := 0; i <= 4; i++ {
		err = d.client.StartContainer(container.ID, nil)
		if err == nil || strings.Contains(err.Error(), "Container already running") {
			return &handle{c: container}, nil
		}
		<-time.After(100 * time.Millisecond)
	}

	// Clean up.
	_ = d.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	})
	return nil, fmt.Errorf("unable to start container")
}

func (d *dockerExec) startOpts(ctx context.Context, state state.State, wa inngest.Step, idx int) (docker.CreateContainerOptions, error) {
	marshalled, err := driver.MarshalV1(ctx, state, wa, idx, "local")
	if err != nil {
		return docker.CreateContainerOptions{}, fmt.Errorf("error marshalling state")
	}
	byt := make([]byte, 3)
	if _, err := rand.Read(byt); err != nil {
		return docker.CreateContainerOptions{}, fmt.Errorf("error generating ID: %w", err)
	}
	name := fmt.Sprintf("%s-%s-%s", state.RunID(), slug.Make(wa.Name), hex.EncodeToString(byt))

	env := []string{}
	if d.envreader != nil {
		parsed := d.envreader.Read(ctx, state.Workflow().ID)
		if parsed != nil {
			env = []string{}
			for k, v := range parsed {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	return docker.CreateContainerOptions{
		Name: name,
		HostConfig: &docker.HostConfig{
			// This driver uses net mode host.
			NetworkMode: d.config.Network,
		},
		Config: &docker.Config{
			Image: wa.DSN,
			Cmd:   []string{string(marshalled)},
			Env:   env,
		},
	}, nil
}

func (d *dockerExec) watch(ctx context.Context, h *handle) (stdout, stderr io.Reader, err error) {
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	logs := docker.LogsOptions{
		Context:      ctx,
		Container:    h.c.ID,
		OutputStream: stdout.(*bytes.Buffer),
		ErrorStream:  stderr.(*bytes.Buffer),
		Follow:       true,
		Timestamps:   true,
		Stdout:       true,
		Stderr:       true,
	}
	if err := d.client.Logs(logs); err != nil {
		return nil, nil, fmt.Errorf("error fetching logs: %w", err)
	}
	return stdout, stderr, nil
}
