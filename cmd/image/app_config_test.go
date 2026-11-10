package imagecmd

import (
	"strings"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestParseApps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		toml  string
		err   string
		check func(t *testing.T, apps map[string]*app)
	}{
		{
			name: "minimal serve",
			toml: `[apps.inngest]
serve = { port = 8080, path = "/api/inngest" }`,
			check: func(t *testing.T, apps map[string]*app) {
				a := apps["inngest"]
				require.Equal(t, uint32(8080), *a.Serve.Port)
				require.Equal(t, "/api/inngest", *a.Serve.Path)
				require.Nil(t, a.Connect)
				require.Nil(t, a.Spec)
				require.Nil(t, a.ServeConfig)
				require.Nil(t, a.Description)
				require.Equal(t, "./Dockerfile", a.dockerfile())
			},
		},
		{
			name: "minimal connect",
			toml: `[apps.worker]
connect = { replicas = 2 }`,
			check: func(t *testing.T, apps map[string]*app) {
				a := apps["worker"]
				require.Equal(t, uint32(2), *a.Connect.Replicas)
				require.Nil(t, a.Serve)
			},
		},
		{
			name: "empty serve table is serve with server defaults",
			toml: `[apps.web]
serve = {}`,
			check: func(t *testing.T, apps map[string]*app) {
				a := apps["web"]
				require.NotNil(t, a.Serve)
				require.Nil(t, a.Serve.Port)
				require.Nil(t, a.Serve.Path)
			},
		},
		{
			name: "every setting",
			toml: `[apps.inngest]
connect = { replicas = 10 }
description = "Serves basic functions for Inngest itself"
dockerfile = "./cmd/all-in-one/Dockerfile"
spec = { memory = 4096, cpu = 2 }`,
			check: func(t *testing.T, apps map[string]*app) {
				a := apps["inngest"]
				require.Equal(t, "./cmd/all-in-one/Dockerfile", a.dockerfile())
				require.Equal(t, uint32(4096), *a.Spec.Memory)
				require.Equal(t, uint32(2), *a.Spec.CPU)
				require.Equal(t, "Serves basic functions for Inngest itself", *a.Description)
			},
		},
		{
			name: "section tables",
			toml: `[apps.web.serve]
port = 3000
[apps.web.serve_config]
idle = "0s"
lifetime = "1h30m"
[apps.worker.connect]`,
			check: func(t *testing.T, apps map[string]*app) {
				require.Equal(t, uint32(3000), *apps["web"].Serve.Port)
				require.Equal(t, "1h30m", *apps["web"].ServeConfig.Lifetime)
				require.NotNil(t, apps["worker"].Connect)
			},
		},
		{
			name: "example config",
			toml: exampleConfig,
			check: func(t *testing.T, apps map[string]*app) {
				require.Len(t, apps, 1)
				require.NotNil(t, apps["my-app"].Serve)
			},
		},
		{
			name: "example config with every commented setting",
			toml: uncommentSettings(exampleConfig, "connect = "),
			check: func(t *testing.T, apps map[string]*app) {
				a := apps["my-app"]
				require.NotNil(t, a.Serve)
				require.NotNil(t, a.ServeConfig)
				require.NotNil(t, a.Spec)
				require.NotNil(t, a.Description)
				require.Equal(t, "./Dockerfile", a.Dockerfile)
			},
		},
		{
			name: "no apps",
			toml: ``,
			check: func(t *testing.T, apps map[string]*app) {
				require.Empty(t, apps)
			},
		},
		{
			name: "both serve and connect",
			toml: `[apps.inngest]
connect = { replicas = 1 }
serve = { port = 8080 }`,
			err: "[apps.inngest]: set exactly one of connect or serve",
		},
		{
			name: "neither serve nor connect",
			toml: `[apps.inngest]
dockerfile = "./Dockerfile"`,
			err: "[apps.inngest]: set exactly one of connect or serve",
		},
		{
			name: "empty app",
			toml: `[apps.inngest]`,
			err:  "set exactly one of connect or serve",
		},
		{
			name: "bad idle duration",
			toml: `[apps.web]
serve = {}
serve_config = { idle = "15" }`,
			err: `serve_config.idle "15" must be a nonnegative duration`,
		},
		{
			name: "negative lifetime",
			toml: `[apps.web]
serve = {}
serve_config = { lifetime = "-1m" }`,
			err: `serve_config.lifetime "-1m" must be a nonnegative duration`,
		},
		{
			name: "duration as a number",
			toml: `[apps.web]
serve = {}
serve_config = { idle = 15 }`,
			err: "invalid inngest.toml",
		},
		{
			name: "serve_config with connect",
			toml: `[apps.worker]
connect = {}
serve_config = { multiplex = 1 }`,
			err: "serve_config requires serve",
		},
		{
			name: "zero replicas",
			toml: `[apps.worker]
connect = { replicas = 0 }`,
			err: "connect.replicas must be at least 1",
		},
		{
			name: "negative replicas",
			toml: `[apps.worker]
connect = { replicas = -1 }`,
			err: "invalid inngest.toml",
		},
		{
			name: "port out of range",
			toml: `[apps.web]
serve = { port = 65536 }`,
			err: "serve.port 65536 must be between 1 and 65535",
		},
		{
			name: "relative path",
			toml: `[apps.web]
serve = { path = "api/inngest" }`,
			err: `serve.path "api/inngest" must be an absolute path`,
		},
		{
			name: "multiplex too high",
			toml: `[apps.web]
serve = {}
serve_config = { multiplex = 256 }`,
			err: "serve_config.multiplex 256 must be between 0 and 255",
		},
		{
			name: "misspelled setting",
			toml: `[apps.web]
serve = {}
serve_conifg = { multiplex = 1 }`,
			err: "serve_conifg",
		},
		{
			name: "misspelled section",
			toml: `[app.web]
serve = { port = 8080 }`,
			err: "app.web.serve.port",
		},
		{
			name: "invalid app name",
			toml: `[apps.Web]
serve = {}`,
			err: "[apps.Web]: the app name must be",
		},
		{
			name: "reserved app name",
			toml: `[apps.default]
serve = {}`,
			err: "[apps.default]: the app name must be",
		},
		{
			name: "subcommand as app name",
			toml: `[apps.example-config]
serve = {}`,
			err: "[apps.example-config]: the app name must be",
		},
		{
			name: "help as app name",
			toml: `[apps.h]
serve = {}`,
			err: "[apps.h]: the app name must be",
		},
		{
			name: "invalid toml",
			toml: `[apps.web`,
			err:  "invalid inngest.toml",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apps, err := parseApps(strings.NewReader(tc.toml))
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			tc.check(t, apps)
		})
	}
}

