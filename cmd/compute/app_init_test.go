package computecmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteAppsFile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dir     string
		answers string
		// want is the file, without its first comment line.
		want string
		// asked is text the questionnaire printed, such as a retry.
		asked []string
		err   string
	}{
		{
			name:    "defaults",
			dir:     "My Project",
			answers: "\n\n\n\n\n",
			want: `[apps.my-project]
serve = { port = 8080, path = "/api/inngest" }
dockerfile = "./Dockerfile"
`,
		},
		{
			name:    "connect",
			dir:     "worker",
			answers: "jobs.v2\nconnect\n3\n./build/Dockerfile\n",
			want: `[apps."jobs.v2"]
connect = { replicas = 3 }
dockerfile = "./build/Dockerfile"
`,
		},
		{
			name:    "invalid answers are asked again",
			dir:     "web",
			answers: "Web App\nweb\nhttp\nserve\n0\n3000\napi\n/inngest\n\n",
			want: `[apps.web]
serve = { port = 3000, path = "/inngest" }
dockerfile = "./Dockerfile"
`,
			asked: []string{
				"use 1 to 63 lowercase letters",
				"answer serve or connect",
				"use a number from 1 to 65535",
				"use an absolute path such as /api/inngest",
			},
		},
		{
			name:    "numbers with leading zeros",
			dir:     "web",
			answers: "\nserve\n08080\n\n\n",
			want: `[apps.web]
serve = { port = 8080, path = "/api/inngest" }
dockerfile = "./Dockerfile"
`,
		},
		{
			name:    "subcommand as app name is asked again",
			dir:     "web",
			answers: "example-config\nworker\nconnect\n007\n\n",
			want: `[apps.worker]
connect = { replicas = 7 }
dockerfile = "./Dockerfile"
`,
			asked: []string{`and not "default", "example-config", "help" or "h"`},
		},
		{
			name:    "directory name that is not an app name",
			dir:     "default",
			answers: "\nconnect\n\n\n",
			want: `[apps.my-app]
connect = { replicas = 1 }
dockerfile = "./Dockerfile"
`,
		},
		{
			name:    "stdin ends before the last answer",
			dir:     "web",
			answers: "\nserve\n",
			err:     "inngest.toml was not written: unexpected EOF",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), tc.dir)
			require.NoError(t, os.Mkdir(dir, 0o755))
			t.Chdir(dir)

			var stdout bytes.Buffer
			err := writeAppsFile(strings.NewReader(tc.answers), &stdout)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				require.NoFileExists(t, appsFile)
				return
			}
			require.NoError(t, err)

			written, err := os.ReadFile(appsFile)
			require.NoError(t, err)
			_, body, _ := strings.Cut(string(written), "\n")
			require.Equal(t, tc.want, body)
			for _, text := range tc.asked {
				require.Contains(t, stdout.String(), text)
			}

			_, err = parseApps(bytes.NewReader(written))
			require.NoError(t, err)

			// a second run does not replace the file.
			err = writeAppsFile(strings.NewReader("\n\n\n\n\n"), &stdout)
			require.ErrorIs(t, err, os.ErrExist)
		})
	}
}

func TestDefaultAppName(t *testing.T) {
	for dir, want := range map[string]string{
		"api":                   "api",
		"My Project":            "my-project",
		"_internal.tools":       "internal.tools",
		"!!!":                   "my-app",
		"default":               "my-app",
		strings.Repeat("a", 70): strings.Repeat("a", 63),
	} {
		require.Equal(t, want, defaultAppName(dir), dir)
	}
}
