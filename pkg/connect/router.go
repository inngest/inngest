package connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/routing"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

const (
	pkgNameRouter = "connect.router"
)

type connectRouterSvc struct {
	logger *slog.Logger

	stateManager state.StateManager
	receiver     pubsub.RequestReceiver

	rnd *util.FrandRNG

	tracer trace.ConditionalTracer
}

func (c *connectRouterSvc) Name() string {
	return "connect-router"
}

func (c *connectRouterSvc) Pre(ctx context.Context) error {
	// For weighted shuffles generate a new rand.
	c.rnd = util.NewFrandRNG()

	// Set up router-specific logger with info for correlations
	c.logger = logger.StdlibLogger(ctx)

	return nil
}

func (c *connectRouterSvc) Run(ctx context.Context) error {
	onSubscribed := make(chan struct{})

	go func() {
		err := c.receiver.ReceiveExecutorMessages(ctx, func(_ []byte, data *connect.GatewayExecutorRequestData) {
			log := c.logger.With("env_id", data.EnvId, "app_id", data.AppId, "req_id", data.RequestId, "run_id", data.RunId)

			err := routing.Route(ctx, c.stateManager, c.receiver, c.rnd, c.tracer, log, data)
			if err != nil {
				if errors.Is(err, routing.ErrNoHealthyConnection) {
					err = c.receiver.NackMessage(ctx, data.RequestId, pubsub.AckSourceRouter, syscode.Error{
						Code:    syscode.CodeConnectNoHealthyConnection,
						Message: "Could not find a healthy connection",
					})
					if err != nil {
						log.Error("failed to nack message", "err", err)
					}
					return
				}

				log.Error("failed to route message", "err", err)
			}

			err = c.receiver.AckMessage(ctx, data.RequestId, pubsub.AckSourceRouter)
			if err != nil {
				log.Error("failed to ack message", "err", err)
				// TODO Log error, retry?
				return
			}
		}, onSubscribed)
		if err != nil {
			// TODO Log error, retry?
			return
		}
	}()

	// TODO Periodically ping random gateways via PubSub and only consider them active if they respond in time -> Multiple routers will do this

	eg := errgroup.Group{}
	eg.Go(func() error {
		err := c.receiver.Wait(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("could not listen for pubsub messages: %w", err)
		}
		return nil
	})

	<-onSubscribed

	c.logger.Debug("connect router is ready")

	return eg.Wait()

}

func (c *connectRouterSvc) Stop(ctx context.Context) error {
	return nil
}

func NewConnectMessageRouterService(stateManager state.StateManager, receiver pubsub.RequestReceiver, tracer trace.ConditionalTracer) *connectRouterSvc {
	return &connectRouterSvc{
		stateManager: stateManager,
		receiver:     receiver,
		tracer:       tracer,
	}
}
