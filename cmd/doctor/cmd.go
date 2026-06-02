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
		// cli auto-injects a "help" subcommand into cmd.Commands during
		// setupDefaults; it's not a health check, and invoking its Action
		// (helpCommandAction) against an unparsed command panics in
		// cmd.Args().First().
		if sub.Name == "help" {
			continue
		}
		fmt.Printf("• %s ... ", sub.Name)
		if err := runSubCheck(ctx, sub); err != nil {
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

// runSubCheck invokes a subcommand's Action with its flag values populated
// from defaults and Sources (env vars, files). We can't go through sub.Run:
// it routes ExitCoder errors (cli.Exit) through OsExiter = os.Exit, which
// would kill the loop on the first failing check. Call Action directly, but
// drive the same Pre/PostParse phases sub.Run would, so env-var Sources
// resolve.
func runSubCheck(ctx context.Context, sub *cli.Command) error {
	if sub.Action == nil {
		return nil
	}
	for _, f := range sub.Flags {
		if err := f.PreParse(); err != nil {
			return err
		}
	}
	for _, f := range sub.Flags {
		if err := f.PostParse(); err != nil {
			return err
		}
	}
	return sub.Action(ctx, sub)
}
