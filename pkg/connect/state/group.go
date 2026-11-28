package state

import (
	"bytes"
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
	"sort"
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

	b, err := canonicalFunctionConfig(appConfig.Functions)
	if err != nil {
		return nil, fmt.Errorf("could not canonicalize function config: %w", err)
	}

	res := sha256.Sum256(b)
	functionHash = res[:]

	return functionHash, nil
}

// canonicalFunctionConfig returns the JCS form of the function config with the
// top-level function list sorted by function ID. SDKs may list the same
// functions in a different order, which must not produce a different worker
// group hash: the hash is also the sync idempotency key, so a new order would
// trigger an unnecessary app sync.
//
// Only the top-level list is reordered. Nested lists such as triggers and steps
// keep the order the SDK sent. A list already sorted by ID canonicalizes to the
// same bytes as before, so its hash is unchanged.
func canonicalFunctionConfig(functions []byte) ([]byte, error) {
	b, err := jcs.Transform(functions)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 || b[0] != '[' {
		return b, nil
	}

	// Elements of the canonical list are themselves canonical, and RawMessage
	// keeps their exact bytes.
	var items []json.RawMessage
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}

	type entry struct {
		id  string
		raw json.RawMessage
	}
	entries := make([]entry, len(items))
	for i, item := range items {
		var fn struct {
			ID string `json:"id"`
		}
		// Non-object elements sort first, by their bytes.
		_ = json.Unmarshal(item, &fn)
		entries[i] = entry{id: fn.ID, raw: item}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].id != entries[j].id {
			return entries[i].id < entries[j].id
		}
		return bytes.Compare(entries[i].raw, entries[j].raw) < 0
	})

	out := make([]byte, 0, len(b))
	out = append(out, '[')
	for i, e := range entries {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, e.raw...)
	}
	return append(out, ']'), nil
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

	featureObservations := sdk.FeatureObservationsFromProto(appConfig.GetFeatureObservations())
	featureObservationsHash, err := sdkFeatureObservationsHash(featureObservations)
	if err != nil {
		return nil, &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "Invalid SDK feature observations",
		}
	}

	workerGroup := &WorkerGroup{
		AccountID:               authResp.AccountID,
		EnvID:                   authResp.EnvID,
		AppName:                 appConfig.AppName,
		SDKLang:                 req.SdkLanguage,
		SDKVersion:              req.SdkVersion,
		SDKPlatform:             req.GetPlatform(),
		AppVersion:              appConfig.AppVersion,
		FunctionSlugs:           slugs,
		Hash:                    hash,
		FeatureObservationsHash: featureObservationsHash,
		SyncData: SyncData{
			Functions:           functions,
			SyncToken:           req.AuthData.SyncToken,
			AppConfig:           appConfig,
			FeatureObservations: featureObservations,
		},
		CreatedAt: time.Now(),
	}

	return workerGroup, nil
}
