package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cli/output"
	debugpkg "github.com/inngest/inngest/pkg/debug"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func BacklogCommand() *cli.Command {
	return &cli.Command{
		Name:    "backlog",
		Aliases: []string{"bl"},
		Usage:   "Backlog / Key Queue debugging commands",
		Commands: []*cli.Command{
			backlogShadowPartitionCommand(),
			backlogListCommand(),
			backlogSizeCommand(),
		},
	}
}

func backlogShadowPartitionCommand() *cli.Command {
	return &cli.Command{
		Name:      "shadow",
		Aliases:   []string{"sp"},
		Usage:     "Get shadow partition details for a partition",
		ArgsUsage: "<partition-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("partition ID is required")
			}

			partitionID := cmd.Args().Get(0)

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.GetShadowPartition(ctx, &pb.ShadowPartitionRequest{
				PartitionId: partitionID,
			})
			if err != nil {
				return fmt.Errorf("failed to get shadow partition: %w", err)
			}

			return output.TextShadowPartition(resp)
		},
	}
}

func backlogListCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Aliases:   []string{"ls"},
		Usage:     "List backlogs for a partition",
		ArgsUsage: "<partition-id>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of backlogs to return",
				Value: 100,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("partition ID is required")
			}

			partitionID := cmd.Args().Get(0)

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.GetBacklogs(ctx, &pb.BacklogsRequest{
				PartitionId: partitionID,
				Limit:       int64(cmd.Int("limit")),
			})
			if err != nil {
				return fmt.Errorf("failed to list backlogs: %w", err)
			}

			return output.TextBacklogList(resp)
		},
	}
}

// backlogIDFlags returns the common flags for constructing a backlog ID from its components.
func backlogIDFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "fn",
			Usage: "Function ID (UUID) to construct the backlog ID",
		},
		&cli.BoolFlag{
			Name:  "start",
			Usage: "Whether this is a start backlog",
		},
		&cli.StringFlag{
			Name:  "throttle-expr",
			Usage: "Throttle key expression (e.g. 'event.data.customerId')",
		},
		&cli.StringFlag{
			Name:  "throttle-value",
			Usage: "Evaluated throttle key value (e.g. 'customer-123')",
		},
		&cli.StringFlag{
			Name:  "ck1-expr",
			Usage: "Concurrency key 1: raw expression (e.g. 'event.data.customerId')",
		},
		&cli.StringFlag{
			Name:  "ck1-value",
			Usage: "Concurrency key 1: evaluated value (e.g. 'customer-123')",
		},
		&cli.StringFlag{
			Name:  "ck1-scope",
			Usage: "Concurrency key 1: scope (fn, env, account)",
			Value: "fn",
		},
		&cli.StringFlag{
			Name:  "ck1-scope-id",
			Usage: "Concurrency key 1: scope entity UUID. Defaults to --fn value for fn scope",
		},
		&cli.StringFlag{
			Name:  "ck2-expr",
			Usage: "Concurrency key 2: raw expression",
		},
		&cli.StringFlag{
			Name:  "ck2-value",
			Usage: "Concurrency key 2: evaluated value",
		},
		&cli.StringFlag{
			Name:  "ck2-scope",
			Usage: "Concurrency key 2: scope (fn, env, account)",
			Value: "fn",
		},
		&cli.StringFlag{
			Name:  "ck2-scope-id",
			Usage: "Concurrency key 2: scope entity UUID. Defaults to --fn value for fn scope",
		},
	}
}

// resolveBacklogID returns the backlog ID either from the positional argument or by building it from flags.
func resolveBacklogID(cmd *cli.Command) (string, error) {
	// If a positional argument is provided, use it directly
	if cmd.NArg() > 0 {
		return cmd.Args().Get(0), nil
	}

	// Otherwise, build from flags
	fnStr := cmd.String("fn")
	if fnStr == "" {
		return "", fmt.Errorf("either a backlog ID argument or --fn flag is required")
	}

	fnID, err := uuid.Parse(fnStr)
	if err != nil {
		return "", fmt.Errorf("invalid function ID: %w", err)
	}

	var throttle *osqueue.ThrottleKeyInput
	if cmd.IsSet("throttle-expr") || cmd.IsSet("throttle-value") {
		throttle = &osqueue.ThrottleKeyInput{
			Expression:     cmd.String("throttle-expr"),
			EvaluatedValue: cmd.String("throttle-value"),
		}
	}

	// Throttle only applies to start backlogs, so imply start when throttle is provided
	start := cmd.Bool("start") || throttle != nil

	var concurrencyKeys []osqueue.ConcurrencyKeyInput
	if cmd.IsSet("ck1-value") || cmd.IsSet("ck1-expr") {
		ck, err := parseConcurrencyKey(cmd, "ck1", fnID)
		if err != nil {
			return "", fmt.Errorf("invalid concurrency key 1: %w", err)
		}
		concurrencyKeys = append(concurrencyKeys, ck)
	}
	if cmd.IsSet("ck2-value") || cmd.IsSet("ck2-expr") {
		ck, err := parseConcurrencyKey(cmd, "ck2", fnID)
		if err != nil {
			return "", fmt.Errorf("invalid concurrency key 2: %w", err)
		}
		concurrencyKeys = append(concurrencyKeys, ck)
	}

	return osqueue.BuildBacklogID(fnID, start, throttle, concurrencyKeys), nil
}

func parseConcurrencyKey(cmd *cli.Command, prefix string, fnID uuid.UUID) (osqueue.ConcurrencyKeyInput, error) {
	expr := cmd.String(prefix + "-expr")
	value := cmd.String(prefix + "-value")
	scopeStr := cmd.String(prefix + "-scope")
	scopeIDStr := cmd.String(prefix + "-scope-id")

	scope, err := parseConcurrencyScope(scopeStr)
	if err != nil {
		return osqueue.ConcurrencyKeyInput{}, err
	}

	scopeID := fnID
	if scopeIDStr != "" {
		scopeID, err = uuid.Parse(scopeIDStr)
		if err != nil {
			return osqueue.ConcurrencyKeyInput{}, fmt.Errorf("invalid scope ID: %w", err)
		}
	}

	return osqueue.ConcurrencyKeyInput{
		Expression:     expr,
		EvaluatedValue: value,
		Scope:          scope,
		ScopeID:        scopeID,
	}, nil
}

func parseConcurrencyScope(s string) (enums.ConcurrencyScope, error) {
	switch s {
	case "fn", "function":
		return enums.ConcurrencyScopeFn, nil
	case "env", "environment":
		return enums.ConcurrencyScopeEnv, nil
	case "account", "acct":
		return enums.ConcurrencyScopeAccount, nil
	default:
		return enums.ConcurrencyScopeFn, fmt.Errorf("unknown scope %q, expected: fn, env, account", s)
	}
}

func backlogSizeCommand() *cli.Command {
	return &cli.Command{
		Name:  "size",
		Usage: "Get the number of items in a specific backlog",
		ArgsUsage: "[backlog-id]",
		UsageText: `Pass a backlog ID directly:
   inngest debug queue backlog size <backlog-id>

Or construct one from components:
   inngest debug queue backlog size --fn <function-uuid> --start --ck1-expr "event.data.customerId" --ck1-value "customer-123"`,
		Flags: backlogIDFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			backlogID, err := resolveBacklogID(cmd)
			if err != nil {
				return err
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.GetBacklogSize(ctx, &pb.BacklogSizeRequest{
				BacklogId: backlogID,
			})
			if err != nil {
				return fmt.Errorf("failed to get backlog size: %w", err)
			}

			return output.TextBacklogSize(resp)
		},
	}
}
