package imagecmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// fakeDocker puts a docker script first in PATH.  it writes its args and
// its stdin to files in the returned directory.
func fakeDocker(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake docker is a shell script")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + dir + "/args\"\ncat > \"" + dir + "/stdin\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func TestBuildPipesStdinToDocker(t *testing.T) {
	dir := fakeDocker(t)
	dockerfile := "FROM busybox:1.36\n"

	root := &cli.Command{
		Name:      "inngest",
		Reader:    strings.NewReader(dockerfile),
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Commands:  []*cli.Command{Command()},
	}
	err := root.Run(context.Background(), []string{"inngest", "image", "build", "-t", "app", ".", "--", "-f", "-"})
	require.NoError(t, err)

	args, err := os.ReadFile(filepath.Join(dir, "args"))
	require.NoError(t, err)
	require.Equal(t, []string{"build", "--platform", "linux/amd64", "--load", "-t", "app:latest", ".", "-f", "-"}, strings.Fields(string(args)))

	stdin, err := os.ReadFile(filepath.Join(dir, "stdin"))
	require.NoError(t, err)
	require.Equal(t, dockerfile, string(stdin))
}
