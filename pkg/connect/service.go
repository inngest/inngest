package connect

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"log"
	"net/http"
	"time"
)

type connectGatewaySvc struct {
	runCtx context.Context
}

const GatewaySubProtocol = "v0.connect.inngest.com"

type gatewayMessageType string

const gatewayMessageTypeHello gatewayMessageType = "gateway-hello"
const gatewayMessageTypeSDKConnect gatewayMessageType = "sdk-connect"

const gatewayMessageTypeExecutorRequest gatewayMessageType = "executor-request"
const gatewayMessageTypeSDKReply gatewayMessageType = "sdk-reply"

type gatewayMessage struct {
	Kind gatewayMessageType `json:"kind"`
	Data json.RawMessage    `json:"data"`
}

func (c *connectGatewaySvc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the context as needed. Use of r.Context() is not recommended
		// to avoid surprising behavior (see http.Hijacker).
		ctx, cancel := context.WithCancel(c.runCtx)
		defer cancel()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{
				GatewaySubProtocol,
			},
		})
		if err != nil {
			return
		}
		defer func() {
			logger.StdlibLogger(ctx).Debug("Closing WebSocket connection")

			ws.CloseNow()
		}()

		err = wsjson.Write(ctx, ws, gatewayMessage{
			Kind: gatewayMessageTypeHello,
		})
		if err != nil {
			return
		}

		shorterContext, cancelShorter := context.WithTimeout(ctx, 5*time.Second)
		defer cancelShorter()
		var initialMessage gatewayMessage
		err = wsjson.Read(shorterContext, ws, &initialMessage)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logger.StdlibLogger(ctx).Debug("Timeout waiting for SDK connect message")
				ws.Close(websocket.StatusPolicyViolation, "Timeout waiting for SDK connect message")
			}

			return
		}

		if initialMessage.Kind != gatewayMessageTypeSDKConnect {
			ws.Close(websocket.StatusPolicyViolation, "Invalid first message, expected sdk-connect")
			return
		}

		// TODO Run auth, add to distributed state

		// Run loop
		for {
			if ctx.Err() != nil {
				break
			}

			var v gatewayMessage
			err = wsjson.Read(ctx, ws, &v)
			if err != nil {
				return
			}

			log.Printf("received: %v", v)

			switch v.Kind {
			case gatewayMessageTypeSDKReply:
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

func NewConnectGatewayService() (service.Service, http.Handler) {
	svc := &connectGatewaySvc{}

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
