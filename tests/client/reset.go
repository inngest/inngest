package client

import (
	"context"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func (c *Client) ResetAll(t *testing.T) {
	ctx := context.Background()

	reqUrl, err := url.Parse(c.APIHost + "/test/reset")
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl.String(), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
}
