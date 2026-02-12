package queue

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	debugpkg "github.com/inngest/inngest/pkg/debug"
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

func backlogSizeCommand() *cli.Command {
	return &cli.Command{
		Name:      "size",
		Usage:     "Get the number of items in a specific backlog",
		ArgsUsage: "<backlog-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("backlog ID is required")
			}

			backlogID := cmd.Args().Get(0)

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
