package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Action struct {
	DSN     string
	Name    string
	Tagline string
	Latest  *ActionVersion
}

type ActionVersion struct {
	VersionMajor int
	VersionMinor int
	ValidFrom    *time.Time
	ValidTo      *time.Time
	Runtime      string
}

func (c httpClient) Actions(ctx context.Context, includePublic bool) ([]*Action, error) {
	query := `
	  query ($filter: ActionFilter) {
	    actions(filter: $filter) {
	      dsn name tagline
	      latest {
		versionMajor
		versionMinor
		validFrom
		validTo
		runtime
	      }
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"filter": map[string]interface{}{
			"excludePublic": !includePublic,
		},
	}})
	if err != nil {
		return nil, err
	}

	type response struct {
		Actions []*Action
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.Actions, nil
}

// Action returns a specific action based on DSN
func (c httpClient) Action(ctx context.Context, dsn string) (*Action, error) {
	query := `
query ($dsn: String!) {
	action(dsn: $dsn) {
		dsn name tagline
		latest {
			versionMajor
			versionMinor
			validFrom
			validTo
			runtime
		}
	}
}
`

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"dsn": dsn,
	},
	})
	if err != nil {
		return nil, err
	}

	type response struct {
		Action *Action
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.Action, nil
}

func (c httpClient) CreateAction(ctx context.Context, input string) (*Action, error) {
	query := `
	  mutation CreateAction($config: String!) {
	    createAction(config: $config) {
	      dsn name
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"config": input,
	}})
	if err != nil {
		return nil, err
	}

	type response struct {
		CreateAction *Action
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.CreateAction, nil
}

type ActionVersionQualifier struct {
	DSN          string `json:"dsn"`
	VersionMajor int    `json:"versionMajor"`
	VersionMinor int    `json:"versionMinor"`
}

func (c httpClient) UpdateActionVersion(ctx context.Context, v ActionVersionQualifier, enabled bool) (*ActionVersion, error) {
	query := `
	  mutation UpdateActionVersion($version: ActionVersionQualifier!, $enabled: Boolean!) {
	    updateActionVersion(version: $version, enabled: $enabled) {
		versionMajor
		versionMinor
		validFrom
		validTo
		runtime
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"version": v,
		"enabled": enabled,
	}})
	if err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		return nil, resp.Errors
	}

	type response struct {
		UpdateActionVersion *ActionVersion
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.UpdateActionVersion, nil
}
