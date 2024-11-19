package connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"gonum.org/v1/gonum/stat/sampleuv"
	"log/slog"
	"os"
)

type connectRouterSvc struct {
	logger *slog.Logger

	stateManager state.StateManager
	receiver     pubsub.RequestReceiver
	dbcqrs       cqrs.Manager

	rnd *util.FrandRNG
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
	go func() {
		err := c.receiver.ReceiveExecutorMessages(ctx, func(rawBytes []byte, data *connect.GatewayExecutorRequestData) {
			log := c.logger.With("app_id", data.AppId, "req_id", data.RequestId)

			appId, err := uuid.Parse(data.AppId)
			if err != nil {
				log.Error("could not parse app ID")
				return
			}

			log.Debug("router received msg")

			err = c.receiver.AckMessage(ctx, appId, data.RequestId, pubsub.AckSourceRouter)
			if err != nil {
				log.Error("failed to ack message", "err", err)
				// TODO Log error, retry?
				return
			}

			// We need to add an idempotency key to ensure only one router instance processes the message
			err = c.stateManager.SetRequestIdempotency(ctx, appId, data.RequestId)
			if err != nil {
				if errors.Is(err, state.ErrIdempotencyKeyExists) {
					// Another connection was faster than us, we can ignore this message
					return
				}

				// TODO Log error
				return
			}

			// Now we're guaranteed to be the exclusive connection processing this message!

			c.stateManager.GetConnectionsByAppID(ctx, appId)

			// TODO Resolve gateway
			gatewayId := ""
			if os.Getenv("CONNECT_TEST_GATEWAY_ID") != "" {
				gatewayId = os.Getenv("CONNECT_TEST_GATEWAY_ID")
			}

			// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
			// We'll potentially lose data here

			connId := ""

			// Forward message to the gateway
			err = c.receiver.RouteExecutorRequest(ctx, gatewayId, appId, connId, data)
			if err != nil {
				// TODO Should we retry? Log error?
				log.Error("failed to route request to gateway", "err", err, "gateway_id", gatewayId)
				return
			}
		})
		if err != nil {
			// TODO Log error, retry?
			return
		}
	}()

	// TODO Periodically ping random gateways via PubSub and only consider them active if they respond in time -> Multiple routers will do this

	err := c.receiver.Wait(ctx)
	if err != nil {
		return fmt.Errorf("could not listen for pubsub messages: %w", err)
	}

	return nil

}

func (c *connectRouterSvc) getSuitableConnection(ctx context.Context, appId uuid.UUID, fnSlug string) (*connect.ConnMetadata, error) {
	conns, err := c.stateManager.GetConnectionsByAppID(ctx, appId)
	if err != nil {
		return nil, fmt.Errorf("could not get connections by app ID: %w", err)
	}

	healthy := make([]*connect.ConnMetadata, 0, len(conns))
	for i := range conns {
		if conns[i].Status == connect.ConnectionStatus_READY {
			healthy = append(healthy, conns[i])
		}
	}

	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy connections found")
	}

	weights := make([]float64, len(healthy))
	for i, conn := range healthy {
		weights[i] = 1.0
	}

	w := sampleuv.NewWeighted(weights, c.rnd)
	result := make([]*connect.ConnMetadata, len(weights))
	for n := range result {
		idx, ok := w.Take()
		if !ok && len(result) < len(weights)-1 {
			return result, util.ErrWeightedSampleRead
		}
		result[n] = shuffleIdx[idx]
	}

}

func (c *connectRouterSvc) Stop(ctx context.Context) error {
	return nil
}

func newConnectRouter(stateManager state.StateManager, receiver pubsub.RequestReceiver, db cqrs.Manager) service.Service {
	return &connectRouterSvc{
		stateManager: stateManager,
		receiver:     receiver,
		dbcqrs:       db,
	}
}
