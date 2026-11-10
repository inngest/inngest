package imagecmd

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/pelletier/go-toml"
)

// appsFile is the file that `inngest deploy` reads.  it is parsed here, not
// in cmd/internal/config, until that loader reads TOML.
const appsFile = "inngest.toml"

const defaultDockerfile = "./Dockerfile"

// apps is the content of inngest.toml.
type apps struct {
	Apps map[string]*app `toml:"apps"`
}

// app is one [apps.<name>] section.  pointer fields are optional, and an
// omitted field is left out of the deploy request so that the server applies
// its default.
type app struct {
	Description *string          `toml:"description"`
	Dockerfile  string           `toml:"dockerfile"`
	Spec        *appSpec         `toml:"spec"`
	Connect     *appConnect      `toml:"connect"`
	Serve       *appServe        `toml:"serve"`
	ServeConfig *appServeRuntime `toml:"serve_config"`
}

type appSpec struct {
	// Memory is in MiB.
	Memory *uint32 `toml:"memory"`
	// CPU is in vCPUs.
	CPU *uint32 `toml:"cpu"`
}

type appConnect struct {
	Replicas *uint32 `toml:"replicas"`
}

type appServe struct {
	Port *uint32 `toml:"port"`
	Path *string `toml:"path"`
}

type appServeRuntime struct {
	Multiplex   *uint32 `toml:"multiplex"`
	MaxRequests *uint32 `toml:"max_requests"`
	Idle        *string `toml:"idle"`
	Lifetime    *string `toml:"lifetime"`
}

// parseApps reads inngest.toml and validates every app in it.  unknown keys
// are errors, so a misspelled setting is not silently ignored.
func parseApps(r io.Reader) (map[string]*app, error) {
	var file apps
	if err := toml.NewDecoder(r).Strict(true).Decode(&file); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", appsFile, err)
	}
	for _, name := range slices.Sorted(maps.Keys(file.Apps)) {
		if err := file.Apps[name].validate(name); err != nil {
			return nil, fmt.Errorf("invalid %s: [apps.%s]: %w", appsFile, name, err)
		}
	}
	return file.Apps, nil
}

// selectApps checks that every name is an app in the file, once, and returns
// the names.  no names selects the only app in the file.
func selectApps(apps map[string]*app, names []string) ([]string, error) {
	all := slices.Sorted(maps.Keys(apps))
	if len(all) == 0 {
		return nil, fmt.Errorf("%s has no apps; add an [apps.<name>] section, as in `inngest deploy example-config`", appsFile)
	}
	if len(names) == 0 {
		if len(all) > 1 {
			return nil, fmt.Errorf("%s has more than one app; name the apps to deploy with `inngest deploy <app>...`: %s", appsFile, strings.Join(all, ", "))
		}
		return all, nil
	}
	for i, name := range names {
		if _, ok := apps[name]; !ok {
			return nil, fmt.Errorf("no app %q in %s; apps: %s", name, appsFile, strings.Join(all, ", "))
		}
		if slices.Contains(names[:i], name) {
			return nil, fmt.Errorf("app %q is named more than once", name)
		}
	}
	return names, nil
}

// validate checks the settings that do not depend on the account.  the
// server checks spec sizes against the account's plan.
func (a *app) validate(name string) error {
	if a == nil {
		return errors.New("app is empty")
	}
	if !validAppName(name) {
		return errors.New("the app name must be " + appNameRules)
	}
	if (a.Connect == nil) == (a.Serve == nil) {
		return errors.New("set exactly one of connect or serve")
	}
	if a.Connect != nil {
		if a.Connect.Replicas != nil && *a.Connect.Replicas == 0 {
			return errors.New("connect.replicas must be at least 1")
		}
		if a.ServeConfig != nil {
			return errors.New("serve_config requires serve; remove it or use serve instead of connect")
		}
	}
	if a.Serve != nil {
		if p := a.Serve.Port; p != nil && (*p == 0 || *p > 65535) {
			return fmt.Errorf("serve.port %d must be between 1 and 65535", *p)
		}
		if p := a.Serve.Path; p != nil && (!strings.HasPrefix(*p, "/") || strings.ContainsAny(*p, "?#")) {
			return fmt.Errorf("serve.path %q must be an absolute path such as /api/inngest, without a query or fragment", *p)
		}
	}
	if rt := a.ServeConfig; rt != nil {
		if rt.Multiplex != nil && *rt.Multiplex > 255 {
			return fmt.Errorf("serve_config.multiplex %d must be between 0 and 255", *rt.Multiplex)
		}
		for _, setting := range []struct {
			key   string
			value *string
		}{{"idle", rt.Idle}, {"lifetime", rt.Lifetime}} {
			if setting.value == nil {
				continue
			}
			if d, err := time.ParseDuration(*setting.value); err != nil || d < 0 {
				return fmt.Errorf("serve_config.%s %q must be a nonnegative duration such as 15s or 15m", setting.key, *setting.value)
			}
		}
	}
	return nil
}

