package image

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestContextArchiveDockerignoreAndReproducibility(t *testing.T) {
	for _, specific := range []bool{false, true} {
		t.Run(map[bool]string{false: "root ignore", true: "Dockerfile specific ignore"}[specific], func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range map[string]string{"Dockerfile": "FROM scratch\n", ".dockerignore": "*.secret\nexcluded\n!excluded/keep\n", "password.secret": "never upload", "included": "hello", "excluded/drop": "drop", "excluded/keep": "keep"} {
				path := filepath.Join(dir, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
				require.NoError(t, os.WriteFile(path, []byte(body), 0600))
			}
			if specific {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile.dockerignore"), []byte("*.secret\nincluded\n"), 0600))
			}
			build := func() []byte {
				f, err := os.CreateTemp(t.TempDir(), "archive")
				require.NoError(t, err)
				defer f.Close()
				_, _, err = compressedArchive(f, func(w io.Writer) error { return contextArchive(context.Background(), dir, "Dockerfile", w) })
				require.NoError(t, err)
				data, err := os.ReadFile(f.Name())
				require.NoError(t, err)
				return data
			}
			first := build()
			require.Equal(t, first, build())
			gz, err := gzip.NewReader(bytes.NewReader(first))
			require.NoError(t, err)
			defer gz.Close()
			tr := tar.NewReader(gz)
			files := map[string]string{}
			for {
				h, err := tr.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				data, err := io.ReadAll(tr)
				require.NoError(t, err)
				files[h.Name] = string(data)
			}
			require.NotContains(t, files, "password.secret")
			if specific {
				require.NotContains(t, files, "included")
				require.Contains(t, files, "excluded/drop")
			} else {
				require.Equal(t, "hello", files["included"])
				require.NotContains(t, files, "excluded/drop")
			}
			require.Equal(t, "keep", files["excluded/keep"])
		})
	}
}

func TestContextArchiveRejectsEscapingSymlink(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Symlink("../private", filepath.Join(dir, "escape")))
	err := contextArchive(context.Background(), dir, "Dockerfile", io.Discard)
	require.ErrorContains(t, err, "escapes")
}

func TestDockerMetadata(t *testing.T) {
	for _, test := range []struct {
		name, architecture, health string
		wantError                  bool
	}{{"linux metadata", "amd64", "", false}, {"disabled healthcheck", "amd64", "NONE", false}, {"wrong architecture", "arm64", "", true}, {"unsupported healthcheck", "amd64", "CMD", true}} {
		t.Run(test.name, func(t *testing.T) {
			var d dockerInspection
			require.NoError(t, json.Unmarshal([]byte(`{"Os":"linux","Architecture":"`+test.architecture+`","Config":{"Entrypoint":["/app"],"Cmd":["serve"],"Env":["A=old","A=new"],"WorkingDir":"/work","User":"1000:1000","StopSignal":"SIGQUIT","Labels":{"hello":"world"},"Volumes":{"/data":{}},"ExposedPorts":{"8080/tcp":{}},"Healthcheck":{"Test":["`+test.health+`"]}}}`), &d))
			if test.health == "" {
				d.Config.Healthcheck = nil
			}
			config, err := d.config()
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"/app"}, config.Entrypoint)
			require.Equal(t, []string{"serve"}, config.Cmd)
			require.Equal(t, "new", config.Env["A"])
			require.Equal(t, "/work", config.WorkingDir)
			require.Equal(t, "1000:1000", config.User)
			require.Equal(t, "SIGQUIT", config.StopSignal)
			require.Equal(t, []string{"/data"}, config.Volumes)
			require.Equal(t, []string{"8080/tcp"}, config.ExposedPorts)
		})
	}
}

func TestDockerExportPinsPlatformAndCleansContainer(t *testing.T) {
	for _, failExport := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failed export"}[failExport], func(t *testing.T) {
			var calls [][]string
			imageID := "sha256:" + strings.Repeat("a", 64)
			container := strings.Repeat("b", 64)
			run := func(_ context.Context, out, _ io.Writer, args ...string) error {
				calls = append(calls, args)
				switch args[0] {
				case "buildx":
					require.Equal(t, []string{"buildx", "build", "--platform", "linux/amd64", "--load"}, args[:5])
					return os.WriteFile(args[6], []byte(imageID), 0600)
				case "image":
					require.Equal(t, imageID, args[2])
					_, err := io.WriteString(out, `[{"Os":"linux","Architecture":"amd64","Config":{}}]`)
					return err
				case "create":
					require.Equal(t, imageID, args[len(args)-1])
					_, err := io.WriteString(out, container)
					return err
				case "export":
					if failExport {
						return errors.New("failed export")
					}
					_, err := io.WriteString(out, "tar bytes")
					return err
				case "rm":
					require.Equal(t, []string{"rm", "--force", "--volumes", container}, args)
					return nil
				default:
					t.Fatalf("unexpected Docker operation %v", args)
					return nil
				}
			}
			_, err := exportDocker(context.Background(), run, ".", "Dockerfile", "", nil, t.TempDir(), io.Discard, io.Discard)
			if failExport {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, "rm", calls[len(calls)-1][0])
		})
	}
}

func TestUploadHeadersAndWriteOnceRetry(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("X-Inngest-Env"))
		require.Equal(t, "*", r.Header.Get("If-None-Match"))
		require.EqualValues(t, 7, r.ContentLength)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "archive", string(body))
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	defer server.Close()
	c := client{key: "api-secret", environment: "workspace", http: server.Client()}
	f, err := os.CreateTemp(t.TempDir(), "archive")
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("archive")
	require.NoError(t, err)
	require.NoError(t, c.upload(context.Background(), server.URL, map[string]string{"If-None-Match": "*"}, f))
}

func TestParseTag(t *testing.T) {
	for _, input := range []string{"inngest/base:latest", "../x:a", "app:tag:bad", "team/a/b", "app@sha256:abc", "app:UPPER"} {
		_, _, err := parseTag(input)
		require.Error(t, err, input)
	}
	name, tag, err := parseTag("app")
	require.NoError(t, err)
	require.Equal(t, "app", name)
	require.Equal(t, "latest", tag)
}

func TestMultipartUploadRangesAndRetry(t *testing.T) {
	calls := []string{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("X-Inngest-Env"))
		require.EqualValues(t, 3, r.ContentLength)
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		calls = append(calls, string(data))
		if len(calls) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	c := client{key: "api-secret", environment: "workspace", http: server.Client()}
	file, err := os.CreateTemp(t.TempDir(), "archive")
	require.NoError(t, err)
	defer file.Close()
	_, err = file.WriteString("abcdef")
	require.NoError(t, err)
	grant := &apiv2.ImageUploadGrant{Parts: []*apiv2.ImageUploadPart{{Number: 1, SizeBytes: 3, Url: server.URL}, {Number: 2, Offset: 3, SizeBytes: 3, Url: server.URL}}}
	require.NoError(t, c.uploadGrant(context.Background(), grant, file))
	require.Equal(t, []string{"abc", "abc", "def"}, calls)
	grant.AlreadyUploaded = true
	require.NoError(t, c.uploadGrant(context.Background(), grant, file))
	require.Len(t, calls, 3)
	grant.AlreadyUploaded = false
	grant.Parts[1].Offset = 2
	require.ErrorContains(t, c.uploadGrant(context.Background(), grant, file), "byte ranges")
	require.Len(t, calls, 3)
}
