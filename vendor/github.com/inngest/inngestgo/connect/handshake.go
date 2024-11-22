package connect

import (
	"context"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
	"runtime"
	"time"
)

func (h *connectHandler) performConnectHandshake(ctx context.Context, connectionId string, ws *websocket.Conn, gatewayHost string, data connectionEstablishData) (bool, error) {
	// Wait for gateway hello message
	{
		initialMessageTimeout, cancelInitialTimeout := context.WithTimeout(ctx, 5*time.Second)
		defer cancelInitialTimeout()
		var helloMessage connectproto.ConnectMessage
		err := wsproto.Read(initialMessageTimeout, ws, &helloMessage)
		if err != nil {
			h.hostsManager.markUnreachableGateway(gatewayHost)
			return true, fmt.Errorf("did not receive gateway hello message: %w", err)
		}

		if helloMessage.Kind != connectproto.GatewayMessageType_GATEWAY_HELLO {
			h.hostsManager.markUnreachableGateway(gatewayHost)
			return true, fmt.Errorf("expected gateway hello message, got %s", helloMessage.Kind)
		}

		h.logger.Debug("received gateway hello message")
	}

	// Send connect message
	{

		apiOrigin := h.opts.APIBaseUrl
		if h.opts.IsDev {
			apiOrigin = h.opts.DevServerUrl
		}

		data, err := proto.Marshal(&connectproto.WorkerConnectRequestData{
			SessionId: &connectproto.SessionIdentifier{
				BuildId:      h.opts.BuildId,
				InstanceId:   h.instanceId(),
				ConnectionId: connectionId,
			},
			AuthData: &connectproto.AuthData{
				HashedSigningKey: data.hashedSigningKey,
			},
			AppName: h.opts.AppName,
			Config: &connectproto.ConfigDetails{
				Capabilities: data.marshaledCapabilities,
				Functions:    data.marshaledFns,
				ApiOrigin:    apiOrigin,
			},
			SystemAttributes: &connectproto.SystemAttributes{
				CpuCores: data.numCpuCores,
				MemBytes: data.totalMem,
				Os:       runtime.GOOS,
			},
			Environment:              h.opts.Env,
			Platform:                 h.opts.Platform,
			SdkVersion:               h.opts.SDKVersion,
			SdkLanguage:              h.opts.SDKLanguage,
			WorkerManualReadinessAck: data.manualReadinessAck,
		})
		if err != nil {
			return false, fmt.Errorf("could not serialize sdk connect message: %w", err)
		}

		err = wsproto.Write(ctx, ws, &connectproto.ConnectMessage{
			Kind:    connectproto.GatewayMessageType_WORKER_CONNECT,
			Payload: data,
		})
		if err != nil {
			return true, fmt.Errorf("could not send initial message")
		}
	}

	// Wait for gateway ready message
	{
		connectionReadyTimeout, cancelConnectionReadyTimeout := context.WithTimeout(ctx, 20*time.Second)
		defer cancelConnectionReadyTimeout()
		var connectionReadyMsg connectproto.ConnectMessage
		err := wsproto.Read(connectionReadyTimeout, ws, &connectionReadyMsg)
		if err != nil {
			return true, fmt.Errorf("did not receive gateway connection ready message: %w", err)
		}

		if connectionReadyMsg.Kind != connectproto.GatewayMessageType_GATEWAY_CONNECTION_READY {
			return true, fmt.Errorf("expected gateway connection ready message, got %s", connectionReadyMsg.Kind)
		}

		h.logger.Debug("received gateway connection ready message")
	}

	return false, nil
}
