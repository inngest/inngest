package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/inngest/inngest/cmd/devserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestDevHelpTelemetry(t *testing.T) {
	type sent struct {
		command string
		context map[string]any
	}
	var got []sent
	orig := sendHelpEvent
	t.Cleanup(func() { sendHelpEvent = orig })
	sendHelpEvent = func(_ context.Context, command string, eventContext map[string]any) {
		got = append(got, sent{command, eventContext})
	}

	want := []sent{{"inngest dev", map[string]any{"flags": map[string]bool{"help": true}}}}

	for _, args := range [][]string{
		{"inngest", "dev", "--help"},
		{"inngest", "dev", "-h"},
		{"inngest", "dev", "help"},
		{"inngest", "help", "dev"},
	} {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			got = nil
			app := &cli.Command{
				Name:     "inngest",
				Writer:   io.Discard,
				Commands: []*cli.Command{devserver.Command()},
			}
			require.NoError(t, app.Run(context.Background(), args))
			assert.Equal(t, want, got)
		})
	}

	t.Run("other help", func(t *testing.T) {
		got = nil
		app := &cli.Command{
			Name:     "inngest",
			Writer:   io.Discard,
			Commands: []*cli.Command{devserver.Command()},
		}
		require.NoError(t, app.Run(context.Background(), []string{"inngest", "--help"}))
		assert.Empty(t, got)
	})
}
