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
	"time"

	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// TestDeployApp runs `inngest deploy` with fake docker against a fake API,
// logged in through the saved login that `inngest login` writes.
func TestDeployApp(t *testing.T) {
	// the SHA256 of "image bytes", which fake docker saves.
	const digest = "de7030234493a8bea844dbe1d8676e68a2c1a4b014c721f0425a22b6df66faec"
	machineSize := `{"errors":[{"code":"machine_size_not_allowed","message":"deploy machine size is not allowed: spec of 2 vCPU and 4096 MiB exceeds this account's limit of 1 vCPU and 1024 MiB per instance; lower spec in inngest.toml or upgrade your plan"}]}`

	for _, tc := range []struct {
		name string
		toml string
		// args follow `inngest deploy`.
		args []string
		// wantDocker is the docker build args after --platform and --load,
		// with the local tag as <ref>.
		wantDocker []string
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
			wantDocker: []string{"-t", "<ref>", "-f", "./cmd/all-in-one/Dockerfile", "."},
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
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"connect":{"replicas":2}}}`,
			deploy:     `{"data":{"id":"d1","status":"ready","artifact_digest":"abc"}}`,
			output:     []string{"Deploy d1 is ready.  Artifact digest abc"},
		},
		{
			name:       "app by name with docker build args",
			toml:       "[apps.web]\nserve = {}\n[apps.inngest]\nconnect = {}",
			args:       []string{"inngest", "--", "--build-arg", "FDB_VERSION=7.3.68", "--target", "worker"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", ".", "--build-arg", "FDB_VERSION=7.3.68", "--target", "worker"},
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"connect":{}}}`,
			deploy:     `{"data":{"id":"d1","status":"ready","artifact_digest":"abc"}}`,
		},
		{
			name:       "only app with docker build args",
			toml:       "[apps.inngest]\nconnect = {}",
			args:       []string{"--", "--build-arg", "VERSION=1"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", ".", "--build-arg", "VERSION=1"},
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"connect":{}}}`,
			deploy:     `{"data":{"id":"d1","status":"ready","artifact_digest":"abc"}}`,
		},
		{
			name:         "spec over the plan limit",
			toml:         "[apps.inngest]\nconnect = {}\nspec = { cpu = 2, memory = 4096 }",
			wantDocker:   []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantCreate:   `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"spec":{"cpu":2,"memory":4096},"connect":{}}}`,
			createStatus: http.StatusForbidden,
			createBody:   machineSize,
			err:          "create deploy: POST /deploys/inngest: 403 Forbidden: machine_size_not_allowed: deploy machine size is not allowed: spec of 2 vCPU and 4096 MiB exceeds this account's limit of 1 vCPU and 1024 MiB per instance; lower spec in inngest.toml or upgrade your plan",
		},
		{
			name:       "failed deploy",
			toml:       "[apps.inngest]\nserve = {}",
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantCreate: `{"name":"inngest","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"` + digest + `","config":{"serve":{}}}`,
			deploy:     `{"data":{"id":"d1","status":"failed","error":"Image build failed or timed out"}}`,
			err:        "deploy d1 failed: Image build failed or timed out",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dockerDir := fakeDocker(t)

			var uploaded, committed []byte
			mux := http.NewServeMux()
			mux.HandleFunc("POST /v2/deploys/inngest", func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
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
			login(t, srv.URL)

			project := t.TempDir()
			t.Chdir(project)
			require.NoError(t, os.WriteFile(appsFile, []byte(tc.toml), 0o600))
			require.NoError(t, os.MkdirAll("cmd/all-in-one", 0o755))
			for _, dockerfile := range []string{"Dockerfile", "cmd/all-in-one/Dockerfile"} {
				require.NoError(t, os.WriteFile(dockerfile, []byte("FROM scratch\n"), 0o600))
			}

			var stdout bytes.Buffer
			root := &cli.Command{
				Name:      "inngest",
				Reader:    strings.NewReader(""),
				Writer:    &stdout,
				ErrWriter: io.Discard,
				Commands:  []*cli.Command{DeployCommand()},
			}
			err := root.Run(context.Background(), append([]string{"inngest", "deploy"}, tc.args...))

			args, readErr := os.ReadFile(filepath.Join(dockerDir, "args"))
			require.NoError(t, readErr)
			gotDocker := strings.Fields(string(args))
			// a tag per run keeps concurrent deploys of an app apart.
			require.Regexp(t, `^inngest-deploy:inngest-[0-9a-f]{16}$`, gotDocker[5])
			saved, readErr := os.ReadFile(filepath.Join(dockerDir, "saved"))
			require.NoError(t, readErr)
			require.Equal(t, gotDocker[5], string(saved))
			gotDocker[5] = "<ref>"
			require.Equal(t, append([]string{"build", "--platform", "linux/amd64", "--load"}, tc.wantDocker...), gotDocker)

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
			if !strings.Contains(tc.toml, "serve") {
				require.NotContains(t, stdout.String(), "sync")
			}
		})
	}
}

// login saves a login for the API at url, as `inngest login` does.
func login(t *testing.T, url string) {
	t.Helper()
	t.Setenv("INNGEST_API_HOST", url)
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	store, err := cliauth.NewStore()
	require.NoError(t, err)
	expiry := time.Now().Add(time.Hour)
	require.NoError(t, store.Save(cliauth.Metadata{
		Issuer:           url,
		Resource:         cliauth.Resource(url),
		SessionID:        "session",
		SessionExpiresAt: expiry,
	}, cliauth.Credential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       expiry,
	}, true))
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
