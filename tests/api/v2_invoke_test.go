package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	testgolang "github.com/inngest/inngest/tests/golang"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const devURL = "http://127.0.0.1:8288"

// invokeResponse represents the JSON envelope returned by the V2 invoke endpoint.
type invokeResponse struct {
	Data struct {
		RunID string `json:"runId"`
	} `json:"data"`
	Metadata struct {
		FetchedAt string `json:"fetchedAt"`
	} `json:"metadata"`
}

func postInvoke(ctx context.Context, functionSlug string, body map[string]any) (*http.Response, error) {
	byt, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v2/functions/%s/invoke", devURL, functionSlug),
		bytes.NewReader(byt),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func TestV2InvokeFunction(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "v2invoke-" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := testgolang.NewSDKHandler(t, appID)
	defer server.Close()

	fnID := "test-fn"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      fnID,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			return "hello", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Allow registration to propagate
	<-time.After(2 * time.Second)

	slug := fmt.Sprintf("%s-%s", appID, fnID)
	resp, err := postInvoke(ctx, slug, map[string]any{
		"data": map[string]any{"test": true},
	})
	r.NoError(err)
	defer resp.Body.Close()

	r.Equal(http.StatusOK, resp.StatusCode)

	var result invokeResponse
	body, err := io.ReadAll(resp.Body)
	r.NoError(err)
	r.NoError(json.Unmarshal(body, &result))

	r.NotEmpty(result.Data.RunID, "response should contain a run ID")
	// Validate the run ID is a valid ULID
	_, err = ulid.Parse(result.Data.RunID)
	r.NoError(err, "run ID should be a valid ULID")

	// Verify the function actually ran to completion
	runID := result.Data.RunID
	c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
		Status:    models.FunctionStatusCompleted,
		NewTraces: true,
	})
}

func TestV2InvokeFunctionIdempotency(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	appID := "v2idemp-" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := testgolang.NewSDKHandler(t, appID)
	defer server.Close()

	fnID := "idemp-fn"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      fnID,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			return "ok", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	<-time.After(2 * time.Second)

	slug := fmt.Sprintf("%s-%s", appID, fnID)

	// First invoke with idempotency key
	resp1, err := postInvoke(ctx, slug, map[string]any{
		"data":            map[string]any{"key": "value"},
		"idempotency_key": "test-key-1",
	})
	r.NoError(err)
	defer resp1.Body.Close()
	r.Equal(http.StatusOK, resp1.StatusCode)

	var result1 invokeResponse
	body1, err := io.ReadAll(resp1.Body)
	r.NoError(err)
	r.NoError(json.Unmarshal(body1, &result1))
	r.NotEmpty(result1.Data.RunID)

	// Second invoke with same idempotency key → 409
	resp2, err := postInvoke(ctx, slug, map[string]any{
		"data":            map[string]any{"key": "value"},
		"idempotency_key": "test-key-1",
	})
	r.NoError(err)
	defer resp2.Body.Close()
	r.Equal(http.StatusConflict, resp2.StatusCode)

	var result2 invokeResponse
	body2, err := io.ReadAll(resp2.Body)
	r.NoError(err)
	r.NoError(json.Unmarshal(body2, &result2))
	r.Equal(result1.Data.RunID, result2.Data.RunID, "idempotent request should return same run ID")

	// Third invoke with different idempotency key → 200
	resp3, err := postInvoke(ctx, slug, map[string]any{
		"data":            map[string]any{"key": "value"},
		"idempotency_key": "test-key-2",
	})
	r.NoError(err)
	defer resp3.Body.Close()
	r.Equal(http.StatusOK, resp3.StatusCode)

	var result3 invokeResponse
	body3, err := io.ReadAll(resp3.Body)
	r.NoError(err)
	r.NoError(json.Unmarshal(body3, &result3))
	r.NotEmpty(result3.Data.RunID)
	assert.NotEqual(t, result1.Data.RunID, result3.Data.RunID, "different idempotency key should produce different run ID")
}

func TestV2InvokeFunctionNotFound(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	resp, err := postInvoke(ctx, "nonexistent-fn", map[string]any{
		"data": map[string]any{"test": true},
	})
	r.NoError(err)
	defer resp.Body.Close()

	r.Equal(http.StatusNotFound, resp.StatusCode)
}

func TestV2InvokeFunctionValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing data returns 400", func(t *testing.T) {
		r := require.New(t)
		// POST with no data field
		resp, err := postInvoke(ctx, "some-fn", map[string]any{})
		r.NoError(err)
		defer resp.Body.Close()
		r.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("null data returns 400", func(t *testing.T) {
		r := require.New(t)
		resp, err := postInvoke(ctx, "some-fn", map[string]any{
			"data": nil,
		})
		r.NoError(err)
		defer resp.Body.Close()
		r.Equal(http.StatusBadRequest, resp.StatusCode)
	})
}
