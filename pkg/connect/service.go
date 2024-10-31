package connect

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	sdk_connect "github.com/inngest/inngestgo/connect"
	"log"
	"net/http"
	"time"
)

type gatewayOpt func(*connectGatewaySvc)

type AuthResponse struct {
	AccountID uuid.UUID
}

type GatewayAuthHandler func(context.Context, sdk_connect.GatewayMessageTypeSDKConnectData) (*AuthResponse, error)

type connectGatewaySvc struct {
	runCtx context.Context

	auther       GatewayAuthHandler
	stateManager ConnectionStateManager
	receiver     pubsub.RequestReceiver
	dbcqrs       cqrs.Manager
}

func WithGatewayAuthHandler(auth GatewayAuthHandler) gatewayOpt {
	return func(c *connectGatewaySvc) {
		c.auther = auth
	}
}

func WithConnectionStateManager(m ConnectionStateManager) gatewayOpt {
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

func (c *connectGatewaySvc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the context as needed. Use of r.Context() is not recommended
		// to avoid surprising behavior (see http.Hijacker).
		ctx, cancel := context.WithCancel(c.runCtx)
		defer cancel()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{
				sdk_connect.GatewaySubProtocol,
			},
		})
		if err != nil {
			return
		}
		defer func() {
			logger.StdlibLogger(ctx).Debug("Closing WebSocket connection")

			ws.CloseNow()
		}()

		logger.StdlibLogger(ctx).Debug("WebSocket connection established, sending hello")

		{
			err = wsjson.Write(ctx, ws, sdk_connect.GatewayMessage{
				Kind: sdk_connect.GatewayMessageTypeHello,
			})
			if err != nil {
				logger.StdlibLogger(ctx).Error("could not send hello", "err", err)

				return
			}
		}

		var initialMessageData sdk_connect.GatewayMessageTypeSDKConnectData
		{
			var initialMessage sdk_connect.GatewayMessage
			shorterContext, cancelShorter := context.WithTimeout(ctx, 5*time.Second)
			defer cancelShorter()
			err = wsjson.Read(shorterContext, ws, &initialMessage)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					logger.StdlibLogger(ctx).Debug("Timeout waiting for SDK connect message")
					ws.Close(websocket.StatusPolicyViolation, "Timeout waiting for SDK connect message")
				}

				return
			}

			if initialMessage.Kind != sdk_connect.GatewayMessageTypeSDKConnect {
				logger.StdlibLogger(ctx).Debug("initial SDK message was not connect")

				ws.Close(websocket.StatusPolicyViolation, "Invalid first message, expected sdk-connect")
				return
			}

			if err := json.Unmarshal(initialMessage.Data, &initialMessageData); err != nil {
				logger.StdlibLogger(ctx).Debug("initial SDK message contained invalid JSON")

				ws.Close(websocket.StatusPolicyViolation, "Invalid JSON in SDK connect message")
				return
			}
		}

		var authResp *AuthResponse
		{
			// Run auth, add to distributed state
			authResp, err = c.auther(ctx, initialMessageData)
			if err != nil {
				logger.StdlibLogger(ctx).Error("connect auth failed", "err", err)
				ws.Close(websocket.StatusInternalError, "Internal error")
				return
			}

			if authResp == nil {
				logger.StdlibLogger(ctx).Debug("Auth failed")

				ws.Close(websocket.StatusPolicyViolation, "Authentication failed")
				return
			}
		}

		logger.StdlibLogger(ctx).Debug("SDK successfully authenticated", "authResp", authResp)

		// TODO Check whether SDK group was already synced
		isAlreadySynced := false
		if !isAlreadySynced {
			logger.StdlibLogger(ctx).Debug("Sending sync message to SDK")
			data := map[string]any{
				// TODO Set this to prevent unattached syncs!
				"deployId": nil,
			}
			marshaled, err := json.Marshal(data)
			if err != nil {
				// TODO Handle this properly
				return
			}
			err = wsjson.Write(ctx, ws, sdk_connect.GatewayMessage{
				Kind: sdk_connect.GatewayMessageTypeSync,
				Data: marshaled,
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
				logger.StdlibLogger(ctx).Error("could not get apps", "err", err)
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

				logger.StdlibLogger(ctx).Error("could not find app", "appName", initialMessageData.AppName)
				ws.Close(websocket.StatusPolicyViolation, "Could not find app")
				return
			}

			break
		}

		fmt.Println("found app, connection is ready")

		// Wait for relevant messages and forward them to the socket
		go func() {
			// NOTE: This is not an exclusive 1-1 link between PubSub messages and connections:
			// - There are multiple gateway instances
			// - There are possibly multiple SDK deployments, each with their own WebSocket connection
			// -> We need to prevent sending the same request multiple times, to different connections
			err := c.receiver.ReceiveExecutorMessages(ctx, appId, func(data sdk_connect.GatewayMessageTypeExecutorRequestData) {
				fmt.Println("received msg", appId, data.RequestId)
				// This will be sent at least once (if there are more than one connection, every connection receives the message)
				err = c.receiver.AckMessage(ctx, appId, data.RequestId)
				if err != nil {
					// TODO Log error, retry?
					return
				}

				err = c.stateManager.SetRequestIdempotency(ctx, appId, data.RequestId)
				if err != nil {
					if errors.Is(err, ErrIdempotencyKeyExists) {
						// Another connection was faster than us, we can ignore this message
						return
					}

					// TODO Log error
					return
				}

				// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
				// We'll potentially lose data here

				// Now we're guaranteed to be the exclusive connection processing this message!

				marshaled, err := json.Marshal(data)
				if err != nil {
					return
				}

				// Forward message to SDK!
				err = wsjson.Write(ctx, ws, sdk_connect.GatewayMessage{
					Kind: sdk_connect.GatewayMessageTypeExecutorRequest,
					Data: marshaled,
				})
				if err != nil {
					// TODO The connection cannot be used, we need to let the executor know!
					return
				}
			})
			if err != nil {
				// TODO Log error, retry?
			}
		}()

		// Run loop
		go func() {
			for {
				if ctx.Err() != nil {
					break
				}

				var msg sdk_connect.GatewayMessage
				err = wsjson.Read(ctx, ws, &msg)
				if err != nil {
					return
				}

				log.Printf("received: %v", msg)

				switch msg.Kind {
				case sdk_connect.GatewayMessageTypeSDKReply:
					// Handle SDK reply
					err := c.handleSdkReply(ctx, appId, msg)
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

func (c *connectGatewaySvc) handleSdkReply(ctx context.Context, appId uuid.UUID, msg sdk_connect.GatewayMessage) error {
	var data sdk_connect.SdkResponse
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return fmt.Errorf("invalid response type: %w", err)
	}

	fmt.Println("notifying executor about response", appId, data.RequestId)

	err := c.receiver.NotifyExecutor(ctx, appId, pubsub.ProxyResponse{
		SdkResponse: &data,
	})
	if err != nil {
		return fmt.Errorf("could not notify executor: %w", err)
	}

	return nil
}

func NewConnectGatewayService(opts ...gatewayOpt) (service.Service, http.Handler) {
	svc := &connectGatewaySvc{}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, svc.Handler()
}

func (c *connectGatewaySvc) Name() string {
	return "connect-gateway"
}

func (c *connectGatewaySvc) Pre(ctx context.Context) error {
	return nil
}

func (c *connectGatewaySvc) Run(ctx context.Context) error {
	c.runCtx = ctx

	err := c.receiver.Wait(ctx)
	if err != nil {
		// TODO Should we retry? Exit here? This will interrupt existing connections!
		return fmt.Errorf("could not listen for pubsub messages: %w", err)
	}

	return nil
}

func (c *connectGatewaySvc) Stop(ctx context.Context) error {
	return nil
}
