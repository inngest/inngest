package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/publicerr"
)

var (
	DeployErrUnauthorized        = fmt.Errorf("unauthorized")
	DeployErrForbidden           = fmt.Errorf("forbidden")
	DeployErrNotFound            = fmt.Errorf("url_not_found")
	DeployErrInternalError       = fmt.Errorf("internal_server_error")
	DeployErrNoBranchName        = fmt.Errorf("missing_branch_env_name")
	DeployErrInvalidSigningKey   = fmt.Errorf("invalid_signing_key")
	DeployErrNoSigningKey        = fmt.Errorf("missing_signing_key")
	DeployErrInvalidFunction     = fmt.Errorf("invalid_function")
	DeployErrNoFunctions         = fmt.Errorf("no_functions")
	DeployErrUnreachable         = fmt.Errorf("unreachable")
	DeployErrUnsupportedProtocol = fmt.Errorf("unsupported_protocol")

	Client = http.Client{
		Timeout: 10 * time.Second,
	}
)

type pingResult struct {
	Err error

	// If ping response came from the SDK.
	IsSDK bool
}

func Ping(ctx context.Context, url string) pingResult {
	isSDK := false

	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return pingResult{
			Err: publicerr.WrapWithData(
				err,
				400,
				fmt.Sprintf("There was an error registering your app: %s", err.Error()),
				map[string]any{
					"error_code": err.Error(),
				},
			),
			IsSDK: isSDK,
		}
	}
	req.Header.Set(headers.HeaderKeyServerKind, headers.ServerKindDev)
	resp, err := Client.Do(req)
	if err != nil {
		err = handlePingError(err)
		return pingResult{
			Err: publicerr.WrapWithData(
				err,
				400,
				"There was an error registering your app",
				map[string]any{
					"error_code": err.Error(),
				},
			),
			IsSDK: isSDK,
		}
	}

	// Assume that the response came from the SDK if it has the SDK header.
	isSDK = resp.Header.Get(headers.HeaderKeySDK) != ""

	// If there was no client error, attempt to get any errors
	// from the SDK response
	if err = GetDeployError(resp); err != nil {
		return pingResult{
			Err: publicerr.WrapWithData(
				err,
				400,
				"There was an error registering your app",
				map[string]any{
					"error_code":           err.Error(),
					"response_headers":     resp.Header,
					"response_status_code": resp.StatusCode,
				},
			),
			IsSDK: isSDK,
		}
	}
	return pingResult{IsSDK: isSDK}
}

func handlePingError(err error) error {
	if strings.Contains(err.Error(), "server gave HTTP response to HTTPS") {
		return DeployErrUnsupportedProtocol
	}
	if strings.Contains(err.Error(), "unsupported protocol") {
		return DeployErrUnsupportedProtocol
	}
	return DeployErrUnreachable
}

// GetDeployError returns a deploy error, if found, from the given
// deploy request.
func GetDeployError(resp *http.Response) error {
	if resp.StatusCode == http.StatusBadRequest {
		// 400s usually contain SDK error messages that we want to pass through
		byt, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		type result struct {
			Message string `json:"message"`
		}
		r := &result{}
		if err := json.Unmarshal(byt, &r); err != nil {
			return err
		}
		// XXX: We should move these error codes into each SDK.
		if r.Message == "Your signing key is invalid" {
			return DeployErrInvalidSigningKey
		} else if strings.Contains(r.Message, "No functions registered") {
			return DeployErrNoFunctions
		} else if strings.Contains(r.Message, "function is invalid") {
			return DeployErrInvalidFunction
		} else if strings.Contains(r.Message, "You didn't specify your workspace's signing key") {
			return DeployErrNoSigningKey
		} else if strings.Contains(r.Message, "No INNGEST_ENV branch name found") {
			return DeployErrNoBranchName
		}
		return fmt.Errorf(r.Message)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return DeployErrUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return DeployErrForbidden
	}
	if resp.StatusCode == http.StatusNotFound {
		return DeployErrNotFound
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return DeployErrInternalError
	}
	return nil
}
