package dockerdriver

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gosimple/slug"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/execution/driver"
	"github.com/inngest/inngestctl/pkg/execution/state"
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
}

type handle struct {
	c *docker.Container
}

func (dockerExec) RuntimeType() string {
	return "docker"
}

func (d *dockerExec) Execute(ctx context.Context, state state.State, action inngest.ActionVersion, wf inngest.Step) (*driver.Response, error) {
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

	h, err = d.start(ctx, state, wf)
	if err != nil {
		return nil, err
	}

	stdout, _, err := d.watch(ctx, h)
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

	resp := &driver.Response{
		Output: map[string]interface{}{},
	}
	if exit != 0 {
		resp.Err = fmt.Errorf("non-zero status code: %d", exit)
	}

	if len(byt) == 0 {
		// TODO: read and log stderr.
		// byt, _ := io.ReadAll(stderr)
		return resp, nil
	}

	// note: right now we're ignoring timestamps.
	split := bytes.SplitN(byt, []byte(" "), 2)
	_, content := split[0], split[1]

	// XXX: we could support ndjson here, treating the last line as output and any previous lines
	// as stderr logs.

	// Return the output as JSON
	if err := json.Unmarshal(content, &resp.Output); err != nil {
		resp.Output["body"] = string(content)
	}

	return resp, nil
}

// start creates and runs the container.
func (d *dockerExec) start(ctx context.Context, state state.State, wa inngest.Step) (*handle, error) {
	opts, err := d.startOpts(ctx, state, wa)
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

func (d *dockerExec) startOpts(ctx context.Context, state state.State, wa inngest.Step) (docker.CreateContainerOptions, error) {
	marshalled, err := json.Marshal(map[string]interface{}{
		"event": state.Event(),
		"steps": state.Actions(),
		"ctx": map[string]interface{}{
			"workflow_id": state.WorkflowID(),
		},
	})
	if err != nil {
		return docker.CreateContainerOptions{}, fmt.Errorf("error marshalling state")
	}
	byt := make([]byte, 3)
	if _, err := rand.Read(byt); err != nil {
		return docker.CreateContainerOptions{}, fmt.Errorf("error generating ID: %w", err)
	}
	name := fmt.Sprintf("%s-%s-%s", state.RunID(), slug.Make(wa.Name), hex.EncodeToString(byt))
	return docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image: wa.DSN,
			Cmd:   []string{string(marshalled)},
			// Add all env vars from the local machine
			Env: os.Environ(),
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
