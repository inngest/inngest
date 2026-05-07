package doctor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/inngest/inngest/cmd/doctor/healthcheck"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:    "doctor",
		Aliases: []string{"dr"},
		Usage:   "Run diagnostics against the local inngest environment and server",
		Commands: []*cli.Command{
			healthcheck.Command(),
		},
		Action: runAllChecks,
	}
}

func runAllChecks(ctx context.Context, cmd *cli.Command) error {
	var failed []string
	for _, sub := range cmd.Commands {
		fmt.Printf("• %s ... ", sub.Name)
		if err := sub.Action(ctx, sub); err != nil {
			fmt.Println("FAIL")
			if msg := err.Error(); msg != "" {
				fmt.Fprintf(os.Stderr, "  %s\n", msg)
			}
			failed = append(failed, sub.Name)
			continue
		}
		fmt.Println("OK")
	}
	if len(failed) > 0 {
		return cli.Exit(fmt.Sprintf("%d check(s) failed: %s", len(failed), strings.Join(failed, ", ")), 1)
	}
	return nil
}
