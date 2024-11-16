package connect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

type gatewayOpt func(*connectGatewaySvc)

type AuthResponse struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

type GatewayAuthHandler func(context.Context, *connect.WorkerConnectRequestData) (*AuthResponse, error)

type connectGatewaySvc struct {
	chi.Router

	// gatewayId is a unique identifier, generated each time the service is started.
	// This should be used to uniquely identify the gateway instance when sending messages and routing requests.
	gatewayId string
	dev       bool

	logger *slog.Logger

	runCtx context.Context

	auther       GatewayAuthHandler
	stateManager state.StateManager
	receiver     pubsub.RequestReceiver
	dbcqrs       cqrs.Manager

	lifecycles []ConnectGatewayLifecycleListener
}

func WithGatewayAuthHandler(auth GatewayAuthHandler) gatewayOpt {
	return func(c *connectGatewaySvc) {
		c.auther = auth
	}
}

func WithConnectionStateManager(m state.StateManager) gatewayOpt {
	return func(c *connectGatewaySvc) {
		c.stateManager = m
	}
}

func WithRequestReceiver(r pubsub.RequestReceiver) gatewayOpt {
	return func(c *connectGatewaySvc) {
		c.receiver = r
	}
}

func WithDB(m cqrs.Manager) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.dbcqrs = m
	}
}

func WithLifeCycles(lifecycles []ConnectGatewayLifecycleListener) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.lifecycles = lifecycles
	}
}

func WithDev() gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.dev = true
	}
}

func NewConnectGatewayService(opts ...gatewayOpt) ([]service.Service, http.Handler) {
	gateway := &connectGatewaySvc{
		Router:     chi.NewRouter(),
		gatewayId:  ulid.MustNew(ulid.Now(), nil).String(),
		lifecycles: []ConnectGatewayLifecycleListener{},
	}
	if os.Getenv("CONNECT_TEST_GATEWAY_ID") != "" {
		gateway.gatewayId = os.Getenv("CONNECT_TEST_GATEWAY_ID")
	}

	for _, opt := range opts {
		opt(gateway)
	}

	router := newConnectRouter(gateway.stateManager, gateway.receiver, gateway.dbcqrs)

	return []service.Service{gateway, router}, gateway.Handler()
}

func (c *connectGatewaySvc) Name() string {
	return "connect-gateway"
}

func (c *connectGatewaySvc) Pre(ctx context.Context) error {
	// Set up gateway-specific logger with info for correlations
	c.logger = logger.StdlibLogger(ctx).With("gateway_id", c.gatewayId)

	// Setup REST endpoint
	c.Use(
		middleware.Heartbeat("/health"),
	)
	c.Route("/v0", func(r chi.Router) {
		r.Use(headers.ContentTypeJsonResponse())

		r.Get("/envs/{envID}/conns", c.showConnectionsByEnv)
		r.Get("/apps/{appID}/conns", c.showConnectionsByApp)
	})

	return nil
}

func (c *connectGatewaySvc) Run(ctx context.Context) error {
	c.runCtx = ctx

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		port := 8289
		if v, err := strconv.Atoi(os.Getenv("CONNECT_GATEWAY_API_PORT")); err == nil && v > 0 {
			port = v
		}
		addr := fmt.Sprintf(":%d", port)
		server := &http.Server{
			Addr:    addr,
			Handler: c,
		}
		c.logger.Info(fmt.Sprintf("starting gateway api at %s", addr))
		return server.ListenAndServe()
	})

	eg.Go(func() error {
		// TODO Mark gateway as active a couple seconds into the future (how do we make sure PubSub is connected and ready to receive?)
		// Start listening for messages, this will block until the context is cancelled
		err := c.receiver.Wait(ctx)
		if err != nil {
			// TODO Should we retry? Exit here? This will interrupt existing connections!
			return fmt.Errorf("could not listen for pubsub messages: %w", err)
		}

		return nil
	})

	return eg.Wait()
}

func (c *connectGatewaySvc) Stop(ctx context.Context) error {
	// TODO Mark gateway as inactive, stop receiving requests

	// TODO Drain connections!

	return nil
}

type connectRouterSvc struct {
	logger *slog.Logger

	stateManager state.StateManager
	receiver     pubsub.RequestReceiver
	dbcqrs       cqrs.Manager
}

func (c *connectRouterSvc) Name() string {
	return "connect-router"
}

func (c *connectRouterSvc) Pre(ctx context.Context) error {
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

			// TODO Should the router ack or the gateway itself?

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

			// TODO Resolve gateway
			gatewayId := ""
			if os.Getenv("CONNECT_TEST_GATEWAY_ID") != "" {
				gatewayId = os.Getenv("CONNECT_TEST_GATEWAY_ID")
			}

			// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
			// We'll potentially lose data here

			// Forward message to the gateway
			err = c.receiver.RouteExecutorRequest(ctx, gatewayId, appId, data)
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
