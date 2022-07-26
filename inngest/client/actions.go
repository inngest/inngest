package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/internal/cuedefs"
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
	CreatedAt   *time.Time
	ImageSha256 *string
}

func (av ActionVersion) VersionMajor() (uint, error) {
	return av.Version.Major, nil
}
func (av ActionVersion) VersionMinor() (uint, error) {
	return av.Version.Minor, nil
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

	if !c.isCloudAPI {
		query = `
			query ($dsn: String!, $major: Int, $minor: Int) {
				actionVersion(query: {
					dsn: $dsn
					versionMajor: $major
					versionMinor: $minor
				}) {
					dsn
					name
					versionMajor
					versionMinor
					validFrom
					validTo
					config
				}
			}`
	}

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"dsn":   dsn,
		"major": major,
		"minor": minor,
	}})
	if err != nil {
		return nil, err
	}

	// TODO(df) make sure this works
	if !c.isCloudAPI {
		type response struct {
			ActionVersion *ActionVersion
		}
		data := &response{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return nil, fmt.Errorf("error unmarshalling response: %w", err)
		}
		parsed, err := cuedefs.ParseAction(data.ActionVersion.Config)
		if err != nil {
			return nil, err
		}
		av := data.ActionVersion
		av.ActionVersion = *parsed
		return av, nil
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

	if !c.isCloudAPI {
		query = `
			mutation CreateAction($config: String!) {
				createActionVersion(input: {
					config: $config
				}) {
					dsn name
				}
			}`
	}

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"config": input,
	}})
	if err != nil {
		return nil, err
	}

	if !c.isCloudAPI {
		type response struct {
			CreateActionVersion *ActionVersion
		}
		data := &response{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return nil, fmt.Errorf("error unmarshalling response: %w", err)
		}
		return &Action{
			DSN:  data.CreateActionVersion.DSN,
			Name: data.CreateActionVersion.Name,
		}, nil
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
	fields := "dsn name versionMajor versionMinor validFrom validTo config"
	query := fmt.Sprintf(`
	  mutation UpdateActionVersion($version: ActionVersionQualifier!, $enabled: Boolean!) {
	    updateActionVersion(version: $version, enabled: $enabled) {
			%s imageSha256
            }
          }`, fields)
	variables := map[string]interface{}{
		"version": v,
		"enabled": enabled,
	}

	if !c.isCloudAPI {
		query = fmt.Sprintf(`
			mutation CreateAction($dsn: String!, $versionMajor: Int!, $versionMinor: Int!, $enabled: Boolean!) {
				updateActionVersion(input: {
					dsn: $dsn, versionMajor: $versionMajor, versionMinor: $versionMinor, enabled: $enabled
				}) {
					%s
				}
			}`, fields)
		variables = map[string]interface{}{
			"dsn":          v.DSN,
			"versionMajor": v.VersionMajor,
			"versionMinor": v.VersionMinor,
			"enabled":      enabled,
		}
	}

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: variables})
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
