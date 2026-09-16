package api

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	testgolang "github.com/inngest/inngest/tests/golang"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type v2RunsEventData struct {
	Selection string `json:"selection"`
}

type v2RunsListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Page struct {
		HasMore bool    `json:"hasMore"`
		Cursor  *string `json:"cursor"`
	} `json:"page"`
}

type v2RunsErrorResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func getV2Runs(ctx context.Context, params url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, devURL+"/api/v2/runs?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func readV2RunsResponse(resp *http.Response) (v2RunsListResponse, error) {
	defer resp.Body.Close()
	var result v2RunsListResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

func TestV2ListRunsCEL(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	apiClient := client.New(t)

	appID := "v2runs-" + ulid.MustNew(ulid.Now(), nil).String()
	eventName := appID + "/run"
	functionID := "cel-filter"
	invocations := make(chan struct {
		selection string
		runID     string
	}, 4)

	inngestClient, server, registerFuncs := testgolang.NewSDKHandler(t, appID)
	defer server.Close()

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: functionID, Retries: new(0)},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[v2RunsEventData]) (any, error) {
			invocations <- struct {
				selection string
				runID     string
			}{selection: input.Event.Data.Selection, runID: input.InputCtx.RunID}
			return input.Event.Data.Selection, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Allow registration to propagate before sending the two control events.
	<-time.After(2 * time.Second)
	for _, selection := range []string{"match", "other"} {
		_, err := inngestClient.Send(ctx, inngestgo.Event{
			Name: eventName,
			Data: map[string]any{"selection": selection},
		})
		r.NoError(err)
	}

	runIDs := map[string]string{}
	for len(runIDs) < 2 {
		select {
		case invocation := <-invocations:
			runIDs[invocation.selection] = invocation.runID
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for matching and nonmatching runs")
		}
	}
	for _, runID := range runIDs {
		apiClient.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status: models.FunctionStatusCompleted,
		})
	}

	baseParams := url.Values{
		"appId":      {appID},
		"functionId": {functionID},
		"limit":      {"20"},
	}

	// Establish that both control runs are visible before testing selectivity.
	r.Eventually(func() bool {
		resp, err := getV2Runs(ctx, baseParams)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			return false
		}
		result, err := readV2RunsResponse(resp)
		if err != nil || len(result.Data) != 2 {
			return false
		}
		seen := map[string]bool{}
		for _, run := range result.Data {
			seen[run.ID] = true
		}
		return seen[runIDs["match"]] && seen[runIDs["other"]]
	}, 10*time.Second, 200*time.Millisecond)

	filteredParams := maps.Clone(baseParams)
	filteredParams.Set("query", `event.data.selection == "match"`)
	resp, err := getV2Runs(ctx, filteredParams)
	r.NoError(err)
	r.Equal(http.StatusOK, resp.StatusCode)
	result, err := readV2RunsResponse(resp)
	r.NoError(err)
	r.Len(result.Data, 1)
	r.Equal(runIDs["match"], result.Data[0].ID)
	r.False(result.Page.HasMore)

	malformedParams := maps.Clone(baseParams)
	malformedParams.Set("query", "event.data.selection ==")
	resp, err = getV2Runs(ctx, malformedParams)
	r.NoError(err)
	defer resp.Body.Close()
	r.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	r.NoError(err)
	var errorResult v2RunsErrorResponse
	r.NoError(json.Unmarshal(body, &errorResult), string(body))
	r.Len(errorResult.Errors, 1)
	r.Equal("expression_invalid", errorResult.Errors[0].Code)
	r.NotEmpty(errorResult.Errors[0].Message)
}
