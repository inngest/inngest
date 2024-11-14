package connect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

type gatewayOpt func(*connectGatewaySvc)

type AuthResponse struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

type GatewayAuthHandler func(context.Context, *connect.SDKConnectRequestData) (*AuthResponse, error)

type connectGatewaySvc struct {
	chi.Router

	// gatewayId is a unique identifier, generated each time the service is started.
	// This should be used to uniquely identify the gateway instance when sending messages and routing requests.
	gatewayId string
	dev       bool

	logger *slog.Logger

	runCtx context.Context

	auther       GatewayAuthHandler
	stateManager state.ConnectionStateManager
	receiver     pubsub.RequestReceiver
	dbcqrs       cqrs.Manager

	lifecycles []ConnectGatewayLifecycleListener
}

func WithGatewayAuthHandler(auth GatewayAuthHandler) gatewayOpt {
	return func(c *connectGatewaySvc) {
		c.auther = auth
	}
}

func WithConnectionStateManager(m state.ConnectionStateManager) gatewayOpt {
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

func (c *connectGatewaySvc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the context as needed. Use of r.Context() is not recommended
		// to avoid surprising behavior (see http.Hijacker).
		ctx, cancel := context.WithCancel(c.runCtx)
		defer cancel()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{
				types.GatewaySubProtocol,
			},
		})
		if err != nil {
			return
		}
		defer func() {
			c.logger.Debug("Closing WebSocket connection")

			_ = ws.CloseNow()
		}()

		c.logger.Debug("WebSocket connection established, sending hello")

		{
			err = wsproto.Write(ctx, ws, &connect.ConnectMessage{
				Kind: connect.GatewayMessageType_GATEWAY_HELLO,
			})
			if err != nil {
				c.logger.Error("could not send hello", "err", err)

				return
			}
		}

		initialMessageData, serr := c.establishConnection(ctx, ws)
		if serr != nil {
			c.logger.Error("error establishing connection", "error", err)
			_ = ws.Close(serr.StatusCode, serr.Error())

			return
		}
		log := c.logger.With("account_id", initialMessageData.AuthData.AccountId)

		// TODO Check whether SDK group was already synced
		isAlreadySynced := false
		if !isAlreadySynced {
			c.logger.Debug("Sending sync message to SDK")
			data := &connect.GatewaySyncRequestData{
				// TODO Set this to prevent unattached syncs!
				DeployId: nil,
			}
			marshaled, err := proto.Marshal(data)

			if err != nil {
				// TODO Handle this properly
				return
			}
			err = wsproto.Write(ctx, ws, &connect.ConnectMessage{
				Kind:    connect.GatewayMessageType_GATEWAY_REQUEST_SYNC,
				Payload: marshaled,
			})
			if err != nil {
				return
			}
		}

		// wait until app is ready, then fetch details
		// TODO Find better way to load app by name, account for initial register delay
		attempts := 0
		var appId uuid.UUID
		for {
			apps, err := c.dbcqrs.GetAllApps(ctx)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				c.logger.Error("could not get apps", "err", err)
				ws.Close(websocket.StatusInternalError, "Internal error")
				return
			}

			for _, app := range apps {
				if app.Name == initialMessageData.AppName {
					appId = app.ID
				}
			}
			if appId == uuid.Nil {
				if attempts < 10 {
					<-time.After(1 * time.Second)
					attempts++
					continue
				}

				c.logger.Error("could not find app", "appName", initialMessageData.AppName)
				ws.Close(websocket.StatusPolicyViolation, "Could not find app")
				return
			}

			break
		}

		log = log.With("app_id", appId)

		log.Debug("found app, connection is ready")

		// TODO Persist connection state

		// Wait for relevant messages and forward them over the WebSocket connection
		go func() {
			// Receive execution-related messages for the app, forwarded by the router.
			// The router selects only one gateway to handle a request from a pool of one or more workers (and thus WebSockets)
			// running for each app.
			err := c.receiver.ReceiveRoutedRequest(ctx, c.gatewayId, appId, func(rawBytes []byte, data *connect.GatewayExecutorRequestData) {
				log.Debug("gateway received msg", "app_id", appId, "req_id", data.RequestId)

				// This will be sent exactly once, as the router selected this gateway to handle the request
				err = c.receiver.AckMessage(ctx, appId, data.RequestId)
				if err != nil {
					// TODO Log error, retry?
					return
				}

				// Forward message to SDK!
				err = wsproto.Write(ctx, ws, &connect.ConnectMessage{
					Kind:    connect.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST,
					Payload: rawBytes,
				})
				if err != nil {
					// TODO The connection cannot be used, we need to let the executor know!
					return
				}
			})
			if err != nil {
				// TODO Log error, retry?
				return
			}
		}()

		// Run loop
		go func() {
			for {
				if ctx.Err() != nil {
					break
				}

				var msg connect.ConnectMessage
				err = wsproto.Read(ctx, ws, &msg)
				if err != nil {
					return
				}

				log.Debug("received WebSocket message", "kind", msg.Kind.String())

				switch msg.Kind {
				case connect.GatewayMessageType_SDK_REPLY:
					// Handle SDK reply
					err := c.handleSdkReply(ctx, log, appId, &msg)
					if err != nil {
						// TODO Handle error
						continue
					}
				default:
					// TODO Handle proper connection cleanup
					ws.Close(websocket.StatusPolicyViolation, "Invalid message kind")
					return
				}
			}
		}()

		<-ctx.Done()

		ws.Close(websocket.StatusNormalClosure, "")
	})
}

