package connect

import (
	"bytes"
	"context"
	"fmt"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
	"io"
	"log/slog"
	"net/http"
)

type workerApiClient struct {
	client     http.Client
	env        *string
	apiBaseUrl string
}

func newWorkerApiClient(apiBaseUrl string, env *string) *workerApiClient {
	return &workerApiClient{
		apiBaseUrl: apiBaseUrl,
		env:        env,
	}
}

func (a *workerApiClient) start(ctx context.Context, hashedSigningKey []byte, req *connect.StartRequest, logger *slog.Logger) (*connect.StartResponse, error) {
	reqBody, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("could not marshal start request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v0/connect/start", a.apiBaseUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("could not create start request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/protobuf")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(hashedSigningKey)))

	if a.env != nil {
		httpReq.Header.Add("X-Inngest-Env", *a.env)
	}

	logger.Debug("sending start request", "url", httpReq.URL.String(), "env", a.env)

	httpRes, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not send start request: %w", err)
	}

	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		if httpRes.StatusCode == http.StatusUnauthorized {
			return nil, newReconnectErr(ErrUnauthenticated)
		}

		if httpRes.StatusCode == http.StatusTooManyRequests {
			return nil, newReconnectErr(ErrTooManyConnections)
		}

		byt, err := io.ReadAll(httpRes.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read start error: %w", err)
		}

		logger.Error("start request received unexpected error", "url", httpReq.URL.String(), "env", a.env, "status", httpRes.StatusCode, "resp", string(byt))

		return nil, fmt.Errorf("unexpected status code: %d", httpRes.StatusCode)
	}

	byt, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read start response: %w", err)
	}

	res := &connect.StartResponse{}
	err = proto.Unmarshal(byt, res)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal start response: %w", err)
	}

	return res, nil
}

func (a *workerApiClient) sendBufferedMessage(ctx context.Context, hashedSigningKey []byte, req *connect.SDKResponse) error {
	reqBody, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("could not marshal sdk response for flush request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v0/connect/flush", a.apiBaseUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("could not create flush request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/protobuf")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(hashedSigningKey)))

	if a.env != nil {
		httpReq.Header.Add("X-Inngest-Env", *a.env)
	}

	httpRes, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("could not send flush request: %w", err)
	}

	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", httpRes.StatusCode)
	}

	byt, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return fmt.Errorf("could not read flush response: %w", err)
	}

	res := &connect.FlushResponse{}
	err = proto.Unmarshal(byt, res)
	if err != nil {
		return fmt.Errorf("could not unmarshal flush response: %w", err)
	}

	return nil
}
