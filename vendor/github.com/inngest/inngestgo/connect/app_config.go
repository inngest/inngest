package connect

import (
	"encoding/json"
	"fmt"

	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/inngest/inngestgo/pkg/types"
)

type (
	// stepURLsByFn lists the step URLs of each function in one app
	stepURLsByFn map[types.FnSlug][]string
	// stepURLsByApp lists stepURLsByFn for each app
	stepURLsByApp map[types.AppSlug]stepURLsByFn
)

// appConfigurations builds the handshake app list.  it also returns the step
// URLs of each function by app, for the debug log.
func appConfigurations(apps []ConnectApp) ([]*connectproto.AppConfiguration, stepURLsByApp, error) {
	configs := make([]*connectproto.AppConfiguration, len(apps))
	appSlugs := make(stepURLsByApp, len(apps))
	for i, app := range apps {
		functionSlugs := make(stepURLsByFn, len(app.Functions))
		for _, function := range app.Functions {
			stepUrls := make([]string, 0, len(function.Steps))
			for _, step := range function.Steps {
				url, _ := step.Runtime["url"].(string)
				stepUrls = append(stepUrls, url)
			}
			functionSlugs[types.FnSlug(function.Slug)] = stepUrls
		}
		appSlugs[types.AppSlug(app.AppName)] = functionSlugs

		marshaledFns, err := json.Marshal(app.Functions)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to serialize connect config: %w", err)
		}

		configs[i] = &connectproto.AppConfiguration{
			AppName:    app.AppName,
			AppVersion: app.AppVersion,
			Functions:  marshaledFns,
		}
	}

	return configs, appSlugs, nil
}
