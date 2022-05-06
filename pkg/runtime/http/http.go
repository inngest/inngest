package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/inngest/inngestctl/inngest"
)

var (
	DefaultExecutor = executor{
		client: &http.Client{
			Timeout: 15 * time.Minute,
		},
	}
)

func Execute(ctx context.Context, action inngest.ActionVersion, state map[string]interface{}) (map[string]interface{}, error) {
	return DefaultExecutor.Execute(ctx, action, state)
}

type executor struct {
	client *http.Client
}

func (e executor) Execute(ctx context.Context, action inngest.ActionVersion, state map[string]interface{}) (map[string]interface{}, error) {
	rt, ok := action.Runtime.Runtime.(inngest.RuntimeHTTP)
	if !ok {
		return nil, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime")
	}

	input, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, rt.URL, bytes.NewBuffer(input))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}

	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	output := map[string]interface{}{}
	if err = json.Unmarshal(byt, &output); err != nil {
		return nil, fmt.Errorf("Invalid JSON returned: \n%s", string(byt))
	}

	return output, nil
}
