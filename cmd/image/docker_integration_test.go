package image

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRealDockerExport(t *testing.T) {
	if os.Getenv("INNGEST_IMAGE_DOCKER_TEST") != "1" {
		t.Skip("set INNGEST_IMAGE_DOCKER_TEST=1 to run local Docker qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	dir, temp := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello"), []byte("custom image fixture\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\nCOPY hello /work/hello\nWORKDIR /work\nENV GREETING=hello\nUSER 1000:1000\nSTOPSIGNAL SIGQUIT\nLABEL test.id="+uuid.NewString()+"\nENTRYPOINT [\"/app\"]\nCMD [\"serve\"]\n"), 0600))
	var out, logs bytes.Buffer
	config, err := exportDocker(ctx, runDocker, dir, "Dockerfile", "", nil, temp, &out, &logs)
	t.Cleanup(func() {
		id, err := os.ReadFile(filepath.Join(temp, "image-id"))
		if err == nil {
			_ = runDocker(context.Background(), io.Discard, io.Discard, "image", "rm", strings.TrimSpace(string(id)))
		}
	})
	require.NoError(t, err, logs.String())
	require.Equal(t, "1000:1000", config.User)
	require.Equal(t, "SIGQUIT", config.StopSignal)
	require.Equal(t, "/work", config.WorkingDir)
	require.Equal(t, "hello", config.Env["GREETING"])
	tr := tar.NewReader(&out)
	found := false
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if h.Name == "work/hello" {
			body, err := io.ReadAll(tr)
			require.NoError(t, err)
			require.Equal(t, "custom image fixture\n", string(body))
			found = true
		}
	}
	require.True(t, found, "export must preserve the user's filesystem without running ENTRYPOINT")
}