func (c *connectGatewaySvc) establishConnection(ctx context.Context, ws *websocket.Conn) (*connect.SDKConnectRequestData, *SocketError) {
	var (
		initialMessageData connect.SDKConnectRequestData
		initialMessage     connect.ConnectMessage
	)

	shorterContext, cancelShorter := context.WithTimeout(ctx, 5*time.Second)
	defer cancelShorter()
	err := wsproto.Read(shorterContext, ws, &initialMessage)
	if err != nil {
		code := syscode.CodeConnectInternal
		statusCode := websocket.StatusInternalError
		msg := err.Error()

		if errors.Is(err, context.DeadlineExceeded) {
			code = syscode.CodeConnectHelloTimeout
			statusCode = websocket.StatusPolicyViolation
			msg = "Timeout waiting for SDK connect message"

			c.logger.Debug("Timeout waiting for SDK connect message")
		}

		return nil, &SocketError{
			SysCode:    code,
			StatusCode: statusCode,
			Msg:        msg,
		}
	}

	if initialMessage.Kind != connect.GatewayMessageType_SDK_CONNECT {
		c.logger.Debug("initial SDK message was not connect")

		return nil, &SocketError{
			SysCode:    syscode.CodeConnectHelloInvalidMsg,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Invalid first message, expected sdk-connect",
		}
	}

	if err := proto.Unmarshal(initialMessage.Payload, &initialMessageData); err != nil {
		c.logger.Debug("initial SDK message contained invalid Protobuf")

		return nil, &SocketError{
			SysCode:    syscode.CodeConnectHelloInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Invalid Protobuf in SDK connect message",
		}
	}

	var authResp *AuthResponse
	{
		// Run auth, add to distributed state
		authResp, err = c.auther(ctx, &initialMessageData)
		if err != nil {
			c.logger.Error("connect auth failed", "err", err)
			return nil, &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "Internal error",
			}
		}

		if authResp == nil {
			c.logger.Debug("Auth failed")
			return nil, &SocketError{
				SysCode:    syscode.CodeConnectAuthFailed,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        "Authentication failed",
			}
		}

		initialMessageData.AuthData.AccountId = authResp.AccountID.String()
		initialMessageData.AuthData.EnvId = authResp.EnvID.String()

		c.logger.Debug("SDK successfully authenticated", "authResp", authResp)
	}

	if err := c.stateManager.AddConnection(ctx, &initialMessageData); err != nil {
		_ = ws.Close(websocket.StatusInternalError, "connection not stored")
		return nil, &SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "connection not stored",
		}
	}

	for _, l := range c.lifecycles {
		go l.OnConnected(ctx, &initialMessageData)
	}

	return &initialMessageData, nil
}

func (c *connectGatewaySvc) handleSdkReply(ctx context.Context, log *slog.Logger, appId uuid.UUID, msg *connect.ConnectMessage) error {
	var data connect.SDKResponse
	if err := proto.Unmarshal(msg.Payload, &data); err != nil {
		return fmt.Errorf("invalid response type: %w", err)
	}

	log.Debug("notifying executor about response", "status", data.Status.String(), "no_retry", data.NoRetry, "retry_after", data.RetryAfter)

	err := c.receiver.NotifyExecutor(ctx, appId, &data)
	if err != nil {
		return fmt.Errorf("could not notify executor: %w", err)
	}

	return nil
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

	stateManager state.ConnectionStateManager
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

func newConnectRouter(stateManager state.ConnectionStateManager, receiver pubsub.RequestReceiver, db cqrs.Manager) service.Service {
	return &connectRouterSvc{
		stateManager: stateManager,
		receiver:     receiver,
		dbcqrs:       db,
	}
}
