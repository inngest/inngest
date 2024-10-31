package connect

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
)

const GatewaySubProtocol = "v0.connect.inngest.com"

type GatewayMessageType string

const GatewayMessageTypeHello GatewayMessageType = "gateway-hello"

const GatewayMessageTypeSDKConnect GatewayMessageType = "sdk-connect"

type AuthData struct {
	HashedSigningKey []byte `json:"hashed_signing_key"`
}

type SessionDetails struct {
	// InstanceId represents the persistent identifier for this connection.
	// This must not change across the lifetime of the connection, including reconnects.
	InstanceId string `json:"instance_id"`

	// ConnectionId is the transient identifier for a concrete connection. This is different
	// from InstanceId as it is generated for each connection.
	// This is mainly used for debugging purposes.
	ConnectionId string `json:"connection_id"`

	FunctionHash []byte  `json:"function_hash"`
	BuildID      *string `json:"build_id"`
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

const GatewayMessageTypeSync GatewayMessageType = "sync"

type GatewayMessageTypeSyncData struct {
	DeployId *string `json:"deployId"`
}

const GatewayMessageTypeExecutorRequest GatewayMessageType = "executor-request"

type GatewayMessageTypeExecutorRequestData struct {
	RequestId string `json:"request_id"`

	AppId uuid.UUID `json:"app_id"`

	FunctionSlug string  `json:"fn_slug"`
	StepId       *string `json:"step_id"`
	RequestBytes []byte  `json:"req"`
}

const GatewayMessageTypeSDKReply GatewayMessageType = "sdk-reply"

type SdkResponseStatus int

const (
	SdkResponseStatusNotCompleted SdkResponseStatus = http.StatusPartialContent
	SdkResponseStatusDone         SdkResponseStatus = http.StatusOK
	SdkResponseStatusError        SdkResponseStatus = http.StatusInternalServerError
)

type SdkResponse struct {
	RequestId string `json:"replyId"`

	Status SdkResponseStatus `json:"status"`
	Body   []byte            `json:"body"`

	// These are modeled after the headers for code reuse in httpdriver.ShouldRetry
	NoRetry    string `json:"no_retry"`
	RetryAfter string `json:"retry_after"`
	SdkVersion string `json:"sdk_version"`
}

type GatewayMessage struct {
	Kind GatewayMessageType `json:"kind"`
	Data json.RawMessage    `json:"data"`
}
