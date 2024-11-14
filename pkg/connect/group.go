package connect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/proto/gen/connect/v1"
)

func workerGroupHashFromConnRequest(req *connect.WorkerConnectRequestData, authResp *AuthResponse, sessionDetails *connect.SessionDetails) (string, error) {
	buildId := ""
	if req.SessionId.BuildId != nil {
		buildId = *req.SessionId.BuildId
	}

	h := sha256.New()

	_, err := h.Write(authResp.AccountID[:])
	if err != nil {
		return "", fmt.Errorf("could not add account ID to hash input: %w", err)
	}

	_, err = h.Write(authResp.EnvID[:])
	if err != nil {
		return "", fmt.Errorf("could not add env ID to hash input: %w", err)
	}

	_, err = h.Write([]byte(req.SdkLanguage))
	if err != nil {
		return "", fmt.Errorf("could not add SDK language to hash input: %w", err)
	}

	_, err = h.Write([]byte(req.SdkVersion))
	if err != nil {
		return "", fmt.Errorf("could not add SDK version to hash input: %w", err)
	}

	if req.Platform != nil {
		_, err = h.Write([]byte(req.GetPlatform()))
		if err != nil {
			return "", fmt.Errorf("could not add SDK platform to hash input: %w", err)
		}
	}

	_, err = h.Write(sessionDetails.FunctionHash)
	if err != nil {
		return "", fmt.Errorf("could not add function hash to hash input: %w", err)
	}

	_, err = h.Write([]byte(buildId))
	if err != nil {
		return "", fmt.Errorf("could not add build ID to hash input: %w", err)
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
		slugs[i] = fn.Slug
	}

	workerGroup := &state.WorkerGroup{
		AccountID: authResp.AccountID,
		EnvID:     authResp.EnvID,
		// TODO Fix this
		AppID:         &uuid.Nil, // If the app was not synced, the ID won't exist yet.
		SDKLang:       req.SdkLanguage,
		SDKVersion:    req.SdkVersion,
		SDKPlatform:   req.GetPlatform(),
		FunctionSlugs: slugs,
		// TODO Can we load the initial sync ID from the state?
		SyncID: nil,
		Hash:   hash,
		SyncData: state.SyncData{
			Env:              req.GetEnvironment(),
			Functions:        functions,
			Capabilities:     sdk.Capabilities{},
			AppName:          req.AppName,
			APIOrigin:        "http://127.0.0.1:8288",
			HashedSigningKey: string(req.AuthData.GetHashedSigningKey()),
		},
	}

	return workerGroup, nil
}
