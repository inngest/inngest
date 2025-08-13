package connect

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	mathRand "math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	pb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
	grpcLib "google.golang.org/grpc"
)

const (
	GatewayInstrumentInterval = 20 * time.Second
	GatewayGCInterval         = 30 * time.Minute
)

type gatewayOpt func(*connectGatewaySvc)

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

type ConnectEntitlementRetriever interface {
	AppsPerConnection(ctx context.Context, accountId uuid.UUID) (int, error)
}

type connectGatewaySvc struct {
	pb.ConnectGatewayServer

	gatewayPublicPort int

	gatewayRoutes  chi.Router
	maintenanceApi chi.Router

	grpcServer    *grpcLib.Server
	wsConnections sync.Map

	// gatewayId is a unique identifier, generated each time the service is started.
	// This should be used to uniquely identify the gateway instance when sending messages and routing requests.
	gatewayId ulid.ULID
	dev       bool

	logger logger.Logger

	runCtx context.Context

	auther       auth.Handler
	stateManager state.StateManager
	apiBaseUrl   string

	consecutiveWorkerHeartbeatMissesBeforeConnectionClose int
	workerHeartbeatInterval                               time.Duration
	workerRequestExtendLeaseInterval                      time.Duration
	workerRequestLeaseDuration                            time.Duration

	hostname  string
	ipAddress net.IP

	// groupName specifies the name of the deployment group in case this gateway is one of many replicas.
	groupName string

	lifecycles []ConnectGatewayLifecycleListener

	isDraining      atomic.Bool
	connectionCount connectionCounter
	drainListener   *drainListener
	stateUpdateLock sync.Mutex

	grpcClientManager *grpc.GRPCClientManager[pb.ConnectExecutorClient]
}

func (c *connectGatewaySvc) MaintenanceAPI() http.Handler {
	return c.maintenanceApi
}

func (c *connectGatewaySvc) IsDraining() bool {
	return c.isDraining.Load()
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
		svc.isDraining.Store(isDraining)
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

func WithWorkerHeartbeatInterval(interval time.Duration) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.workerHeartbeatInterval = interval
	}
}

func WithWorkerExtendLeaseInterval(interval time.Duration) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.workerRequestExtendLeaseInterval = interval
	}
}

func WithWorkerRequestLeaseDuration(duration time.Duration) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.workerRequestLeaseDuration = duration
	}
}

func WithConsecutiveWorkerHeartbeatMissesBeforeConnectionClose(misses int) gatewayOpt {
	return func(svc *connectGatewaySvc) {
		svc.consecutiveWorkerHeartbeatMissesBeforeConnectionClose = misses
	}
}

