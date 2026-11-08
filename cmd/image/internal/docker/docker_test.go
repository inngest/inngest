package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRejectsOtherPlatform(t *testing.T) {
	c := &Client{path: "docker-not-run"}
	for _, args := range [][]string{
		{"--platform", "linux/arm64", "."},
		{"--platform=linux/arm64", "."},
		{"--platform", "linux/amd64,linux/arm64", "."},
	} {
		err := c.Build(context.Background(), "linux/amd64", args)
		require.ErrorContains(t, err, "images must be built for linux/amd64", args)
	}
}

func TestImagePlatform(t *testing.T) {
	require.Equal(t, "linux/amd64", Image{OS: "linux", Architecture: "amd64"}.Platform())
	require.Equal(t, "linux/arm/v7", Image{OS: "linux", Architecture: "arm", Variant: "v7"}.Platform())
}
