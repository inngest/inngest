package connect

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	apiv0 "github.com/inngest/inngest/pkg/connect/rest/v0"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
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

func NewConnectGatewayService(opts ...gatewayOpt) (*connectGatewaySvc, *connectRouterSvc, http.Handler) {
	gateway := &connectGatewaySvc{
		Router:     chi.NewRouter(),
		gatewayId:  ulid.MustNew(ulid.Now(), rand.Reader).String(),
		lifecycles: []ConnectGatewayLifecycleListener{},
	}

	for _, opt := range opts {
		opt(gateway)
	}

	router := newConnectRouter(gateway.stateManager, gateway.receiver, gateway.dbcqrs)

	return gateway, router, gateway.Handler()
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
	c.Mount("/v0", apiv0.New(c, apiv0.Opts{
		ConnectManager: c.stateManager,
		GroupManager:   c.stateManager,
		Dev:            c.dev,
	}))

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
