package imagecmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRefFlag(t *testing.T) {
	validate := refFlag(true).Validator
	for _, ref := range []string{"app", "app:v1", "my-app.v2:1.2.3", "a_b:latest"} {
		require.NoError(t, validate(ref), ref)
	}
	for _, ref := range []string{"", "App", "app:V1", "ghcr.io/acme/app:v1", "acme/app", "default", "-app", "app:"} {
		require.Error(t, validate(ref), ref)
	}
}

func TestParseRef(t *testing.T) {
	name, tag, err := parseRef("app:v1")
	require.NoError(t, err)
	require.Equal(t, "app", name)
	require.Equal(t, "v1", tag)

	name, tag, err = parseRef("app")
	require.NoError(t, err)
	require.Equal(t, "app", name)
	require.Equal(t, "latest", tag)

	for _, ref := range []string{"", "app:", "App:v1", "ghcr.io/acme/app:v1", "default", "app@sha256:abc"} {
		_, _, err := parseRef(ref)
		require.Error(t, err, ref)
	}
}
