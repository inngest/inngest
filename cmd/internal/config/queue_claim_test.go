package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestQueueClaimConcurrencyConfig(t *testing.T) {
	for _, tc := range []struct {
		name      string
		file      string
		env       string
		flag      string
		want      int
		errorText string
	}{
		{name: "default", want: 1},
		{name: "file", file: `{"queue-claim-concurrency":8}`, want: 8},
		{name: "environment overrides file", file: `{"queue-claim-concurrency":2}`, env: "4", want: 4},
		{name: "flag overrides environment", file: `{"queue-claim-concurrency":2}`, env: "4", flag: "8", want: 8},
		{name: "explicit FIFO overrides file", file: `{"queue-claim-concurrency":8}`, flag: "1", want: 1},
		{name: "worker count boundary", flag: "32", want: 32},
		{name: "zero environment", env: "0", errorText: "must be at least 1"},
		{name: "zero file", file: `{"queue-claim-concurrency":0}`, errorText: "must be at least 1"},
		{name: "negative flag", flag: "-1", errorText: "must be at least 1"},
		{name: "above worker count", flag: "33", errorText: "must not exceed queue-workers"},
		{name: "invalid environment", env: "invalid", errorText: "error unmarshaling config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupTest()
			path := filepath.Join(t.TempDir(), "inngest.json")
			data := tc.file
			if data == "" {
				data = "{}"
			}
			require.NoError(t, os.WriteFile(path, []byte(data), 0600))
			if tc.env != "" {
				t.Setenv("INNGEST_QUEUE_CLAIM_CONCURRENCY", tc.env)
			}
			got := 0
			cmd := &cli.Command{
				Flags: []cli.Flag{&cli.IntFlag{Name: "queue-claim-concurrency", Value: 1}, &cli.StringFlag{Name: "config"}},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if err := InitStartConfig(ctx, cmd); err != nil {
						return err
					}
					got = GetIntValue(cmd, "queue-claim-concurrency", 1)
					return ValidateQueueClaimConcurrency(got, 32)
				},
			}
			args := []string{"config-test", "--config", path}
			if tc.flag != "" {
				args = append(args, "--queue-claim-concurrency", tc.flag)
			}
			err := cmd.Run(context.Background(), args)
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
