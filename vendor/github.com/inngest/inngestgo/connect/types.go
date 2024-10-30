package connect

import (
	"encoding/json"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"net/http"
)

const GatewaySubProtocol = "v0.connect.inngest.com"

type GatewayMessageType string

const GatewayMessageTypeHello GatewayMessageType = "gateway-hello"

const GatewayMessageTypeSDKConnect GatewayMessageType = "sdk-connect"

type AuthData struct {
	Challenge []byte `json:"challenge"`
	Signature []byte `json:"signature"`
}

type SessionDetails struct {
	// InstanceId represents the persistent identifier for this connection.
	// This must not change across the lifetime of the connection, including reconnects.
	InstanceId string `json:"instance_id"`

	// ConnectionId is the transient identifier for a concrete connection. This is different
	// from InstanceId as it is generated for each connection.
	// This is mainly used for debugging purposes.
	ConnectionId string `json:"connectionId"`
}

type GatewayMessageTypeSDKConnectData struct {
	Session SessionDetails `json:"session"`

	Authz AuthData `json:"authz"`

	AppName     string  `json:"app_name"`
	Env         *string `json:"env"`
	Framework   *string `json:"framework"`
	Platform    *string `json:"platform"`
	SDKAuthor   string  `json:"sdk_author"`
	SDKLanguage string  `json:"sdk_language"`
	SDKVersion  string  `json:"sdk_version"`
}

const GatewayMessageTypeExecutorRequest GatewayMessageType = "executor-request"

type GatewayMessageTypeExecutorRequestData struct {
	FunctionSlug string             `json:"fn_slug"`
	StepId       *string            `json:"step_id"`
	Request      sdkrequest.Request `json:"req"`
}

const GatewayMessageTypeSDKReply GatewayMessageType = "sdk-reply"

type SdkResponseStatus int

const (
	SdkResponseStatusNotCompleted SdkResponseStatus = http.StatusPartialContent
	SdkResponseStatusDone         SdkResponseStatus = http.StatusOK
	SdkResponseStatusError        SdkResponseStatus = http.StatusInternalServerError
)

type SdkResponse struct {
	Status SdkResponseStatus       `json:"status"`
	Ops    []state.GeneratorOpcode `json:"ops"`
	Resp   any                     `json:"resp"`
	Err    *string                 `json:"err"`
}

type GatewayMessage struct {
	Kind GatewayMessageType `json:"kind"`
	Data json.RawMessage    `json:"data"`
}
