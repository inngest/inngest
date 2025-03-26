package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/connect/auth"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"time"
)

func workerGroupHashFromConnRequest(req *connect.WorkerConnectRequestData, authResp *auth.Response, appConfig *connect.AppConfiguration, functionHash []byte) (string, error) {
	appVersion := ""
	if appConfig.AppVersion != nil {
		appVersion = *appConfig.AppVersion
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
		functionHash,
		appVersion,
	)

	h := sha256.New()
	_, err := h.Write([]byte(base))
	if err != nil {
		return "", fmt.Errorf("could not compute worker group hash: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func functionConfigHash(appConfig *connect.AppConfiguration) ([]byte, error) {
	var functionHash []byte

	b, err := jcs.Transform(appConfig.Functions)
	if err != nil {
		return nil, fmt.Errorf("could not canonicalize function config: %w", err)
	}

	res := sha256.Sum256(b)
	functionHash = res[:]

	return functionHash, nil
}

// NewWorkerGroupFromConnRequest instantiates but does not store WorkerGroup for a new session.
func NewWorkerGroupFromConnRequest(
	ctx context.Context,
	req *connect.WorkerConnectRequestData,
	authResp *auth.Response,
	appConfig *connect.AppConfiguration,
) (*WorkerGroup, error) {
	functionHash, err := functionConfigHash(appConfig)
	if err != nil {
		return nil, fmt.Errorf("could not compute function config hash: %w", err)
	}

	hash, err := workerGroupHashFromConnRequest(req, authResp, appConfig, functionHash)
	if err != nil {
		return nil, &connecterrors.SocketError{
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
	if err := json.Unmarshal(appConfig.Functions, &functions); err != nil {
		return nil, connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInvalidFunctionConfig,
			Msg:        "Invalid function config",
			StatusCode: websocket.StatusPolicyViolation,
		}
	}

	if err := json.Unmarshal(req.Capabilities, &capabilities); err != nil {
		return nil, &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "Invalid SDK capabilities",
		}
	}

	slugs := make([]string, len(functions))
	for i, fn := range functions {
		// Use slug as is
		slugs[i] = fn.Slug
	}

	workerGroup := &WorkerGroup{
		AccountID:     authResp.AccountID,
		EnvID:         authResp.EnvID,
		AppName:       appConfig.AppName,
		SDKLang:       req.SdkLanguage,
		SDKVersion:    req.SdkVersion,
		SDKPlatform:   req.GetPlatform(),
		AppVersion:    appConfig.AppVersion,
		FunctionSlugs: slugs,
		Hash:          hash,
		SyncData: SyncData{
			Functions: functions,
			SyncToken: req.AuthData.SyncToken,
			AppConfig: appConfig,
		},
		CreatedAt: time.Now(),
	}

	return workerGroup, nil
}
