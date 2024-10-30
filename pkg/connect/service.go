package connect

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	sdk_connect "github.com/inngest/inngestgo/connect"
	"log"
	"net/http"
	"time"
)

type gatewayOpt func(*connectGatewaySvc)

type GatewayAuthHandler func(context.Context, sdk_connect.GatewayMessageTypeSDKConnectData) (bool, error)

type connectGatewaySvc struct {
	runCtx context.Context

	auther       GatewayAuthHandler
	stateManager ConnectionStateManager
	receiver     RequestReceiver
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

func WithRequestReceiver(r RequestReceiver) gatewayOpt {
	return func(c *connectGatewaySvc) {
		c.receiver = r
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

		{
			err = wsjson.Write(ctx, ws, sdk_connect.GatewayMessage{
				Kind: sdk_connect.GatewayMessageTypeHello,
			})
			if err != nil {
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
				ws.Close(websocket.StatusPolicyViolation, "Invalid first message, expected sdk-connect")
				return
			}

			if err := json.Unmarshal(initialMessage.Data, &initialMessageData); err != nil {
				ws.Close(websocket.StatusPolicyViolation, "Invalid JSON in SDK connect message")
				return
			}
		}

		{
			// Run auth, add to distributed state
			ok, err := c.auther(ctx, initialMessageData)
			if err != nil {
				logger.StdlibLogger(ctx).Error("connect auth failed", "err", err)
				ws.Close(websocket.StatusInternalError, "Internal error")
				return
			}

			if !ok {
				ws.Close(websocket.StatusPolicyViolation, "Authentication failed")
				return
			}
		}

		// Run loop
		for {
			if ctx.Err() != nil {
				break
			}

			var v sdk_connect.GatewayMessage
			err = wsjson.Read(ctx, ws, &v)
			if err != nil {
				return
			}

			log.Printf("received: %v", v)

			switch v.Kind {
			case sdk_connect.GatewayMessageTypeSDKReply:
				// TODO Handle SDK reply
			default:
				// TODO Handle proper connection cleanup
				ws.Close(websocket.StatusPolicyViolation, "Invalid message kind")
				return
			}
		}

		ws.Close(websocket.StatusNormalClosure, "")
	})
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
	<-ctx.Done()
	return nil
}

func (c *connectGatewaySvc) Stop(ctx context.Context) error {
	return nil
}
