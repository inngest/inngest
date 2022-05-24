package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/internal/cuedefs"
)

type Action struct {
	DSN     string
	Name    string
	Tagline string
	Latest  *ActionVersion
	Version *ActionVersion
}

// ActionVersion represents the data received from GQL for an action version. This
// is a superset of an inngest.ActionVersion;  it contains the configuration for
// the version plus additional account-specific metadata.
type ActionVersion struct {
	inngest.ActionVersion

	Name        string
	DSN         string
	Config      string
	ValidFrom   *time.Time
	ValidTo     *time.Time
	ImageSha256 *string
}

func (c httpClient) Action(ctx context.Context, dsn string, v *inngest.VersionInfo) (*ActionVersion, error) {
	var major, minor *uint
	if v != nil {
		major = &v.Major
		minor = &v.Minor
	}

	query := `
	  query ($dsn: String!, $major: Int, $minor: Int) {
	    action(dsn: $dsn) {
	      dsn name tagline
	      version(major: $major, minor: $minor) {
	        dsn
		name
		validFrom
		validTo
		imageSha256
		config
	      }
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"dsn":   dsn,
		"major": major,
		"minor": minor,
	}})
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

	av, err := cuedefs.ParseAction(data.Action.Version.Config)
	if err != nil {
		return nil, err
	}
	data.Action.Version.ActionVersion = *av
	return data.Action.Version, nil
}

func (c httpClient) Actions(ctx context.Context, includePublic bool) ([]*Action, error) {
	query := `
	  query ($filter: ActionFilter) {
	    actions(filter: $filter) {
	      dsn name tagline
	      latest {
	        dsn
		name
		versionMajor
		versionMinor
		validFrom
		validTo
		runtime
		imageSha256
		config
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
	VersionMajor uint   `json:"versionMajor"`
	VersionMinor uint   `json:"versionMinor"`
}

func (c httpClient) UpdateActionVersion(ctx context.Context, v ActionVersionQualifier, enabled bool) (*ActionVersion, error) {
	query := `
	  mutation UpdateActionVersion($version: ActionVersionQualifier!, $enabled: Boolean!) {
	    updateActionVersion(version: $version, enabled: $enabled) {
		versionMajor
		versionMinor
		validFrom
		validTo
		config
		name
		dsn
		imageSha256
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
