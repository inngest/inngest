package connect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"strings"
)

func workerGroupHashFromConnRequest(req *connect.WorkerConnectRequestData, authResp *AuthResponse, sessionDetails *connect.SessionDetails) (string, error) {
	buildId := ""
	if req.SessionId.BuildId != nil {
		buildId = *req.SessionId.BuildId
	}

	platform := "-"
	if req.Platform != nil {
		platform = req.GetPlatform()
	}

	base := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s",
		authResp.AccountID,
		authResp.EnvID,
		req.SdkLanguage,
		req.SdkVersion,
		platform,
		sessionDetails.FunctionHash,
		buildId,
	)

	h := sha256.New()
	_, err := h.Write([]byte(base))
	if err != nil {
		return "", fmt.Errorf("could not compute worker group hash: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func NewWorkerGroupFromConnRequest(
	ctx context.Context,
	req *connect.WorkerConnectRequestData,
	authResp *AuthResponse,
	sessionDetails *connect.SessionDetails,
) (*state.WorkerGroup, error) {
	hash, err := workerGroupHashFromConnRequest(req, authResp, sessionDetails)
	if err != nil {
		return nil, &SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "Internal error",
		}
	}

	// TODO: check state store and see if group already exists, if so return that

	var (
		functions    []sdk.SDKFunction
		capabilities sdk.Capabilities
	)
	if err := json.Unmarshal(req.Config.Functions, &functions); err != nil {
		return nil, SocketError{
			SysCode:    syscode.CodeConnectInvalidFunctionConfig,
			Msg:        "Invalid function config",
			StatusCode: websocket.StatusPolicyViolation,
		}
	}

	if err := json.Unmarshal(req.Config.Capabilities, &capabilities); err != nil {
		return nil, &SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "Invalid SDK capabilities",
		}
	}

	slugs := make([]string, len(functions))
	for i, fn := range functions {
		slugs[i] = strings.TrimPrefix(fn.Slug, fmt.Sprintf("%s-", req.AppName))
	}

	workerGroup := &state.WorkerGroup{
		AccountID:     authResp.AccountID,
		EnvID:         authResp.EnvID,
		SDKLang:       req.SdkLanguage,
		SDKVersion:    req.SdkVersion,
		SDKPlatform:   req.GetPlatform(),
		FunctionSlugs: slugs,
		Hash:          hash,
		SyncData: state.SyncData{
			Functions: functions,
		},
	}

	return workerGroup, nil
}