// uncommentSettings removes the leading "# " from the example's commented
// settings, except the setting that starts with skip.
func uncommentSettings(config, skip string) string {
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		setting := strings.TrimPrefix(line, "# ")
		if setting != line && strings.Contains(setting, " = ") && !strings.HasPrefix(setting, skip) {
			lines[i] = setting
		}
	}
	return strings.Join(lines, "\n")
}

func TestSelectApps(t *testing.T) {
	one := map[string]*app{"web": {}}
	two := map[string]*app{"web": {}, "worker": {}}
	for _, tc := range []struct {
		name string
		apps map[string]*app
		args []string
		want []string
		err  string
	}{
		{name: "only app", apps: one, want: []string{"web"}},
		{name: "named app", apps: two, args: []string{"worker"}, want: []string{"worker"}},
		{name: "named apps in the order given", apps: two, args: []string{"worker", "web"}, want: []string{"worker", "web"}},
		{name: "unknown app", apps: two, args: []string{"api"}, err: `no app "api" in inngest.toml; apps: web, worker`},
		{name: "unknown app after a known app", apps: two, args: []string{"web", "api"}, err: `no app "api" in inngest.toml; apps: web, worker`},
		{name: "unknown app with one app", apps: one, args: []string{"api"}, err: `no app "api" in inngest.toml; apps: web`},
		{name: "app named twice", apps: two, args: []string{"web", "worker", "web"}, err: `app "web" is named more than once`},
		{name: "several apps without a name", apps: two, err: "name the apps to deploy with `inngest deploy <app>...`: web, worker"},
		{name: "no apps", apps: nil, err: "inngest.toml has no apps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			names, err := selectApps(tc.apps, tc.args)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, names)
		})
	}
}

// TestDeployConfig checks the JSON that CreateDeploy receives for each
// config.  settings that the file omits must be absent, not zero, so that the
// server applies its defaults and plan limits.
func TestDeployConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		toml string
		want string
	}{
		{
			name: "minimal serve",
			toml: `serve = { port = 8080, path = "/api/inngest" }`,
			want: `{"serve":{"port":8080,"path":"/api/inngest"}}`,
		},
		{
			name: "minimal connect",
			toml: `connect = { replicas = 2 }`,
			want: `{"connect":{"replicas":2}}`,
		},
		{
			name: "connect without replicas",
			toml: `connect = {}`,
			want: `{"connect":{}}`,
		},
		{
			name: "serve without port or path",
			toml: `serve = {}`,
			want: `{"serve":{}}`,
		},
		{
			name: "every serve setting",
			toml: `serve = { port = 8080, path = "/api/inngest" }
serve_config = { multiplex = 10, max_requests = 50, idle = "15s", lifetime = "15m" }
description = "Serves basic functions for Inngest itself"
dockerfile = "./cmd/all-in-one/Dockerfile"
spec = { memory = 2048, cpu = 1 }`,
			want: `{
				"description":"Serves basic functions for Inngest itself",
				"spec":{"memory":2048,"cpu":1},
				"serve":{"port":8080,"path":"/api/inngest"},
				"serveConfig":{"multiplex":10,"maxRequests":50,"idle":"15s","lifetime":"15m"}
			}`,
		},
		{
			name: "explicit zeros are sent",
			toml: `serve = {}
serve_config = { multiplex = 0, max_requests = 0, idle = "0s" }`,
			want: `{"serve":{},"serveConfig":{"multiplex":0,"maxRequests":0,"idle":"0s"}}`,
		},
		{
			name: "partial spec",
			toml: `connect = { replicas = 3 }
spec = { cpu = 2 }`,
			want: `{"spec":{"cpu":2},"connect":{"replicas":3}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apps, err := parseApps(strings.NewReader("[apps.inngest]\n" + tc.toml))
			require.NoError(t, err)

			got, err := protojson.Marshal(apps["inngest"].deployConfig())
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(got))

			// the server decodes the same message from the request body.
			decoded := &apiv2.DeployConfig{}
			require.NoError(t, protojson.Unmarshal(got, decoded))
			require.JSONEq(t, tc.want, protojson.Format(decoded))
		})
	}
}
