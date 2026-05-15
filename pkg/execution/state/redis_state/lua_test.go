package redis_state

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestLuaEndsWith(t *testing.T) {
	runScript := func(t *testing.T, rc rueidis.Client, key string) bool {
		val, err := scripts["test/ends_with"].Exec(
			t.Context(),
			rc,
			[]string{key},
			[]string{},
		).AsInt64()
		require.NoError(t, err)

		switch val {
		case 1:
			return true
		default:
			return false
		}
	}

	_, rc := initRedis(t)
	defer rc.Close()

	defaultShard := shardFromClient("default", rc)
	kg := defaultShard.Client().kg

	t.Run("with empty string", func(t *testing.T) {
		key := kg.BacklogSet("")
		require.Contains(t, key, ":-")
		require.False(t, runScript(t, rc, key))
	})

	t.Run("with non empty string", func(t *testing.T) {
		key := kg.BacklogSet("hello")
		require.NotContains(t, key, ":-")
		require.True(t, runScript(t, rc, key))
	})
}

func TestLuaScriptSnapshots(t *testing.T) {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}

	scripts := make(map[string]string)

	var readRedisScripts func(path string, entries []fs.DirEntry)

	readRedisScripts = func(path string, entries []fs.DirEntry) {
		for _, e := range entries {
			// NOTE: When using embed go always uses forward slashes as a path
			// prefix. filepath.Join uses OS-specific prefixes which fails on
			// windows, so we construct the path using Sprintf for all platforms
			if e.IsDir() {
				entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
				readRedisScripts(path+"/"+e.Name(), entries)
				continue
			}

			byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
			if err != nil {
				panic(fmt.Errorf("error reading redis lua script: %w", err))
			}

			name := path + "/" + e.Name()
			name = strings.TrimPrefix(name, "lua/")
			name = strings.TrimSuffix(name, ".lua")
			val := string(byt)

			// Add any includes.
			items := include.FindAllStringSubmatch(val, -1)
			if len(items) > 0 {
				// Replace each include
				for _, include := range items {
					byt, err = embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
					if err != nil {
						panic(fmt.Errorf("error reading redis lua include: %w", err))
					}
					val = strings.ReplaceAll(val, include[0], string(byt))
				}
			}

			scripts[name] = val
		}
	}

	readRedisScripts("lua", entries)

	// Test each script
	for scriptName, rawContent := range scripts {
		t.Run(scriptName, func(t *testing.T) {
			// Process the script

			// Read expected snapshot from fixture file
			snapshotPath := filepath.Join("testdata", "snapshots", scriptName+".lua")
			// Generate snapshot file if it doesn't exist
			err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755)
			require.NoError(t, err)

			err = os.WriteFile(snapshotPath, []byte(rawContent), 0o644)
			require.NoError(t, err)

			t.Logf("Generated snapshot for %s at %s", scriptName, snapshotPath)
		})
	}
}