const appNameRules = `1 to 63 lowercase letters, digits, '.', '_' or '-', and not "default", "example-config", "help" or "h"`

// validAppName reports whether name follows appNameRules.  `inngest deploy
// <name>` runs the subcommand instead of deploying an app with its name.
func validAppName(name string) bool {
	switch name {
	case "default", "example-config", "help", "h":
		return false
	}
	return namePattern.MatchString(name)
}

func (a *app) dockerfile() string {
	if a.Dockerfile == "" {
		return defaultDockerfile
	}
	return a.Dockerfile
}

// deployConfig returns the config for CreateDeployRequest.  it sends only the
// settings in the file.
func (a *app) deployConfig() *apiv2.DeployConfig {
	config := &apiv2.DeployConfig{
		Description: a.Description,
	}
	if a.Spec != nil {
		config.Spec = &apiv2.DeploySpec{
			Memory: a.Spec.Memory,
			Cpu:    a.Spec.CPU,
		}
	}
	if a.Connect != nil {
		config.Mode = &apiv2.DeployConfig_Connect{
			Connect: &apiv2.DeployConnectConfig{
				Replicas: a.Connect.Replicas,
			},
		}
	}
	if a.Serve != nil {
		config.Mode = &apiv2.DeployConfig_Serve{
			Serve: &apiv2.DeployServeConfig{
				Port: a.Serve.Port,
				Path: a.Serve.Path,
			},
		}
	}
	if rt := a.ServeConfig; rt != nil {
		config.ServeConfig = &apiv2.DeployServeRuntimeConfig{
			Multiplex:   rt.Multiplex,
			MaxRequests: rt.MaxRequests,
			Idle:        rt.Idle,
			Lifetime:    rt.Lifetime,
		}
	}
	return config
}

// exampleConfig is printed by `inngest deploy example-config`.  it must
// parse, because agents copy it as is.
const exampleConfig = `# inngest.toml configures ` + "`inngest deploy`" + `.  Put it in the project root,
# which is the docker build context.
#
# Each [apps.<name>] section is one app.  <name> is 1 to 63 lowercase
# letters, digits, '.', '_' or '-'.  Deploy an app with
# ` + "`inngest deploy <name>`" + `, or with ` + "`inngest deploy`" + ` when the file has one app.
[apps.my-app]

# Set exactly one of serve or connect.
#
# serve runs the app as an HTTP server that Inngest calls.
#   port: the port the app listens on.  When omitted, Inngest uses a port
#         that the image exposes.
#   path: the path of the Inngest endpoint.  Defaults to /api/inngest.
serve = { port = 8080, path = "/api/inngest" }

# connect runs the app as workers that connect to Inngest.
#   replicas: the number of workers.  Defaults to 1.
# connect = { replicas = 2 }

# Optional, serve only.  Omitted settings use Inngest's defaults.
#   multiplex:    requests that run at once in one instance, 0 to 255.
#   max_requests: requests an instance serves before it drains.  0 is no limit.
#   idle:         stop an instance after no request runs for this long.
#   lifetime:     drain an instance after it runs for this long.
# Durations use Go syntax, such as 15s or 15m.
# serve_config = { multiplex = 10, max_requests = 50, idle = "15s", lifetime = "15m" }

# Optional.  A description of the app.
# description = "Serves my app's functions"

# Optional.  The Dockerfile to build, relative to this file.  Defaults to
# ./Dockerfile.  The image is built for linux/amd64.
# dockerfile = "./Dockerfile"

# Optional.  The machine for each instance: memory in MiB and cpu in vCPUs.
# Defaults to 1 vCPU and 2048 MiB, or your plan's limit when it is lower.
# spec = { memory = 2048, cpu = 1 }
`