func NewConnectGatewayService(opts ...gatewayOpt) *connectGatewaySvc {
	gateway := &connectGatewaySvc{
		gatewayId:         ulid.MustNew(ulid.Now(), rand.Reader),
		lifecycles:        []ConnectGatewayLifecycleListener{},
		drainListener:     newDrainListener(),
		gatewayPublicPort: 8080,

		workerHeartbeatInterval:                               consts.ConnectWorkerHeartbeatInterval,
		workerRequestExtendLeaseInterval:                      consts.ConnectWorkerRequestExtendLeaseInterval,
		workerRequestLeaseDuration:                            consts.ConnectWorkerRequestLeaseDuration,
		consecutiveWorkerHeartbeatMissesBeforeConnectionClose: 5,

		grpcServer: grpcLib.NewServer(),
	}

	for _, opt := range opts {
		opt(gateway)
	}

	readinessHandler := func(writer http.ResponseWriter, request *http.Request) {
		if gateway.isDraining.Load() {
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

	c.ipAddress = connectConfig.Gateway(ctx).GRPCIP

	if err := c.updateGatewayState(state.GatewayStatusStarting); err != nil {
		return fmt.Errorf("could not set initial gateway state: %w", err)
	}

	c.grpcClientManager = grpc.NewGRPCClientManager(pb.NewConnectExecutorClient, grpc.WithLogger[pb.ConnectExecutorClient](c.logger))

	// Register gRPC server
	pb.RegisterConnectGatewayServer(c.grpcServer, c)

	return nil
}

func (c *connectGatewaySvc) heartbeat(ctx context.Context) {
	heartbeatTicker := time.NewTicker(consts.ConnectGatewayHeartbeatInterval)
	defer heartbeatTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			status := state.GatewayStatusActive
			if c.isDraining.Load() {
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

		if c.isDraining.Load() {
			metrics.GaugeConnectDrainingGateway(ctx, 1, metrics.GaugeOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
			metrics.GaugeConnectActiveGateway(ctx, 0, metrics.GaugeOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
		} else {
			metrics.GaugeConnectActiveGateway(ctx, 1, metrics.GaugeOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
			metrics.GaugeConnectDrainingGateway(ctx, 0, metrics.GaugeOpt{
				PkgName: pkgName,
				Tags:    additionalTags,
			})
		}
	}
}

func (c *connectGatewaySvc) gc(ctx context.Context) {
	for {
		jitter := time.Minute * time.Duration(mathRand.Intn(30))
		periodWithJitter := GatewayGCInterval + jitter

		select {
		case <-ctx.Done():
			return
		case <-time.After(periodWithJitter):
		}

		{
			deleted, err := c.stateManager.GarbageCollectConnections(ctx)
			if err != nil {
				logger.StdlibLogger(ctx).Error("failed to garbage collect connections", "err", err)
			}
			logger.StdlibLogger(ctx).Debug("garbage-collected connections", "deleted", deleted)
		}

		{
			deleted, err := c.stateManager.GarbageCollectGateways(ctx)
			if err != nil {
				logger.StdlibLogger(ctx).Error("failed to garbage collect gateways", "err", err)
			}
			logger.StdlibLogger(ctx).Debug("garbage-collected gateways", "deleted", deleted)
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

	eg := errgroup.Group{}

	eg.Go(func() error {
		c.logger.Info(fmt.Sprintf("starting gateway api at %s", addr))
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	})

	connectConfig := connectConfig.Gateway(ctx)

	eg.Go(func() error {
		addr := fmt.Sprintf(":%d", connectConfig.GRPCPort)

		l, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("could not listen for: %w", err)
		}
		logger.StdlibLogger(ctx).Info("starting connect gateway grpc server", "addr", addr)
		return c.grpcServer.Serve(l)
	})

	if !c.isDraining.Load() {
		err := c.updateGatewayState(state.GatewayStatusActive)
		if err != nil {
			return fmt.Errorf("could not update gateway state: %w", err)
		}
	}

	// Periodically report current status
	go c.heartbeat(ctx)

	// Periodically report metrics
	go c.instrument(ctx)

	// Periodically garbage collect old connections
	go c.gc(ctx)

	return eg.Wait()
}

func (c *connectGatewaySvc) updateGatewayState(status state.GatewayStatus) error {
	c.stateUpdateLock.Lock()
	defer c.stateUpdateLock.Unlock()

	err := c.stateManager.UpsertGateway(context.Background(), &state.Gateway{
		Id:                c.gatewayId,
		Status:            status,
		LastHeartbeatAtMS: time.Now().UnixMilli(),
		Hostname:          c.hostname,
		IPAddress:         c.ipAddress,
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
	c.isDraining.Store(true)
	c.drainListener.Notify()
	return nil
}

func (c *connectGatewaySvc) ActivateGateway() error {
	err := c.updateGatewayState(state.GatewayStatusActive)
	if err != nil {
		return fmt.Errorf("could not update gateway state: %w", err)
	}
	c.isDraining.Store(false)
	return nil
}

// getOrCreateGRPCClient gets the executor IP and returns a gRPC client for that executor,
// creating one if it doesn't exist.
func (c *connectGatewaySvc) getOrCreateGRPCClient(ctx context.Context, envID uuid.UUID, requestId string) (pb.ConnectExecutorClient, error) {
	ip, err := c.stateManager.GetExecutorIP(ctx, envID, requestId)
	if err != nil {
		return nil, err
	}
	executorIP := ip.String()

	grpcURL := net.JoinHostPort(executorIP, fmt.Sprintf("%d", connectConfig.Executor(ctx).GRPCPort))

	return c.grpcClientManager.GetOrCreateClient(ctx, executorIP, grpcURL)
}
