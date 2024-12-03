package connect

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

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

const (
	GatewayHeartbeatInterval = 5 * time.Second
	WorkerHeartbeatInterval  = 10 * time.Second
)

type gatewayOpt func(*connectGatewaySvc)

type AuthResponse struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

type GatewayAuthHandler func(context.Context, *connect.WorkerConnectRequestData) (*AuthResponse, error)

type ConnectAppLoader interface {
	// GetAppByName returns an app by name
	GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*cqrs.App, error)
}

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
	appLoader    ConnectAppLoader

	hostname string

	lifecycles []ConnectGatewayLifecycleListener

	isDraining      bool
	connectionSema  sync.WaitGroup
	drainListener   *drainListener
	stateUpdateLock sync.Mutex
}

type drainListener struct {
	subscribers map[ulid.ULID]func()
	lock        sync.Mutex
}

func newDrainListener() *drainListener {
	return &drainListener{
		subscribers: make(map[ulid.ULID]func()),
	}
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

func WithAppLoader(l ConnectAppLoader) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.appLoader = l
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
		Router:        chi.NewRouter(),
		gatewayId:     ulid.MustNew(ulid.Now(), rand.Reader).String(),
		lifecycles:    []ConnectGatewayLifecycleListener{},
		drainListener: newDrainListener(),
	}

	for _, opt := range opts {
		opt(gateway)
	}

	router := newConnectRouter(gateway.stateManager, gateway.receiver)

	return gateway, router, gateway.Handler()
}

func (c *connectGatewaySvc) Name() string {
	return "connect-gateway"
}

func (c *connectGatewaySvc) Pre(ctx context.Context) error {
	// Set up gateway-specific logger with info for correlations
	c.logger = logger.StdlibLogger(ctx).With("gateway_id", c.gatewayId)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("could not get hostname: %w", err)
	}
	c.hostname = hostname

	// Setup REST endpoint
	c.Use(
		middleware.Heartbeat("/health"),
	)
	c.Mount("/v0", apiv0.New(c, apiv0.Opts{
		ConnectManager:     c.stateManager,
		GroupManager:       c.stateManager,
		Dev:                c.dev,
		GatewayMaintenance: c,
	}))

	if err := c.updateGatewayState(state.GatewayStatusStarting); err != nil {
		return fmt.Errorf("could not set initial gateway state: %w", err)
	}

	return nil
}

func (c *connectGatewaySvc) heartbeat(ctx context.Context) {
	heartbeatTicker := time.NewTicker(GatewayHeartbeatInterval)
	defer heartbeatTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			status := state.GatewayStatusActive
			if c.isDraining {
				status = state.GatewayStatusDraining
			}

			err := c.updateGatewayState(status)
			if err != nil {
				c.logger.Error(fmt.Sprintf("could not update gateway state: %v", err))
			}
		}
	}
}

func (c *connectGatewaySvc) Run(ctx context.Context) error {
	c.runCtx = ctx

	port := 8289
	if v, err := strconv.Atoi(os.Getenv("CONNECT_GATEWAY_API_PORT")); err == nil && v > 0 {
		port = v
	}
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: c,
	}

	go func() {
		<-ctx.Done()

		c.logger.Info("shutting down gateway")

		err := c.DrainGateway()
		if err != nil {
			c.logger.Error(fmt.Sprintf("could not start draining gateway: %v", err))
		}

		c.logger.Info("waiting for connections to drain")
		c.connectionSema.Wait()
		c.logger.Info("shutting down gateway api")
		_ = server.Shutdown(ctx)
	}()

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
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

	err := c.updateGatewayState(state.GatewayStatusActive)
	if err != nil {
		return fmt.Errorf("could not update gateway state: %w", err)
	}

	// Periodically report current status
	go c.heartbeat(ctx)

	return eg.Wait()
}

func (c *connectGatewaySvc) updateGatewayState(status state.GatewayStatus) error {
	c.stateUpdateLock.Lock()
	defer c.stateUpdateLock.Unlock()

	err := c.stateManager.UpsertGateway(context.Background(), &state.Gateway{
		Id:              c.gatewayId,
		Status:          status,
		LastHeartbeatAt: time.Now(),
		Hostname:        c.hostname,
	})
	if err != nil {
		c.logger.Error("failed to update gateway status in state", "status", status, "error", err)

		return fmt.Errorf("could not upsert gateway: %w", err)
	}

	return nil
}

func (c *connectGatewaySvc) Stop(ctx context.Context) error {
	// Clean up gateway
	err := c.stateManager.DeleteGateway(ctx, c.gatewayId)
	if err != nil {
		return fmt.Errorf("could not delete gateway: %w", err)
	}

	return nil
}

func (c *drainListener) OnDrain(f func()) func() {
	id := ulid.MustNew(ulid.Now(), rand.Reader)

	c.lock.Lock()
	defer c.lock.Unlock()
	c.subscribers[id] = f

	return func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		delete(c.subscribers, id)
	}
}

func (c *drainListener) Notify() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, f := range c.subscribers {
		f()
	}
}

func (c *connectGatewaySvc) GetState() (*state.Gateway, error) {
	gatewayState, err := c.stateManager.GetGateway(context.Background(), c.gatewayId)
	if err != nil {
		return nil, fmt.Errorf("could not get gateway state: %w", err)
	}

	return gatewayState, nil
}

func (c *connectGatewaySvc) DrainGateway() error {
	err := c.updateGatewayState(state.GatewayStatusDraining)
	if err != nil {
		return fmt.Errorf("could not update gateway state: %w", err)
	}
	c.isDraining = true
	c.drainListener.Notify()
	return nil
}

func (c *connectGatewaySvc) ActivateGateway() error {
	err := c.updateGatewayState(state.GatewayStatusActive)
	if err != nil {
		return fmt.Errorf("could not update gateway state: %w", err)
	}
	c.isDraining = false
	return nil
}
