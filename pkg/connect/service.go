package connect

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/rs/zerolog"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

const (
	GatewayHeartbeatInterval  = 5 * time.Second
	GatewayInstrumentInterval = 20 * time.Second
	WorkerHeartbeatInterval   = 10 * time.Second
)

type gatewayOpt func(*connectGatewaySvc)

type ConnectAppLoader interface {
	// GetAppByName returns an app by name
	GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*cqrs.App, error)
}

type connectionCounter struct {
	count  uint64
	waiter sync.WaitGroup
}

func (c *connectionCounter) Add() {
	c.waiter.Add(1)
	atomic.AddUint64(&c.count, 1)
}

func (c *connectionCounter) Done() {
	atomic.AddUint64(&c.count, ^uint64(0))
	c.waiter.Done()
}

func (c *connectionCounter) Count() uint64 {
	return atomic.LoadUint64(&c.count)
}

func (c *connectionCounter) Wait() {
	c.waiter.Wait()
}

type connectGatewaySvc struct {
	gatewayPublicPort int

	gatewayRoutes  chi.Router
	maintenanceApi chi.Router

	// gatewayId is a unique identifier, generated each time the service is started.
	// This should be used to uniquely identify the gateway instance when sending messages and routing requests.
	gatewayId ulid.ULID
	dev       bool

	logger    *slog.Logger
	devlogger *zerolog.Logger

	runCtx context.Context

	auther       auth.Handler
	stateManager state.StateManager
	receiver     pubsub.RequestReceiver
	appLoader    ConnectAppLoader
	apiBaseUrl   string

	hostname string

	// groupName specifies the name of the deployment group in case this gateway is one of many replicas.
	groupName string

	lifecycles []ConnectGatewayLifecycleListener

	isDraining      bool
	connectionCount connectionCounter
	drainListener   *drainListener
	stateUpdateLock sync.Mutex
}

func (c *connectGatewaySvc) MaintenanceAPI() http.Handler {
	return c.maintenanceApi
}

func (c *connectGatewaySvc) IsDraining() bool {
	return c.isDraining
}

func (c *connectGatewaySvc) IsDrained() bool {
	return c.connectionCount.Count() == 0
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

func WithGatewayAuthHandler(auth auth.Handler) gatewayOpt {
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

func WithStartAsDraining(isDraining bool) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.isDraining = isDraining
	}
}

func WithGatewayPublicPort(port int) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.gatewayPublicPort = port
	}
}

func WithApiBaseUrl(url string) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.apiBaseUrl = url
	}
}

func WithGroupName(groupName string) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.groupName = groupName
	}
}

func NewConnectGatewayService(opts ...gatewayOpt) *connectGatewaySvc {
	gateway := &connectGatewaySvc{
		gatewayId:         ulid.MustNew(ulid.Now(), rand.Reader),
		lifecycles:        []ConnectGatewayLifecycleListener{},
		drainListener:     newDrainListener(),
		gatewayPublicPort: 8080,
	}

	for _, opt := range opts {
		opt(gateway)
	}

	readinessHandler := func(writer http.ResponseWriter, request *http.Request) {
		if gateway.isDraining {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}

	gateway.gatewayRoutes = chi.NewRouter().Group(func(r chi.Router) {
		// This is the v0 gateway connect API, which exposes the connect WebSocket endpoint handler
		v0Router := chi.NewRouter()

		// WebSocket endpoint
		v0Router.Handle("/connect", gateway.Handler())

		// Debug endpoint
		v0Router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("."))
		})

		r.Mount("/v0", v0Router)

		// Readiness must be served to traffic port for load balancer health checks
		r.Get("/ready", readinessHandler)
	})

	gateway.maintenanceApi = newMaintenanceApi(gateway)
	gateway.maintenanceApi.Get("/ready", readinessHandler)

	return gateway
}

func (c *connectGatewaySvc) Name() string {
	return "connect-gateway"
}

func (c *connectGatewaySvc) Pre(ctx context.Context) error {
	// Set up gateway-specific logger with info for correlations
	c.logger = logger.StdlibLogger(ctx).With("gateway_id", c.gatewayId)
	if c.dev {
		// Initialize prettier logger for dev server
		c.devlogger = logger.From(ctx)

		// Hide verbose connect gateway logs in dev server by default
		if os.Getenv("CONNECT_GATEWAY_FULL_LOGS") != "true" {
			c.logger = logger.VoidLogger()
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("could not get hostname: %w", err)
	}
	c.hostname = hostname

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

func (c *connectGatewaySvc) metricsTags() map[string]any {
	additionalTags := map[string]any{
		"gateway_id": c.gatewayId,
	}
	if c.groupName != "" {
		additionalTags["group_name"] = c.groupName
	}

	return additionalTags
}

func (c *connectGatewaySvc) instrument(ctx context.Context) {
	instrumentTicker := time.NewTicker(GatewayInstrumentInterval)
	defer instrumentTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-instrumentTicker.C:
		}

		additionalTags := c.metricsTags()

		metrics.GaugeConnectGatewayActiveConnections(ctx, int64(c.connectionCount.Count()), metrics.GaugeOpt{
			PkgName: pkgName,
			Tags:    additionalTags,
		})

		if c.isDraining {
			metrics.GaugeConnectDrainingGateway(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
			metrics.GaugeConnectActiveGateway(ctx, 0, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
		} else {
			metrics.GaugeConnectActiveGateway(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
			metrics.GaugeConnectDrainingGateway(ctx, 0, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
		}
	}
}

func (c *connectGatewaySvc) Run(ctx context.Context) error {
	c.runCtx = ctx

	addr := fmt.Sprintf(":%d", c.gatewayPublicPort)
	server := &http.Server{
		Addr:    addr,
		Handler: c.gatewayRoutes,
	}

	go func() {
		<-ctx.Done()

		c.logger.Info("shutting down gateway")

		err := c.DrainGateway()
		if err != nil {
			c.logger.Error(fmt.Sprintf("could not start draining gateway: %v", err))
		}

		c.logger.Info("waiting for connections to drain")
		c.connectionCount.Wait()
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

	if !c.isDraining {
		err := c.updateGatewayState(state.GatewayStatusActive)
		if err != nil {
			return fmt.Errorf("could not update gateway state: %w", err)
		}
	}

	// Periodically report current status
	go c.heartbeat(ctx)

	// Periodically report metrics
	go c.instrument(ctx)

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
