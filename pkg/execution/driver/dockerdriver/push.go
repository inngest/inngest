package dockerdriver

import (
	"context"
	"fmt"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/inngest/inngestctl/inngest"
)

const (
	DefaultRegistryHost = "registry.inngest.com"
)

// Digest returns the image digest for a given action.
func Digest(ctx context.Context, a inngest.ActionVersion) (string, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return "", err
	}

	img, err := c.InspectImage(a.DSN)
	if err != nil {
		return "", err
	}

	id := strings.Replace(img.ID, "sha256:", "", 1)
	return id, nil
}

// Push pushes the image, returning the checksum of the image pushed.
func Push(ctx context.Context, a inngest.ActionVersion, creds []byte) (string, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return "", err
	}

	imageTag := a.DSN
	if a.Runtime.Runtime.(inngest.RuntimeDocker).Image != "" {
		imageTag = a.Runtime.Runtime.(inngest.RuntimeDocker).Image
	}

	img, err := c.InspectImage(imageTag)
	if err != nil {
		return "", fmt.Errorf("error finding image '%s' to push: %w", imageTag, err)
	}

	id := strings.Replace(img.ID, "sha256:", "", 1)

	host := DefaultRegistryHost
	if os.Getenv("INNGEST_REGISTRY") != "" {
		host = os.Getenv("INNGEST_REGISTRY")
	}

	if host == DefaultRegistryHost && (img.Architecture != "amd64" || img.OS != "linux") {
		return "", fmt.Errorf("image architecture is not linux/amd64, please rebuild")
	}

	image := fmt.Sprintf("%s/%s", host, a.DSN)
	err = c.TagImage(imageTag, docker.TagImageOptions{
		Repo: image,
		Tag:  a.Version.Tag(),
	})
	if err != nil {
		return "", fmt.Errorf("error tagging image: %w", err)
	}

	err = c.PushImage(docker.PushImageOptions{
		Name:         image,
		Tag:          a.Version.Tag(),
		Registry:     host,
		OutputStream: os.Stderr,
	}, docker.AuthConfiguration{
		Username: "jwt",
		Password: string(creds),
	})

	return id, err
}
