package inngest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/inngest/inngestctl/inngest/client"
)

// DeployActionOptions
type DeployActionOptions struct {
	Config string
	// Version is the deserialized cue configuration derived from Config.
	// If not specified, Config will be deserialized automatically.
	Version *ActionVersion

	Client client.Client
}

func DeployAction(ctx context.Context, opts DeployActionOptions) error {
	if opts.Version == nil {
		version, err := ParseAction(opts.Config)
		if err != nil {
			return err
		}
		opts.Version = version
	}

	// TODO: Log creating action
	_, err := opts.Client.CreateAction(ctx, opts.Config)
	if err != nil {
		return err
	}

	if opts.Version.Runtime.RuntimeType() == "docker" {
		// TODO: Log pushing image
		runtime := opts.Version.Runtime.Runtime.(RuntimeDocker)
		return deployImage(ctx, deployImageOptions{
			image:       runtime.Image,
			credentials: opts.Client.Credentials(),
		})
	}

	return fmt.Errorf("unknown runtime type: %s", opts.Version.Runtime.RuntimeType())
}

// deployImage deploys an image to Inngest's registry, allowing the container to be used
// as an action within a workflow.
//
// The action must have been registered within the current account prior to pushing the
// image, else this will error.
func deployImage(ctx context.Context, a deployImageOptions) (err error) {
	dkr, err := docker.NewClientWithOpts(docker.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	if a.dockerfile != "" {
		if a.image, err = buildImage(ctx, a, dkr); err != nil {
			return err
		}
	}

	if a.image == "" {
		return fmt.Errorf("no image specified")
	}

	return pushImage(ctx, a, dkr)
}

func pushImage(ctx context.Context, a deployImageOptions, dkr *docker.Client) error {
	rc, err := dkr.ImagePush(ctx, a.image, types.ImagePushOptions{
		RegistryAuth: a.Auth(),
	})
	if err != nil {
		return err
	}
	defer rc.Close()

	err = jsonmessage.DisplayJSONMessagesStream(rc, os.Stderr, uintptr(syscall.Stderr), true, nil)
	if err != nil {
		var msgerr *jsonmessage.JSONError
		if errors.As(err, &msgerr) {
			return fmt.Errorf("%s", msgerr.Message)
		}
		return fmt.Errorf("error displaying push status: %w", err)
	}

	return nil
}

func buildImage(ctx context.Context, a deployImageOptions, dkr *docker.Client) (string, error) {
	return "", nil
}

type deployImageOptions struct {
	image string

	// TODO: (tonyhb) allow building of dockerfile from file location
	dockerfile  string
	credentials []byte
}

func (a deployImageOptions) Auth() string {
	authConfig := types.AuthConfig{
		Username: "jwt",
		Password: string(a.credentials),
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(encodedJSON)
}
