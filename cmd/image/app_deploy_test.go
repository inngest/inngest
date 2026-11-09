package imagecmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inngest/inngest/cmd/image/internal/docker"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestDeployApp(t *testing.T) {
	// the SHA256 of "image bytes", which fake docker saves.
	const digest = "de7030234493a8bea844dbe1d8676e68a2c1a4b014c721f0425a22b6df66faec"
	machineSize := `{"errors":[{"code":"machine_size_not_allowed","message":"deploy machine size is not allowed: spec of 2 vCPU and 4096 MiB exceeds this account's limit of 1 vCPU and 1024 MiB per instance; lower spec in inngest.toml or upgrade your plan"}]}`

	for _, tc := range []struct {
		name string
		toml string
		// wantCreate is the CreateDeploy body.
		wantCreate string
		// createStatus and createBody replace the CreateDeploy response.
		createStatus int
		createBody   string
		// deploy is the GetDeploy response.
		deploy string
		err    string
		output []string
	}{
		{
			name: "serve",
			toml: `[apps.inngest]
serve = { port = 8080, path = "/api/inngest" }
serve_config = { idle = "15s" }
dockerfile = "./cmd/all-in-one/Dockerfile"`,
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `",
				"config":{"serve":{"port":8080,"path":"/api/inngest"},"serveConfig":{"idle":"15s"}}}`,
			deploy: `{"data":{"id":"d1","status":"ready","artifact_digest":"abc"}}`,
			output: []string{
				"Building app inngest from ./cmd/all-in-one/Dockerfile...",
				"Uploading app inngest (11 bytes) for deploy d1...",
				"Deploy d1 is ready.  Artifact digest abc",
				"inngest/deploy.sync_failed",
			},
		},
		{
			name:       "connect",
			toml:       "[apps.inngest]\nconnect = { replicas = 2 }",
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"connect":{"replicas":2}}}`,
			deploy:     `{"data":{"id":"d1","status":"ready","artifact_digest":"abc"}}`,
			output:     []string{"Deploy d1 is ready.  Artifact digest abc"},
		},
		{
			name:         "spec over the plan limit",
			toml:         "[apps.inngest]\nconnect = {}\nspec = { cpu = 2, memory = 4096 }",
			wantCreate:   `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"spec":{"cpu":2,"memory":4096},"connect":{}}}`,
			createStatus: http.StatusForbidden,
			createBody:   machineSize,
			err:          "create deploy: POST /deploys/inngest: 403 Forbidden: machine_size_not_allowed: deploy machine size is not allowed: spec of 2 vCPU and 4096 MiB exceeds this account's limit of 1 vCPU and 1024 MiB per instance; lower spec in inngest.toml or upgrade your plan",
		},
		{
			name:       "failed deploy",
			toml:       "[apps.inngest]\nserve = {}",
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"serve":{}}}`,
			deploy:     `{"data":{"id":"d1","status":"failed","error":"Image build failed or timed out"}}`,
			err:        "deploy d1 failed: Image build failed or timed out",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := fakeDocker(t)
			apps, err := parseApps(strings.NewReader(tc.toml))
			require.NoError(t, err)

			var uploaded, committed []byte
			mux := http.NewServeMux()
			mux.HandleFunc("POST /v2/deploys/inngest", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				require.JSONEq(t, tc.wantCreate, string(body))
				if tc.createStatus != 0 {
					w.WriteHeader(tc.createStatus)
					_, _ = w.Write([]byte(tc.createBody))
					return
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"data":{"id":"d1","status":"pending"},"upload_url":"http://` + r.Host + `/upload","upload_headers":{"x-amz-checksum-sha256":"abc"}}`))
			})
			mux.HandleFunc("PUT /upload", func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "abc", r.Header.Get("x-amz-checksum-sha256"))
				uploaded, _ = io.ReadAll(r.Body)
			})
			mux.HandleFunc("POST /v2/deploys/inngest/d1/commit", func(w http.ResponseWriter, r *http.Request) {
				committed, _ = io.ReadAll(r.Body)
				_, _ = w.Write([]byte(`{"data":{"id":"d1","status":"uploaded"}}`))
			})
			mux.HandleFunc("GET /v2/deploys/d1", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.deploy))
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			api := &client{
				baseURL: srv.URL + "/v2",
				token:   func(context.Context) (string, error) { return "token", nil },
				http:    srv.Client(),
			}
			dc, err := docker.New(nil, io.Discard, io.Discard)
			require.NoError(t, err)

			var stdout bytes.Buffer
			err = deployApp(context.Background(), &stdout, dc, api, "inngest", apps["inngest"])

			args, readErr := os.ReadFile(filepath.Join(dir, "args"))
			require.NoError(t, readErr)
			require.Equal(t, []string{
				"build", "--platform", "linux/amd64", "--load",
				"-t", "inngest-deploy/inngest:latest", "-f", apps["inngest"].dockerfile(), ".",
			}, strings.Fields(string(args)))

			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				if tc.createStatus != 0 {
					require.Nil(t, uploaded, "a rejected deploy uploads nothing")
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, "image bytes", string(uploaded))
			require.JSONEq(t, `{"deployId":"d1","name":"inngest"}`, string(committed))
			for _, line := range tc.output {
				require.Contains(t, stdout.String(), line)
			}
			if apps["inngest"].Serve == nil {
				require.NotContains(t, stdout.String(), "sync")
			}
		})
	}
}

func TestDeployWithoutAppsFile(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout bytes.Buffer
	root := &cli.Command{
		Name:      "inngest",
		Writer:    &stdout,
		ErrWriter: io.Discard,
		Commands:  []*cli.Command{DeployCommand()},
	}

	err := root.Run(context.Background(), []string{"inngest", "deploy"})
	require.ErrorContains(t, err, "inngest.toml not found in the current directory")
	require.ErrorContains(t, err, "inngest deploy example-config > inngest.toml")

	// the example that the error points to is a valid file.
	require.NoError(t, root.Run(context.Background(), []string{"inngest", "deploy", "example-config"}))
	require.Equal(t, exampleConfig, stdout.String())
	require.NoError(t, os.WriteFile("inngest.toml", stdout.Bytes(), 0o600))
	err = root.Run(context.Background(), []string{"inngest", "deploy", "other"})
	require.EqualError(t, err, `no app "other" in inngest.toml; apps: my-app`)
}
