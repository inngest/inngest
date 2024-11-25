package connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"gonum.org/v1/gonum/stat/sampleuv"
	"log/slog"
	"time"
)

type connectRouterSvc struct {
	logger *slog.Logger

	stateManager state.StateManager
	receiver     pubsub.RequestReceiver

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
			log := c.logger.With("env_id", data.EnvId, "app_id", data.AppId, "req_id", data.RequestId)

			appId, err := uuid.Parse(data.AppId)
			if err != nil {
				log.Error("could not parse app ID")
				return
			}

			envId, err := uuid.Parse(data.EnvId)
			if err != nil {
				log.Error("could not parse env ID")
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

				log.Error("could not store idempotency key", "err", err)
				return
			}

			// Now we're guaranteed to be the exclusive connection processing this message!

			routeTo, err := c.getSuitableConnection(ctx, envId, appId, data.FunctionSlug)
			if err != nil {
				log.Error("could not retrieve suitable connection", "err", err)
				return
			}

			if routeTo == nil {
				log.Warn("no healthy connections")
				return
			}

			log = log.With("gateway_id", routeTo.GatewayId, "conn_id", routeTo.Id, "group_hash", routeTo.GroupId)

			// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
			// We'll potentially lose data here

			// Forward message to the gateway
			err = c.receiver.RouteExecutorRequest(ctx, routeTo.GatewayId, appId, routeTo.Id, data)
			if err != nil {
				// TODO Should we retry? Log error?
				log.Error("failed to route request to gateway", "err", err)
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

func (c *connectRouterSvc) getSuitableConnection(ctx context.Context, envId uuid.UUID, appId uuid.UUID, fnSlug string) (*connect.ConnMetadata, error) {
	conns, err := c.stateManager.GetConnectionsByAppID(ctx, envId, appId)
	if err != nil {
		return nil, fmt.Errorf("could not get connections by app ID: %w", err)
	}

	if len(conns) == 0 {
		return nil, nil
	}

	healthy := make([]*connect.ConnMetadata, 0, len(conns))
	for _, conn := range conns {
		isHealthy, err := c.isHealthy(ctx, envId, appId, fnSlug, conn)
		if err != nil {
			return nil, err
		}

		if isHealthy {
			healthy = append(healthy, conn)
		}
	}

	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy connections found")
	}

	weights := make([]float64, len(healthy))
	for i := range healthy {
		// TODO Calculate weights based on factors like version
		weights[i] = float64(10)
	}

	w := sampleuv.NewWeighted(weights, c.rnd)
	idx, ok := w.Take()
	if !ok {
		return nil, util.ErrWeightedSampleRead
	}
	chosen := healthy[idx]

	return chosen, nil
}

func (c *connectRouterSvc) isHealthy(ctx context.Context, envId uuid.UUID, appId uuid.UUID, fnSlug string, conn *connect.ConnMetadata) (isHealthy bool, err error) {
	defer func() {
		if isHealthy {
			return
		}

		// Clean up unhealthy connection
		err = c.stateManager.DeleteConnection(context.Background(), envId, &appId, conn.GroupId, conn.Id)
		if err != nil {
			c.logger.Error("could not clean up inactive connection", "conn_id", conn.Id, "err", err)
		}
	}()

	if conn.Status != connect.ConnectionStatus_READY {
		return
	}

	if conn.LastHeartbeatAt.AsTime().Before(time.Now().Add(-2 * WorkerHeartbeatInterval)) {
		return
	}

	group, err := c.stateManager.GetWorkerGroupByHash(ctx, envId, conn.GroupId)
	if err != nil {
		return
	}

	var hasFn bool
	for _, slug := range group.FunctionSlugs {
		if slug == fnSlug {
			hasFn = true
			break
		}
	}

	if !hasFn {
		return
	}

	// Ensure gateway is healthy
	gw, err := c.stateManager.GetGateway(ctx, conn.GatewayId)
	if err != nil {
		return
	}

	if gw.Status != state.GatewayStatusActive || gw.LastHeartbeatAt.Before(time.Now().Add(-2*GatewayHeartbeatInterval)) {
		// Clean up unhealthy gateway
		err = c.stateManager.DeleteGateway(ctx, conn.GatewayId)
		if err != nil {
			c.logger.Error("could not clean up inactive gateway", "gateway_id", conn.GatewayId, "err", err)
		}

		return
	}

	isHealthy = true
	return
}

func (c *connectRouterSvc) Stop(ctx context.Context) error {
	return nil
}

func newConnectRouter(stateManager state.StateManager, receiver pubsub.RequestReceiver) *connectRouterSvc {
	return &connectRouterSvc{
		stateManager: stateManager,
		receiver:     receiver,
	}
}
