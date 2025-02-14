package connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"runtime"
	"time"
)

type reconnectError struct {
	err error
}

func newReconnectErr(wrapped error) error {
	return &reconnectError{wrapped}
}

func (e reconnectError) Unwrap() error {
	return e.err
}

func (e reconnectError) Error() string {
	return fmt.Sprintf("reconnect error: %v", e.err)
}

func shouldReconnect(err error) bool {
	var reconnectError *reconnectError
	ok := errors.As(err, &reconnectError)
	return ok
}

var ErrUnauthenticated = fmt.Errorf("authentication failed")
var ErrTooManyConnections = fmt.Errorf("too many connections")

func (h *connectHandler) performConnectHandshake(ctx context.Context, connectionId string, ws *websocket.Conn, startResponse *connectproto.StartResponse, data connectionEstablishData, startTime time.Time) error {
	// Wait for gateway hello message
	{
		initialMessageTimeout, cancelInitialTimeout := context.WithTimeout(ctx, 5*time.Second)
		defer cancelInitialTimeout()
		var helloMessage connectproto.ConnectMessage
		err := wsproto.Read(initialMessageTimeout, ws, &helloMessage)
		if err != nil {
			return newReconnectErr(fmt.Errorf("did not receive gateway hello message: %w", err))
		}

		if helloMessage.Kind != connectproto.GatewayMessageType_GATEWAY_HELLO {
			return newReconnectErr(fmt.Errorf("expected gateway hello message, got %s", helloMessage.Kind))
		}

		h.logger.Debug("received gateway hello message")
	}

	// Send connect message
	{
		data, err := proto.Marshal(&connectproto.WorkerConnectRequestData{
			SessionId: &connectproto.SessionIdentifier{
				AppVersion:   h.opts.AppVersion,
				InstanceId:   h.instanceId(),
				ConnectionId: connectionId,
			},
			AuthData: &connectproto.AuthData{
				SessionToken: startResponse.GetSessionToken(),
				SyncToken:    startResponse.GetSyncToken(),
			},
			AppName: h.opts.AppName,
			Config: &connectproto.ConfigDetails{
				Capabilities: data.marshaledCapabilities,
				Functions:    data.marshaledFns,
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
			StartedAt:                timestamppb.New(startTime),
		})
		if err != nil {
			return fmt.Errorf("could not serialize sdk connect message: %w", err)
		}

		err = wsproto.Write(ctx, ws, &connectproto.ConnectMessage{
			Kind:    connectproto.GatewayMessageType_WORKER_CONNECT,
			Payload: data,
		})
		if err != nil {
			return newReconnectErr(fmt.Errorf("could not send initial message"))
		}
	}

	// Wait for gateway ready message
	{
		connectionReadyTimeout, cancelConnectionReadyTimeout := context.WithTimeout(ctx, 20*time.Second)
		defer cancelConnectionReadyTimeout()
		var connectionReadyMsg connectproto.ConnectMessage
		err := wsproto.Read(connectionReadyTimeout, ws, &connectionReadyMsg)
		if err != nil {
			return newReconnectErr(fmt.Errorf("did not receive gateway connection ready message: %w", err))
		}

		if connectionReadyMsg.Kind != connectproto.GatewayMessageType_GATEWAY_CONNECTION_READY {
			return newReconnectErr(fmt.Errorf("expected gateway connection ready message, got %s", connectionReadyMsg.Kind))
		}

		h.logger.Debug("received gateway connection ready message")
	}

	return nil
}
