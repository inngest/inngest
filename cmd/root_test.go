package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestConfigureLogHandler(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		stdoutIsTerminal bool
		initialHandler   string
		wantHandler      string
	}{
		{
			name:             "terminal uses existing handler by default",
			stdoutIsTerminal: true,
			initialHandler:   "text",
			wantHandler:      "text",
		},
		{
			name:             "non-terminal defaults to JSON",
			stdoutIsTerminal: false,
			initialHandler:   "dev",
			wantHandler:      "json",
		},
		{
			name:             "explicit true enables JSON in a terminal",
			args:             []string{"--json=true"},
			stdoutIsTerminal: true,
			initialHandler:   "dev",
			wantHandler:      "json",
		},
		{
			name:             "explicit true enables JSON outside a terminal",
			args:             []string{"--json"},
			stdoutIsTerminal: false,
			initialHandler:   "text",
			wantHandler:      "json",
		},
		{
			name:             "explicit false disables JSON in a terminal",
			args:             []string{"--json=false"},
			stdoutIsTerminal: true,
			initialHandler:   "json",
			wantHandler:      "dev",
		},
		{
			name:             "explicit false disables JSON outside a terminal",
			args:             []string{"--json=false"},
			stdoutIsTerminal: false,
			initialHandler:   "json",
			wantHandler:      "dev",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LOG_HANDLER", test.initialHandler)

			var got string
			captureHandler := func(context.Context, *cli.Command) error {
				got = os.Getenv("LOG_HANDLER")
				return nil
			}
			app := &cli.Command{
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json"},
				},
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					configureLogHandler(cmd, test.stdoutIsTerminal)
					return ctx, nil
				},
				Commands: []*cli.Command{
					{
						Name:   "dev",
						Action: captureHandler,
					},
				},
			}

			args := append([]string{"inngest"}, test.args...)
			err := app.Run(t.Context(), append(args, "dev"))
			require.NoError(t, err)
			require.Equal(t, test.wantHandler, got)
		})
	}
}
