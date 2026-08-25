package authcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestJSONStatusIsOneLineAndFailsWhenLoggedOut(t *testing.T) {
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	output := bytes.Buffer{}
	command := &cli.Command{
		Name:   "inngest",
		Writer: &output,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json"},
		},
		Commands: []*cli.Command{AuthCommand()},
	}

	err := command.Run(context.Background(), []string{"inngest", "--json", "auth", "status"})
	var reported *ReportedError
	require.True(t, errors.As(err, &reported))

	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)
	result := map[string]any{}
	require.NoError(t, json.Unmarshal(lines[0], &result))
	require.Equal(t, "auth_status", result["type"])
	require.Equal(t, false, result["authenticated"])
}
