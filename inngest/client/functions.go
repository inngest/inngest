package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// XXX: To avoid a import cycle (function uses state which uses client),
// we create a client specific type matching function.FunctionVersion
type FunctionVersion struct {
	FunctionID string     `json:"functionId"`
	Version    int        `json:"version"`
	Config     string     `json:"config"`
	ValidFrom  *time.Time `json:"validFrom"`
	ValidTo    *time.Time `json:"validTo"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// DeployFunction deploys a function for a given environment. Live determines if the function is a draft or live.
func (c httpClient) DeployFunction(ctx context.Context, config string, env string, live bool) (*FunctionVersion, error) {
	query := `
		mutation DeployFunction($config: String!, $env: Environment, $live: Boolean) {
			deployFunction(input: {
				config: $config
				env: $env
				live: $live
			}) {
				functionId version config validFrom validTo createdAt updatedAt
			}
		}`

	type response struct {
		DeployFunction *FunctionVersion
	}
	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"config": config,
		"env":    env,
		"live":   live,
	}})
	if err != nil {
		return nil, err
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling function version: %w", err)
	}

	return data.DeployFunction, nil
}
