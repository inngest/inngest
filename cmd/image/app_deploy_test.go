package imagecmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	ready := `{"data":{"id":"{id}","status":"ready","artifact_digest":"abc"}}`
	twoApps := "[apps.web]\nserve = {}\n[apps.worker]\nconnect = {}"

	for _, tc := range []struct {
		name string
		toml string
		// args follow `inngest deploy`.
		args []string
		// wantDocker is the docker build args after --platform and --load,
		// with the local tag as <ref>.  every build has the same args.
		wantDocker []string
		// wantBuilt is the apps that docker builds, in order.
		wantBuilt []string
		// wantConfig is the CreateDeploy config of each app.
		wantConfig map[string]string
		// rejected is an app whose CreateDeploy fails with machineSize.
		rejected string
		// deploy is the GetDeploy response, with the deploy ID as {id}.
		deploy string
		// wantCalls is the API calls in order.  the API names deploys d1,
		// d2, and so on.
		wantCalls []string
		err       string
		output    []string
	}{
		{
			name: "serve",
			toml: `[apps.inngest]
serve = { port = 8080, path = "/api/inngest" }
serve_config = { idle = "15s" }
dockerfile = "./cmd/all-in-one/Dockerfile"`,
			wantDocker: []string{"-t", "<ref>", "-f", "./cmd/all-in-one/Dockerfile", "."},
			wantBuilt:  []string{"inngest"},
			wantConfig: map[string]string{"inngest": `{"serve":{"port":8080,"path":"/api/inngest"},"serveConfig":{"idle":"15s"}}`},
			deploy:     ready,
			wantCalls:  []string{"create inngest", "upload d1", "commit inngest d1", "get d1"},
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
			wantBuilt:  []string{"inngest"},
			wantConfig: map[string]string{"inngest": `{"connect":{"replicas":2}}`},
			deploy:     ready,
			wantCalls:  []string{"create inngest", "upload d1", "commit inngest d1", "get d1"},
			output:     []string{"Deploy d1 is ready.  Artifact digest abc"},
		},
		{
			name:       "app by name with docker build args",
			toml:       "[apps.web]\nserve = {}\n[apps.inngest]\nconnect = {}",
			args:       []string{"inngest", "--", "--build-arg", "FDB_VERSION=7.3.68", "--target", "worker"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", ".", "--build-arg", "FDB_VERSION=7.3.68", "--target", "worker"},
			wantBuilt:  []string{"inngest"},
			wantConfig: map[string]string{"inngest": `{"connect":{}}`},
			deploy:     ready,
			wantCalls:  []string{"create inngest", "upload d1", "commit inngest d1", "get d1"},
		},
		{
			name:       "only app with docker build args",
			toml:       "[apps.inngest]\nconnect = {}",
			args:       []string{"--", "--build-arg", "VERSION=1"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", ".", "--build-arg", "VERSION=1"},
			wantBuilt:  []string{"inngest"},
			wantConfig: map[string]string{"inngest": `{"connect":{}}`},
			deploy:     ready,
			wantCalls:  []string{"create inngest", "upload d1", "commit inngest d1", "get d1"},
		},
		{
			name:       "spec over the plan limit",
			toml:       "[apps.inngest]\nconnect = {}\nspec = { cpu = 2, memory = 4096 }",
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantBuilt:  []string{"inngest"},
			wantConfig: map[string]string{"inngest": `{"spec":{"cpu":2,"memory":4096},"connect":{}}`},
			rejected:   "inngest",
			// a rejected deploy uploads nothing.
			wantCalls: []string{"create inngest"},
			err:       "create deploy: POST /deploys/inngest: 403 Forbidden: machine_size_not_allowed: deploy machine size is not allowed: spec of 2 vCPU and 4096 MiB exceeds this account's limit of 1 vCPU and 1024 MiB per instance; lower spec in inngest.toml or upgrade your plan",
		},
		{
			name:       "failed deploy",
			toml:       "[apps.inngest]\nserve = {}",
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantBuilt:  []string{"inngest"},
			wantConfig: map[string]string{"inngest": `{"serve":{}}`},
			deploy:     `{"data":{"id":"{id}","status":"failed","error":"Image build failed or timed out"}}`,
			wantCalls:  []string{"create inngest", "upload d1", "commit inngest d1", "get d1"},
			err:        "deploy d1 failed: Image build failed or timed out",
		},
		{
			name:       "several apps in the order given",
			toml:       twoApps,
			args:       []string{"worker", "web"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantBuilt:  []string{"worker", "web"},
			wantConfig: map[string]string{"web": `{"serve":{}}`, "worker": `{"connect":{}}`},
			deploy:     ready,
			wantCalls: []string{
				"create worker", "upload d1", "commit worker d1", "get d1",
				"create web", "upload d2", "commit web d2", "get d2",
			},
			output: []string{"Summary:\n  worker: deploy d1 is ready\n  web: deploy d2 is ready\n"},
		},
		{
			name: "unknown app after a known app",
			toml: twoApps,
			args: []string{"web", "wrker"},
			// nothing is built or deployed.
			err: `no app "wrker" in inngest.toml; apps: web, worker`,
		},
		{
			name:       "failed app does not stop the rest",
			toml:       "[apps.web]\nserve = {}\nspec = { cpu = 2, memory = 4096 }\n[apps.worker]\nconnect = {}",
			args:       []string{"web", "worker"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", "."},
			wantBuilt:  []string{"web", "worker"},
			wantConfig: map[string]string{"web": `{"spec":{"cpu":2,"memory":4096},"serve":{}}`, "worker": `{"connect":{}}`},
			rejected:   "web",
			deploy:     ready,
			wantCalls:  []string{"create web", "create worker", "upload d1", "commit worker d1", "get d1"},
			err:        "1 of 2 apps failed to deploy: web",
			output: []string{
				"Summary:\n  web: failed: create deploy: POST /deploys/web: 403 Forbidden: machine_size_not_allowed: deploy machine size is not allowed",
				"\n  worker: deploy d1 is ready\n",
			},
		},
		{
			name:       "several apps with docker build args",
			toml:       twoApps,
			args:       []string{"web", "worker", "--", "--build-arg", "VERSION=1"},
			wantDocker: []string{"-t", "<ref>", "-f", "./Dockerfile", ".", "--build-arg", "VERSION=1"},
			wantBuilt:  []string{"web", "worker"},
			wantConfig: map[string]string{"web": `{"serve":{}}`, "worker": `{"connect":{}}`},
			deploy:     ready,
			wantCalls: []string{
				"create web", "upload d1", "commit web d1", "get d1",
				"create worker", "upload d2", "commit worker d2", "get d2",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dockerDir := fakeDocker(t)

			var (
				mu    sync.Mutex
				calls []string
				ids   int
			)
			record := func(call string) {
				mu.Lock()
				defer mu.Unlock()
				calls = append(calls, call)
			}
			mux := http.NewServeMux()
			mux.HandleFunc("POST /v2/deploys/{name}", func(w http.ResponseWriter, r *http.Request) {
				name := r.PathValue("name")
				record("create " + name)
				require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
				body, _ := io.ReadAll(r.Body)
				require.JSONEq(t, `{"name":"`+name+`","tag":"latest","architecture":"amd64","uploadSizeBytes":"11","uploadSha256":"`+digest+`","config":`+tc.wantConfig[name]+`}`, string(body))
				if name == tc.rejected {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(machineSize))
					return
				}
				mu.Lock()
				ids++
				id := fmt.Sprintf("d%d", ids)
				mu.Unlock()
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"data":{"id":"` + id + `","status":"pending"},"upload_url":"http://` + r.Host + `/upload/` + id + `","upload_headers":{"x-amz-checksum-sha256":"abc"}}`))
			})
			mux.HandleFunc("PUT /upload/{id}", func(w http.ResponseWriter, r *http.Request) {
				record("upload " + r.PathValue("id"))
				require.Equal(t, "abc", r.Header.Get("x-amz-checksum-sha256"))
				body, _ := io.ReadAll(r.Body)
				require.Equal(t, "image bytes", string(body))
			})
			mux.HandleFunc("POST /v2/deploys/{name}/{id}/commit", func(w http.ResponseWriter, r *http.Request) {
				name, id := r.PathValue("name"), r.PathValue("id")
				record("commit " + name + " " + id)
				body, _ := io.ReadAll(r.Body)
				require.JSONEq(t, `{"deployId":"`+id+`","name":"`+name+`"}`, string(body))
				_, _ = w.Write([]byte(`{"data":{"id":"` + id + `","status":"uploaded"}}`))
			})
			mux.HandleFunc("GET /v2/deploys/{id}", func(w http.ResponseWriter, r *http.Request) {
				record("get " + r.PathValue("id"))
				_, _ = w.Write([]byte(strings.ReplaceAll(tc.deploy, "{id}", r.PathValue("id"))))
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
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}

			// each app has its own local tag, which docker save gets.
			builds, _ := os.ReadFile(filepath.Join(dockerDir, "args"))
			saved, _ := os.ReadFile(filepath.Join(dockerDir, "saved"))
			builtLines := strings.Split(strings.TrimSuffix(string(builds), "\n"), "\n")
			savedLines := strings.Split(strings.TrimSuffix(string(saved), "\n"), "\n")
			if len(tc.wantBuilt) == 0 {
				require.Empty(t, builds, "nothing is built")
			} else {
				require.Len(t, builtLines, len(tc.wantBuilt))
				require.Len(t, savedLines, len(tc.wantBuilt))
			}
			for i, app := range tc.wantBuilt {
				gotDocker := strings.Fields(builtLines[i])
				require.Regexp(t, `^inngest-deploy:`+regexp.QuoteMeta(app)+`-[0-9a-f]{16}$`, gotDocker[5])
				require.Equal(t, gotDocker[5], savedLines[i])
				gotDocker[5] = "<ref>"
				require.Equal(t, append([]string{"build", "--platform", "linux/amd64", "--load"}, tc.wantDocker...), gotDocker)
			}

			mu.Lock()
			require.Equal(t, tc.wantCalls, calls)
			mu.Unlock()
			for _, line := range tc.output {
				require.Contains(t, stdout.String(), line)
			}
			if len(tc.wantBuilt) == 1 {
				require.NotContains(t, stdout.String(), "Summary:", "one app has no summary")
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
