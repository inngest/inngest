package debug

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/logger"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:    "debug",
		Aliases: []string{"dbg"},
		Usage:   "Debug commands",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Aliases: []string{"addr"},
				Value:   "localhost:7777",
				Usage:   "gRPC address of the debug API server",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			addr := cmd.String("addr")

			log := logger.StdlibLogger(ctx)
			log.Debug("connecting to debug API", "addr", addr)

			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return ctx, fmt.Errorf("failed to connect to debug API at %s: %w", addr, err)
			}

			client := dbgpb.NewDebugClient(conn)

			log.Debug("successfully connected to debug API", "addr", addr)

			debugCtx := &DebugContext{
				Client: client,
				Conn:   conn,
			}

			return context.WithValue(ctx, DbgCtxKey, debugCtx), nil
		},
		After: func(ctx context.Context, cmd *cli.Command) error {
			if debugCtx := ctx.Value(DbgCtxKey); debugCtx != nil {
				if dc, ok := debugCtx.(*DebugContext); ok && dc.Conn != nil {
					return dc.Conn.Close()
				}
			}
			return nil
		},
		Commands: []*cli.Command{
			queueCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return fmt.Errorf("debug commands not yet implemented - use subcommands")
		},
	}
}
