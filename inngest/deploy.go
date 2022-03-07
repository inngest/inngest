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
	"github.com/inngest/inngestctl/inngest/log"
)

const (
	DefaultRegistryHost = "registry.inngest.com"
)

// DeployActionOptions
type DeployActionOptions struct {
	PushOnly bool
	Config   string
	// Version is the deserialized cue configuration derived from Config.
	// If not specified, Config will be deserialized automatically.
	Version *ActionVersion

	Client client.Client
}

// DeployAction pushes the action to Inngest, making it available for use within
// all workflows.
func DeployAction(ctx context.Context, opts DeployActionOptions) (*ActionVersion, error) {
	log.From(ctx).Info().Msgf("Deploying action %s\n", opts.Version.DSN)

	if opts.Version == nil {
		version, err := ParseAction(opts.Config)
		if err != nil {
			return nil, err
		}
		opts.Version = version
	}

	if !opts.PushOnly {
		_, err := opts.Client.CreateAction(ctx, opts.Config)
		if err != nil {
			return nil, err
		}
	}

	switch opts.Version.Runtime.RuntimeType() {
	case "docker":
		runtime := opts.Version.Runtime.Runtime.(RuntimeDocker)
		err := prepareAndPushImage(ctx, deployImageOptions{
			version:     opts.Version,
			image:       runtime.Image,
			credentials: opts.Client.Credentials(),
		})
		if err != nil {
			return opts.Version, err
		}
	default:
		return nil, fmt.Errorf("unknown runtime type: %s", opts.Version.Runtime.RuntimeType())
	}

	// Ensure that the version is enabled, allowing all workflows to automatically
	// use the action.
	_, err := opts.Client.UpdateActionVersion(ctx, client.ActionVersionQualifier{
		DSN:          opts.Version.DSN,
		VersionMajor: opts.Version.Version.Major,
		VersionMinor: opts.Version.Version.Minor,
	}, true)
	return opts.Version, err
}

// prepareAndPushImage pushes an image to Inngest's registry, allowing the container
// to be used as an action within a workflow.
//
// The action must have been registered within the current account prior to pushing the
// image, else this will error.
func prepareAndPushImage(ctx context.Context, a deployImageOptions) (err error) {
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
		return errors.New("no image specified")
	}

	resp, _, err := dkr.ImageInspectWithRaw(ctx, a.image)
	if err != nil {
		return err
	}
	if resp.Architecture != "amd64" || resp.Os != "linux" {
		return errors.New("image architecture is not linux/amd64, please rebuild")
	}

	return pushImage(ctx, a, dkr)
}

func pushImage(ctx context.Context, a deployImageOptions, dkr *docker.Client) error {
	host := DefaultRegistryHost
	if os.Getenv("INNGEST_REGISTRY") != "" {
		host = os.Getenv("INNGEST_REGISTRY")
	}

	tag := fmt.Sprintf("%s/%s:%d-%d", host, a.version.DSN, a.version.Version.Major, a.version.Version.Minor)
	if err := dkr.ImageTag(ctx, a.image, tag); err != nil {
		return err
	}

	defer func() {
		if _, err := dkr.ImageRemove(ctx, tag, types.ImageRemoveOptions{}); err != nil {
			log.From(ctx).Info().Msgf("failed to remove docker image %v", err)
		}

	}()

	rc, err := dkr.ImagePush(ctx, tag, types.ImagePushOptions{
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
	version *ActionVersion
	image   string

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
