package docker

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
	"github.com/inngest/inngestctl/inngest"
)

func NewExecutor() (*DockerExecutor, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &DockerExecutor{
		client: c,
	}, nil
}

type DockerExecutor struct {
	client *docker.Client
}

type handle struct {
	c *docker.Container
}

// Execute is a blocking operation which runs a container.
func (d *DockerExecutor) Execute(ctx context.Context, action inngest.ActionVersion, state map[string]interface{}) (map[string]interface{}, error) {
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

	h, err = d.start(ctx, action, state)
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
	_ = exit

	byt, err := io.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("error reading docker output: %w", err)
	}

	if len(byt) == 0 {
		byt, _ := io.ReadAll(stderr)
		if len(byt) > 0 {
			return nil, fmt.Errorf("no stdout received.  stderr: %s", string(byt))
		}
		return nil, fmt.Errorf("no stdout received")
	}

	split := bytes.SplitN(byt, []byte(" "), 2)
	_, content := split[0], split[1]

	// Return the output as JSON
	data := map[string]interface{}{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("error reading output as JSON: \n%s", string(byt))
	}

	return data, nil
}

func (d *DockerExecutor) start(ctx context.Context, action inngest.ActionVersion, state map[string]interface{}) (*handle, error) {
	opts, err := d.startOpts(ctx, action, state)
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
		<-time.After(500 * time.Millisecond)
	}

	// Clean up.
	_ = d.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	})
	return nil, fmt.Errorf("unable to start container")
}

func (d *DockerExecutor) startOpts(ctx context.Context, action inngest.ActionVersion, state map[string]interface{}) (docker.CreateContainerOptions, error) {

	marshalled, err := json.Marshal(map[string]interface{}{
		"args_version": 1,
		"metadata": map[string]interface{}{
			"js": "export default function({ event }) { return { event } }",
		},
		"baggage": map[string]interface{}{
			"WorkspaceEvent": map[string]interface{}{
				"Event": state["event"],
			},
			"Actions": map[uint]map[string]interface{}{
				0: {},
			},
		},
	})
	if err != nil {
		return docker.CreateContainerOptions{}, fmt.Errorf("error marshalling state")
	}

	byt := make([]byte, 3)
	if _, err := rand.Read(byt); err != nil {
		return docker.CreateContainerOptions{}, fmt.Errorf("error generating ID: %w", err)
	}
	name := fmt.Sprintf("%s-%s", slug.Make(action.Name), hex.EncodeToString(byt))
	return docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image: action.DSN,
			Cmd:   []string{string(marshalled)},
		},
	}, nil
}

func (d *DockerExecutor) watch(ctx context.Context, h *handle) (stdout, stderr io.Reader, err error) {
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
